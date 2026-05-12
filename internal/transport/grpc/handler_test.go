package grpc

import (
	"context"
	"testing"

	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/middleware"
	"github.com/zhangpeihaoks/firefly/internal/transport"
	"google.golang.org/grpc/metadata"
)

// TestHandleAPI tests the Handle and Use registration APIs.
func TestHandleAPI(t *testing.T) {
	srv := NewServer()

	// Register a handler
	srv.Handle("/test.Service/Method", func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})

	srv.mu.RLock()
	defer srv.mu.RUnlock()

	if len(srv.handlers) != 1 {
		t.Errorf("expected 1 handler, got %d", len(srv.handlers))
	}
	if _, ok := srv.handlers["/test.Service/Method"]; !ok {
		t.Error("handler not registered for expected method")
	}
}

// TestUseAPI tests the Use method appends middleware.
func TestUseAPI(t *testing.T) {
	srv := NewServer()

	m1 := func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			return next(ctx, req)
		}
	}
	m2 := func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			return next(ctx, req)
		}
	}

	srv.Use(m1)
	if len(srv.ms) != 1 {
		t.Errorf("expected 1 middleware, got %d", len(srv.ms))
	}

	srv.Use(m2)
	if len(srv.ms) != 2 {
		t.Errorf("expected 2 middleware, got %d", len(srv.ms))
	}
}

// TestUseChained tests that middleware from Use is applied in order.
func TestUseChained(t *testing.T) {
	srv := NewServer()

	var order []int
	srv.Use(
		func(next Handler) Handler {
			return func(ctx context.Context, req any) (any, error) {
				order = append(order, 1)
				return next(ctx, req)
			}
		},
		func(next Handler) Handler {
			return func(ctx context.Context, req any) (any, error) {
				order = append(order, 2)
				return next(ctx, req)
			}
		},
	)

	// Build the chain and execute
	chain := middleware.Chain(srv.ms...)
	h := chain(func(ctx context.Context, req any) (any, error) {
		order = append(order, 3)
		return "done", nil
	})

	_, err := h(context.Background(), nil)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Errorf("expected order [1,2,3], got %v", order)
	}
}

// TestNewGRPCContext tests transport context creation.
func TestNewGRPCContext(t *testing.T) {
	md := metadata.New(map[string]string{
		"authorization": "Bearer token",
		"user-id":       "123",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	ctx = NewGRPCContext(ctx, "/test.Service/Method")

	tr := transport.FromContext(ctx)
	if tr == nil {
		t.Fatal("transport.FromContext returned nil")
	}
	if tr.Kind() != transport.KindGRPC {
		t.Errorf("expected KindGRPC, got %s", tr.Kind())
	}
	if tr.Operation() != "/test.Service/Method" {
		t.Errorf("expected Operation /test.Service/Method, got %s", tr.Operation())
	}
	if tr.Endpoint() != "/test.Service/Method" {
		t.Errorf("expected Endpoint, got %s", tr.Endpoint())
	}
}

// TestGRPCHeader tests the grpcHeader implementation.
func TestGRPCHeader(t *testing.T) {
	md := metadata.New(map[string]string{
		"key": "value",
	})
	h := &grpcHeader{md: md}

	if h.Get("key") != "value" {
		t.Errorf("expected 'value', got %q", h.Get("key"))
	}
	if h.Get("missing") != "" {
		t.Errorf("expected empty for missing key, got %q", h.Get("missing"))
	}

	h.Set("newkey", "newvalue")
	if h.Get("newkey") != "newvalue" {
		t.Errorf("expected 'newvalue' after Set, got %q", h.Get("newkey"))
	}

	keys := h.Keys()
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

// TestToGRPCError tests error conversion.
func TestToGRPCError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		if err := toGRPCError(nil); err != nil {
			t.Errorf("expected nil for nil input, got %v", err)
		}
	})

	t.Run("firefly error", func(t *testing.T) {
		ferr := errors.New(404, "NOT_FOUND", "resource missing")
		gerr := toGRPCError(ferr)
		if gerr == nil {
			t.Fatal("expected non-nil gRPC error")
		}
	})

	t.Run("standard error", func(t *testing.T) {
		gerr := toGRPCError(errors.New(500, "ERR", "test"))
		if gerr == nil {
			t.Fatal("expected non-nil gRPC error")
		}
	})
}

// TestGRPCTransporterInterface tests transporter satisfies interface.
func TestGRPCTransporterInterface(t *testing.T) {
	tr := &grpcTransporter{method: "/test/Method", md: metadata.New(nil)}
	var _ transport.Transporter = tr

	if tr.PathParams() != nil {
		t.Error("PathParams should be nil for gRPC")
	}
	if tr.QueryParams() != nil {
		t.Error("QueryParams should be nil for gRPC")
	}
}
