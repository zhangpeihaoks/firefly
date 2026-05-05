// Package grpc provides gRPC server implementation for the Firefly framework.
// It implements the transport.Server interface using gRPC.
package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/log"
	"github.com/zhangpeihaoks/firefly/internal/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Server is the gRPC server implementation.
// It implements transport.Server and transport.Endpointer interfaces.
type Server struct {
	*grpc.Server
	lis            net.Listener
	once           sync.Once
	endpoint       *url.URL
	err            error
	network        string
	address        string
	timeout        time.Duration
	ms             []middleware.Middleware
	log            *slog.Logger
	health         *health.Server
	tlsConf        *tls.Config
	unaryInts      []grpc.UnaryServerInterceptor
	streamInts     []grpc.StreamServerInterceptor
	maxRecvMsgSize int
	maxSendMsgSize int
}

// ServerOption is a function that configures the gRPC server.
type ServerOption func(*Server)

// NewServer creates a new gRPC server with the given options.
func NewServer(opts ...ServerOption) *Server {
	// Create server with default values
	s := &Server{
		network: "tcp",
		address: ":9090",
		timeout: 30 * time.Second,
		log:     log.L(),
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	// Build gRPC server options
	var grpcOpts []grpc.ServerOption

	// Apply TLS if configured
	if s.tlsConf != nil {
		grpcOpts = append(grpcOpts, grpc.Creds(credentials.NewTLS(s.tlsConf)))
	} else {
		// Default to insecure credentials for non-TLS servers
		grpcOpts = append(grpcOpts, grpc.Creds(insecure.NewCredentials()))
	}

	// Apply max message size configuration
	if s.maxRecvMsgSize > 0 {
		grpcOpts = append(grpcOpts, grpc.MaxRecvMsgSize(s.maxRecvMsgSize))
	}
	if s.maxSendMsgSize > 0 {
		grpcOpts = append(grpcOpts, grpc.MaxSendMsgSize(s.maxSendMsgSize))
	}

	// Build interceptor chain from middleware
	chainUnary := s.chainUnaryInterceptors()
	if len(chainUnary) > 0 {
		grpcOpts = append(grpcOpts, grpc.ChainUnaryInterceptor(chainUnary...))
	}

	chainStream := s.chainStreamInterceptors()
	if len(chainStream) > 0 {
		grpcOpts = append(grpcOpts, grpc.ChainStreamInterceptor(chainStream...))
	}

	// Create gRPC server
	s.Server = grpc.NewServer(grpcOpts...)

	// Setup health check service if enabled
	if s.health != nil {
		healthpb.RegisterHealthServer(s.Server, s.health)
	}

	return s
}

// chainUnaryInterceptors builds the chain of unary interceptors.
// It combines custom interceptors with middleware-based interceptors.
func (s *Server) chainUnaryInterceptors() []grpc.UnaryServerInterceptor {
	var interceptors []grpc.UnaryServerInterceptor

	// Add recovery interceptor first to catch panics
	interceptors = append(interceptors, s.recoveryUnaryInterceptor())

	// Add logging interceptor
	interceptors = append(interceptors, s.loggingUnaryInterceptor())

	// Add custom unary interceptors
	interceptors = append(interceptors, s.unaryInts...)

	// Add middleware-based interceptor if middleware is configured
	if len(s.ms) > 0 {
		interceptors = append(interceptors, s.middlewareUnaryInterceptor())
	}

	return interceptors
}

// chainStreamInterceptors builds the chain of stream interceptors.
// It combines custom interceptors with middleware-based interceptors.
func (s *Server) chainStreamInterceptors() []grpc.StreamServerInterceptor {
	var interceptors []grpc.StreamServerInterceptor

	// Add recovery interceptor first to catch panics
	interceptors = append(interceptors, s.recoveryStreamInterceptor())

	// Add logging interceptor
	interceptors = append(interceptors, s.loggingStreamInterceptor())

	// Add custom stream interceptors
	interceptors = append(interceptors, s.streamInts...)

	return interceptors
}

// recoveryUnaryInterceptor returns a unary interceptor that recovers from panics.
func (s *Server) recoveryUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic
				s.log.Error("panic recovered in gRPC unary handler",
					"method", info.FullMethod,
					"panic", r,
				)
				// Convert panic to gRPC error
				err = errors.New(errors.CodeInternal, "INTERNAL_ERROR", fmt.Sprintf("Internal server error: %v", r))
			}
		}()
		return handler(ctx, req)
	}
}

// recoveryStreamInterceptor returns a stream interceptor that recovers from panics.
func (s *Server) recoveryStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if r := recover(); r != nil {
				// Log the panic
				s.log.Error("panic recovered in gRPC stream handler",
					"method", info.FullMethod,
					"panic", r,
				)
				// Convert panic to gRPC error
				err = errors.New(errors.CodeInternal, "INTERNAL_ERROR", fmt.Sprintf("Internal server error: %v", r))
			}
		}()
		return handler(srv, ss)
	}
}

// loggingUnaryInterceptor returns a unary interceptor that logs requests.
func (s *Server) loggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Log request start
		s.log.Debug("gRPC unary request started",
			"method", info.FullMethod,
		)

		// Call handler
		resp, err := handler(ctx, req)

		// Log request completion
		duration := time.Since(start)
		if err != nil {
			s.log.Warn("gRPC unary request failed",
				"method", info.FullMethod,
				"duration", duration,
				"error", err,
			)
		} else {
			s.log.Debug("gRPC unary request completed",
				"method", info.FullMethod,
				"duration", duration,
			)
		}

		return resp, err
	}
}

// loggingStreamInterceptor returns a stream interceptor that logs requests.
func (s *Server) loggingStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		// Log request start
		s.log.Debug("gRPC stream request started",
			"method", info.FullMethod,
		)

		// Call handler
		err := handler(srv, ss)

		// Log request completion
		duration := time.Since(start)
		if err != nil {
			s.log.Warn("gRPC stream request failed",
				"method", info.FullMethod,
				"duration", duration,
				"error", err,
			)
		} else {
			s.log.Debug("gRPC stream request completed",
				"method", info.FullMethod,
				"duration", duration,
			)
		}

		return err
	}
}

// middlewareUnaryInterceptor returns a unary interceptor that applies middleware.
func (s *Server) middlewareUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Build middleware chain
		chain := middleware.Chain(s.ms...)

		// Create handler from middleware chain
		h := chain(func(ctx context.Context, r any) (any, error) {
			return handler(ctx, r)
		})

		// Execute middleware chain
		return h(ctx, req)
	}
}

// Start starts the gRPC server.
// It implements transport.Server interface.
func (s *Server) Start(ctx context.Context) error {
	s.once.Do(func() {
		// Create listener
		lis, err := net.Listen(s.network, s.address)
		if err != nil {
			s.err = err
			return
		}
		s.lis = lis

		// Build endpoint URL
		s.endpoint = &url.URL{
			Scheme: "grpc",
			Host:   lis.Addr().String(),
		}

		// Log server start
		s.log.Info("gRPC server starting",
			"address", s.address,
			"network", s.network,
		)

		// Start server in goroutine
		go func() {
			if err := s.Server.Serve(s.lis); err != nil && err != grpc.ErrServerStopped {
				s.log.Error("gRPC server error", "error", err)
			}
		}()
	})

	return s.err
}

// Stop stops the gRPC server gracefully.
// It implements transport.Server interface.
func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("gRPC server stopping")

	// Set health status to NOT_SERVING before shutdown
	if s.health != nil {
		s.health.Shutdown()
	}

	// Graceful stop with timeout
	done := make(chan struct{})
	go func() {
		s.Server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		s.log.Info("gRPC server stopped")
		return nil
	case <-ctx.Done():
		// Force stop if timeout
		s.Server.Stop()
		s.log.Warn("gRPC server force stopped due to timeout")
		return ctx.Err()
	}
}

// Endpoint returns the server endpoint URL.
// It implements transport.Endpointer interface.
func (s *Server) Endpoint() (*url.URL, error) {
	if s.endpoint == nil {
		return nil, fmt.Errorf("server not started")
	}
	return s.endpoint, nil
}

// RegisterService registers a gRPC service.
func (s *Server) RegisterService(desc *grpc.ServiceDesc, impl any) {
	s.Server.RegisterService(desc, impl)
}

// Network sets the network type for the gRPC server.
func Network(network string) ServerOption {
	return func(s *Server) {
		s.network = network
	}
}

// Address sets the listening address for the gRPC server.
func Address(addr string) ServerOption {
	return func(s *Server) {
		s.address = addr
	}
}

// Timeout sets the timeout for the gRPC server.
func Timeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.timeout = timeout
	}
}

// Middleware sets the middleware for the gRPC server.
func Middleware(m ...middleware.Middleware) ServerOption {
	return func(s *Server) {
		s.ms = m
	}
}

// Logger sets the logger for the gRPC server.
func Logger(logger *slog.Logger) ServerOption {
	return func(s *Server) {
		s.log = logger
	}
}

// TLSConfig sets the TLS configuration for the gRPC server.
func TLSConfig(cfg *tls.Config) ServerOption {
	return func(s *Server) {
		s.tlsConf = cfg
	}
}

// TLSConfigFromFile loads TLS configuration from files and sets it on the server.
// It reads certificate, key, and optional CA files from the filesystem.
func TLSConfigFromFile(certFile, keyFile string, caFile ...string) ServerOption {
	return func(s *Server) {
		var tlsCfg *TLSCfg
		if len(caFile) > 0 && caFile[0] != "" {
			tlsCfg = NewTLSCfg(
				WithCertFile(certFile),
				WithKeyFile(keyFile),
				WithCAFile(caFile[0]),
			)
		} else {
			tlsCfg = NewTLSCfg(
				WithCertFile(certFile),
				WithKeyFile(keyFile),
			)
		}

		cfg, err := LoadTLSCfg(tlsCfg)
		if err != nil {
			// Log the error but don't fail - TLS will be disabled
			s.log.Error("failed to load TLS config", "error", err)
			return
		}
		s.tlsConf = cfg
	}
}

// UnaryInterceptor sets the unary interceptors for the gRPC server.
// Multiple interceptors will be chained in the order they are provided.
func UnaryInterceptor(in ...grpc.UnaryServerInterceptor) ServerOption {
	return func(s *Server) {
		s.unaryInts = append(s.unaryInts, in...)
	}
}

// StreamInterceptor sets the stream interceptors for the gRPC server.
// Multiple interceptors will be chained in the order they are provided.
func StreamInterceptor(in ...grpc.StreamServerInterceptor) ServerOption {
	return func(s *Server) {
		s.streamInts = append(s.streamInts, in...)
	}
}

// MaxRecvMsgSize sets the maximum receive message size for the gRPC server.
// The default value is 4MB. Use this option to increase or decrease the limit.
func MaxRecvMsgSize(size int) ServerOption {
	return func(s *Server) {
		s.maxRecvMsgSize = size
	}
}

// MaxSendMsgSize sets the maximum send message size for the gRPC server.
// The default value is math.MaxInt32. Use this option to decrease the limit.
func MaxSendMsgSize(size int) ServerOption {
	return func(s *Server) {
		s.maxSendMsgSize = size
	}
}

// Health sets the health check server for the gRPC server.
// The health server implements the gRPC health checking protocol.
func Health(h *health.Server) ServerOption {
	return func(s *Server) {
		s.health = h
	}
}

// SetServingStatus sets the health status for a service.
// This is a convenience method that requires the health server to be configured.
func (s *Server) SetServingStatus(service string, status healthpb.HealthCheckResponse_ServingStatus) {
	if s.health != nil {
		s.health.SetServingStatus(service, status)
	}
}

// Shutdown shuts down the health server.
// This should be called before stopping the gRPC server.
func (s *Server) Shutdown() {
	if s.health != nil {
		s.health.Shutdown()
	}
}
