// Package grpc provides gRPC server implementation tests for the Firefly framework.
package grpc

import (
	"context"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/interop/grpc_testing"
)

// TestNewServer tests the creation of a gRPC server with various options.
func TestNewServer(t *testing.T) {
	tests := []struct {
		name   string
		opts   []ServerOption
		expect func(t *testing.T, s *Server)
	}{
		{
			name: "default server",
			opts: nil,
			expect: func(t *testing.T, s *Server) {
				if s.network != "tcp" {
					t.Errorf("expected network tcp, got %s", s.network)
				}
				if s.address != ":9090" {
					t.Errorf("expected address :9090, got %s", s.address)
				}
				if s.timeout != 30*time.Second {
					t.Errorf("expected timeout 30s, got %v", s.timeout)
				}
			},
		},
		{
			name: "custom address",
			opts: []ServerOption{Address(":50051")},
			expect: func(t *testing.T, s *Server) {
				if s.address != ":50051" {
					t.Errorf("expected address :50051, got %s", s.address)
				}
			},
		},
		{
			name: "custom network",
			opts: []ServerOption{Network("unix")},
			expect: func(t *testing.T, s *Server) {
				if s.network != "unix" {
					t.Errorf("expected network unix, got %s", s.network)
				}
			},
		},
		{
			name: "custom timeout",
			opts: []ServerOption{Timeout(10 * time.Second)},
			expect: func(t *testing.T, s *Server) {
				if s.timeout != 10*time.Second {
					t.Errorf("expected timeout 10s, got %v", s.timeout)
				}
			},
		},
		{
			name: "max recv message size",
			opts: []ServerOption{MaxRecvMsgSize(10 * 1024 * 1024)},
			expect: func(t *testing.T, s *Server) {
				if s.maxRecvMsgSize != 10*1024*1024 {
					t.Errorf("expected maxRecvMsgSize 10MB, got %d", s.maxRecvMsgSize)
				}
			},
		},
		{
			name: "max send message size",
			opts: []ServerOption{MaxSendMsgSize(20 * 1024 * 1024)},
			expect: func(t *testing.T, s *Server) {
				if s.maxSendMsgSize != 20*1024*1024 {
					t.Errorf("expected maxSendMsgSize 20MB, got %d", s.maxSendMsgSize)
				}
			},
		},
		{
			name: "health server",
			opts: []ServerOption{Health(health.NewServer())},
			expect: func(t *testing.T, s *Server) {
				if s.health == nil {
					t.Error("expected health server to be set")
				}
			},
		},
		{
			name: "unary interceptor",
			opts: []ServerOption{
				UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
					return handler(ctx, req)
				}),
			},
			expect: func(t *testing.T, s *Server) {
				if len(s.unaryInts) != 1 {
					t.Errorf("expected 1 unary interceptor, got %d", len(s.unaryInts))
				}
			},
		},
		{
			name: "multiple unary interceptors",
			opts: []ServerOption{
				UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
					return handler(ctx, req)
				}),
				UnaryInterceptor(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
					return handler(ctx, req)
				}),
			},
			expect: func(t *testing.T, s *Server) {
				if len(s.unaryInts) != 2 {
					t.Errorf("expected 2 unary interceptors, got %d", len(s.unaryInts))
				}
			},
		},
		{
			name: "stream interceptor",
			opts: []ServerOption{
				StreamInterceptor(func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
					return handler(srv, ss)
				}),
			},
			expect: func(t *testing.T, s *Server) {
				if len(s.streamInts) != 1 {
					t.Errorf("expected 1 stream interceptor, got %d", len(s.streamInts))
				}
			},
		},
		{
			name: "middleware",
			opts: []ServerOption{
				Middleware(func(h middleware.Handler) middleware.Handler {
					return func(ctx context.Context, req any) (any, error) {
						return h(ctx, req)
					}
				}),
			},
			expect: func(t *testing.T, s *Server) {
				if len(s.ms) != 1 {
					t.Errorf("expected 1 middleware, got %d", len(s.ms))
				}
			},
		},
		{
			name: "custom logger",
			opts: []ServerOption{Logger(slog.New(slog.NewTextHandler(os.Stdout, nil)))},
			expect: func(t *testing.T, s *Server) {
				if s.log == nil {
					t.Error("expected logger to be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewServer(tt.opts...)
			if s == nil {
				t.Fatal("expected server to be created")
			}
			tt.expect(t, s)
		})
	}
}

// TestServerStartStop tests starting and stopping the gRPC server.
func TestServerStartStop(t *testing.T) {
	// Find an available port
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	s := NewServer(Address(addr))

	// Start server
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Verify endpoint
	endpoint, err := s.Endpoint()
	if err != nil {
		t.Errorf("failed to get endpoint: %v", err)
	}
	if endpoint == nil {
		t.Error("expected endpoint to be set")
	}

	// Stop server
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.Stop(stopCtx); err != nil {
		t.Errorf("failed to stop server: %v", err)
	}
}

// TestServerDoubleStart tests that double start is handled correctly.
func TestServerDoubleStart(t *testing.T) {
	// Find an available port
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	s := NewServer(Address(addr))

	// Start server twice
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("first start failed: %v", err)
	}

	// Second start should be no-op (once.Do)
	if err := s.Start(ctx); err != nil {
		t.Errorf("second start should not fail: %v", err)
	}

	// Stop server
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = s.Stop(stopCtx)
}

// TestHealthCheck tests the health check functionality.
func TestHealthCheck(t *testing.T) {
	// Create health server
	healthServer := health.NewServer()

	// Find an available port
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	s := NewServer(
		Address(addr),
		Health(healthServer),
	)

	// Start server
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Set serving status
	s.SetServingStatus("test-service", healthpb.HealthCheckResponse_SERVING)

	// Verify health check server is configured
	if s.health == nil {
		t.Error("expected health server to be configured")
	}

	// Stop server
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.Stop(stopCtx); err != nil {
		t.Errorf("failed to stop server: %v", err)
	}
}

// TestMaxMessageSize tests the max message size configuration.
func TestMaxMessageSize(t *testing.T) {
	// Find an available port
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	// Test with max recv message size
	s := NewServer(
		Address(addr),
		MaxRecvMsgSize(10*1024*1024), // 10MB
		MaxSendMsgSize(20*1024*1024), // 20MB
	)

	if s.maxRecvMsgSize != 10*1024*1024 {
		t.Errorf("expected maxRecvMsgSize 10MB, got %d", s.maxRecvMsgSize)
	}
	if s.maxSendMsgSize != 20*1024*1024 {
		t.Errorf("expected maxSendMsgSize 20MB, got %d", s.maxSendMsgSize)
	}

	// Start and stop server to verify it works
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.Stop(stopCtx); err != nil {
		t.Errorf("failed to stop server: %v", err)
	}
}

// TestUnaryInterceptorChain tests the unary interceptor chain.
func TestUnaryInterceptorChain(t *testing.T) {
	var order []string

	// Create interceptors that record their execution order
	interceptor1 := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		order = append(order, "interceptor1-before")
		resp, err := handler(ctx, req)
		order = append(order, "interceptor1-after")
		return resp, err
	}

	interceptor2 := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		order = append(order, "interceptor2-before")
		resp, err := handler(ctx, req)
		order = append(order, "interceptor2-after")
		return resp, err
	}

	// Find an available port
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	s := NewServer(
		Address(addr),
		UnaryInterceptor(interceptor1),
		UnaryInterceptor(interceptor2),
	)

	// Verify interceptors are stored
	if len(s.unaryInts) != 2 {
		t.Errorf("expected 2 unary interceptors, got %d", len(s.unaryInts))
	}

	// Start and stop server
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = s.Stop(stopCtx)
}

// TestStreamInterceptorChain tests the stream interceptor chain.
func TestStreamInterceptorChain(t *testing.T) {
	// Create interceptors
	interceptor1 := func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, ss)
	}

	interceptor2 := func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, ss)
	}

	// Find an available port
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	s := NewServer(
		Address(addr),
		StreamInterceptor(interceptor1),
		StreamInterceptor(interceptor2),
	)

	// Verify interceptors are stored
	if len(s.streamInts) != 2 {
		t.Errorf("expected 2 stream interceptors, got %d", len(s.streamInts))
	}

	// Start and stop server
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = s.Stop(stopCtx)
}

// TestRegisterService tests service registration.
func TestRegisterService(t *testing.T) {
	// Find an available port
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	s := NewServer(Address(addr))

	// Register a test service
	s.RegisterService(&grpc.ServiceDesc{
		ServiceName: "test.TestService",
		HandlerType: (*grpc_testing.TestServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "EmptyCall",
				Handler:    nil,
			},
		},
	}, nil)

	// Start and stop server
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = s.Stop(stopCtx)
}

// TestEndpointBeforeStart tests that endpoint returns error before start.
func TestEndpointBeforeStart(t *testing.T) {
	s := NewServer()

	_, err := s.Endpoint()
	if err == nil {
		t.Error("expected error when getting endpoint before start")
	}
}

// TestGracefulShutdown tests the graceful shutdown behavior.
func TestGracefulShutdown(t *testing.T) {
	// Find an available port
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	s := NewServer(Address(addr))

	// Start server
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Stop with reasonable timeout
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- s.Stop(stopCtx)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("failed to stop server gracefully: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("server took too long to stop")
	}
}

// TestMiddlewareIntegration tests middleware integration with gRPC.
func TestMiddlewareIntegration(t *testing.T) {
	// Create a simple middleware that adds a value to context
	testMiddleware := func(h middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			// Middleware processing before handler
			return h(ctx, req)
		}
	}

	// Find an available port
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	s := NewServer(
		Address(addr),
		Middleware(testMiddleware),
	)

	// Verify middleware is set
	if len(s.ms) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(s.ms))
	}

	// Start and stop server
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = s.Stop(stopCtx)
}

// TestHealthShutdown tests health server shutdown.
func TestHealthShutdown(t *testing.T) {
	healthServer := health.NewServer()

	// Find an available port
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	s := NewServer(
		Address(addr),
		Health(healthServer),
	)

	// Start server
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Set serving status
	s.SetServingStatus("test-service", healthpb.HealthCheckResponse_SERVING)

	// Call shutdown
	s.Shutdown()

	// Stop server
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = s.Stop(stopCtx)
}
