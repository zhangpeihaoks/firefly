package http

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/transport"
)

func TestNewServer(t *testing.T) {
	// Test creating server with default options
	srv := NewServer()
	if srv == nil {
		t.Fatal("NewServer returned nil")
	}

	// Test server implements transport.Server interface
	var _ transport.Server = srv

	// Test default values
	if srv.address != ":8080" {
		t.Errorf("expected address :8080, got %s", srv.address)
	}
	if srv.network != "tcp" {
		t.Errorf("expected network tcp, got %s", srv.network)
	}
	if srv.timeout != 30*time.Second {
		t.Errorf("expected timeout 30s, got %v", srv.timeout)
	}
}

func TestServerOptions(t *testing.T) {
	// Test with custom options
	srv := NewServer(
		Address(":9090"),
		Network("tcp4"),
		Timeout(10*time.Second),
	)

	if srv.address != ":9090" {
		t.Errorf("expected address :9090, got %s", srv.address)
	}
	if srv.network != "tcp4" {
		t.Errorf("expected network tcp4, got %s", srv.network)
	}
	if srv.timeout != 10*time.Second {
		t.Errorf("expected timeout 10s, got %v", srv.timeout)
	}
}

func TestServerStartStop(t *testing.T) {
	ctx := context.Background()

	// Create server with random port
	srv := NewServer(
		Address(":0"), // Use port 0 for random available port
		Timeout(time.Second),
	)

	// Start server
	err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	err = srv.Stop(ctx)
	if err != nil {
		t.Fatalf("failed to stop server: %v", err)
	}
}

func TestServerRoute(t *testing.T) {
	srv := NewServer(Address(":0"))

	// Define a simple handler
	handler := func(ctx context.Context, req any) (any, error) {
		return map[string]string{"message": "hello"}, nil
	}

	// Register a route
	srv.Route(http.MethodGet, "/test", handler)

	// Test that route was registered
	// (This is a basic test - actual HTTP testing would require starting the server)
}

func TestDynamicRouteParameters(t *testing.T) {
	srv := NewServer(Address(":0"))

	// Test handler that extracts path parameters
	userHandler := func(ctx context.Context, req any) (any, error) {
		id, ok := GetPathParamInt(ctx, "id")
		if !ok {
			return nil, fmt.Errorf("id parameter not found")
		}
		return map[string]interface{}{
			"id":   id,
			"name": fmt.Sprintf("User %d", id),
		}, nil
	}

	// Test handler that extracts query parameters
	searchHandler := func(ctx context.Context, req any) (any, error) {
		query, _ := GetQueryParam(ctx, "q")
		page, _ := GetQueryParamInt(ctx, "page")
		return map[string]interface{}{
			"query": query,
			"page":  page,
		}, nil
	}

	// Register dynamic routes
	srv.Route(http.MethodGet, "/users/:id", userHandler)
	srv.Route(http.MethodGet, "/search", searchHandler)

	// Test that routes were registered
	// (Actual HTTP testing would require starting the server and making requests)
}

func TestServerGroup(t *testing.T) {
	srv := NewServer(Address(":0"))

	// Create a group
	api := srv.Group("/api")

	// Define a handler
	handler := func(ctx context.Context, req any) (any, error) {
		return map[string]string{"api": "v1"}, nil
	}

	// Register route in group
	api.Route(http.MethodGet, "/test", handler)

	// Test nested group
	v1 := api.Group("/v1")
	v1.Route(http.MethodGet, "/users", handler)
}

// =============================================================================
// Property-Based Tests
// =============================================================================

// Property 4: Server Configuration Correctness
// Validates: Requirement 7.2
//
// For any server configuration options, the server should have correct values.
func TestProperty4HTTPServerConfig(t *testing.T) {
	tests := []struct {
		name  string
		opts  []ServerOption
		check func(*Server) bool
	}{
		{
			name: "default values",
			opts: []ServerOption{},
			check: func(s *Server) bool {
				return s.address == ":8080" && s.network == "tcp" && s.timeout == 30*time.Second
			},
		},
		{
			name: "custom address",
			opts: []ServerOption{Address(":9090")},
			check: func(s *Server) bool {
				return s.address == ":9090"
			},
		},
		{
			name: "custom network",
			opts: []ServerOption{Network("tcp4")},
			check: func(s *Server) bool {
				return s.network == "tcp4"
			},
		},
		{
			name: "custom timeout",
			opts: []ServerOption{Timeout(60 * time.Second)},
			check: func(s *Server) bool {
				return s.timeout == 60*time.Second
			},
		},
		{
			name: "multiple options",
			opts: []ServerOption{
				Address(":8081"),
				Network("tcp6"),
				Timeout(45 * time.Second),
			},
			check: func(s *Server) bool {
				return s.address == ":8081" && s.network == "tcp6" && s.timeout == 45*time.Second
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(tt.opts...)
			if !tt.check(srv) {
				t.Error("server configuration not applied correctly")
			}
		})
	}
}
