package grpc

import (
	"testing"

	"github.com/zhangpeihaoks/firefly/internal/middleware"
)

// TestServerDefaultObservability verifies the default middleware chain is
// installed on gRPC servers without explicit middleware.
func TestServerDefaultObservability(t *testing.T) {
	srv := NewServer()

	if len(srv.ms) != len(middleware.DefaultObservabilityChain()) {
		t.Fatalf("expected %d default middleware, got %d",
			len(middleware.DefaultObservabilityChain()), len(srv.ms))
	}
}

// TestServerExplicitMiddlewareOverridesDefault verifies explicit middleware
// replaces the default chain.
func TestServerExplicitMiddlewareOverridesDefault(t *testing.T) {
	m := middleware.RequestID()
	srv := NewServer(Middleware(m))

	if len(srv.ms) != 1 {
		t.Fatalf("expected exactly 1 explicit middleware, got %d", len(srv.ms))
	}
}

// TestServerWithoutDefaultMiddleware verifies the opt-out switch.
func TestServerWithoutDefaultMiddleware(t *testing.T) {
	srv := NewServer(WithoutDefaultMiddleware())

	if len(srv.ms) != 0 {
		t.Fatalf("expected no middleware, got %d", len(srv.ms))
	}
}
