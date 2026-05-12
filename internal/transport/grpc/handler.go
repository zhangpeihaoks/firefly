package grpc

import (
	"context"
	"fmt"
	"sync"

	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/middleware"
	"github.com/zhangpeihaoks/firefly/internal/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Handler is a gRPC handler function with the unified Firefly handler signature.
type Handler = middleware.Handler

// Handle registers a unified handler for a gRPC method path.
// The method should be the full path, e.g., "/package.Service/Method".
//
// Example:
//
//	server.Handle("/user.UserService/GetUser", func(ctx context.Context, req any) (any, error) {
//	    return map[string]string{"name": "Alice"}, nil
//	})
func (s *Server) Handle(method string, handler Handler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.handlers == nil {
		s.handlers = make(map[string]Handler)
	}
	s.handlers[method] = handler
}

// Use appends middleware to the server's middleware chain.
func (s *Server) Use(ms ...middleware.Middleware) {
	s.ms = append(s.ms, ms...)
}

// handlerUnaryInterceptor returns a unary interceptor that dispatches
// registered handlers by method name and injects transport context.
func (s *Server) handlerUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		s.mu.RLock()
		h, exists := s.handlers[info.FullMethod]
		s.mu.RUnlock()

		if !exists {
			return handler(ctx, req)
		}

		// Inject gRPC transport context
		ctx = NewGRPCContext(ctx, info.FullMethod)

		// Build middleware chain
		chain := middleware.Chain(s.ms...)
		wrapped := chain(func(ctx context.Context, r any) (any, error) {
			return h(ctx, r)
		})

		resp, err := wrapped(ctx, req)
		if err != nil {
			return nil, toGRPCError(err)
		}
		return resp, nil
	}
}

// NewGRPCContext creates a context with gRPC transport info injected.
func NewGRPCContext(ctx context.Context, method string) context.Context {
	md, _ := metadata.FromIncomingContext(ctx)
	if md == nil {
		md = metadata.New(nil)
	}
	return transport.NewContext(ctx, &grpcTransporter{method: method, md: md})
}

// grpcTransporter implements transport.Transporter.
type grpcTransporter struct {
	method string
	md     metadata.MD
}

func (t *grpcTransporter) Kind() transport.Kind               { return transport.KindGRPC }
func (t *grpcTransporter) Endpoint() string                   { return t.method }
func (t *grpcTransporter) Operation() string                  { return t.method }
func (t *grpcTransporter) PathParams() map[string]string      { return nil }
func (t *grpcTransporter) QueryParams() map[string][]string   { return nil }
func (t *grpcTransporter) RequestHeader() transport.Header    { return &grpcHeader{md: t.md} }
func (t *grpcTransporter) ReplyHeader() transport.Header      { return &grpcHeader{md: metadata.New(nil)} }

// grpcHeader implements transport.Header over gRPC metadata.
type grpcHeader struct {
	md metadata.MD
	mu sync.RWMutex
}

func (h *grpcHeader) Get(key string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if vals := h.md[key]; len(vals) > 0 {
		return vals[0]
	}
	return ""
}

func (h *grpcHeader) Set(key string, value string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.md[key] = []string{value}
}

func (h *grpcHeader) Keys() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	keys := make([]string, 0, len(h.md))
	for k := range h.md {
		keys = append(keys, k)
	}
	return keys
}

// toGRPCError converts a Firefly error to a gRPC status error.
func toGRPCError(err error) error {
	if err == nil {
		return nil
	}
	if ferr, ok := err.(*errors.Error); ok {
		return status.New(codes.Code(errors.HTTPToGRPCCode(int(ferr.Code))), ferr.Message).Err()
	}
	return status.New(codes.Internal, fmt.Sprintf("internal error: %v", err)).Err()
}
