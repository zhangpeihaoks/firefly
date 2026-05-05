// Package integration provides end-to-end integration tests for the Firefly framework.
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/zhangpeihaoks/firefly/app"
	"github.com/zhangpeihaoks/firefly/internal/log"
	"github.com/zhangpeihaoks/firefly/internal/middleware"
	httpserver "github.com/zhangpeihaoks/firefly/internal/transport/http"
)

// TestHTTPServerCreation tests the HTTP server can be created and configured.
func TestHTTPServerCreation(t *testing.T) {
	// Initialize logger
	cleanup := log.New(&log.Config{
		FileName:   "test_server_creation.log",
		MaxSize:    10,
		MaxBackups: 1,
		MaxAge:     1,
		Level:      "debug",
		JSONFormat: false,
		Location:   true,
	})
	defer cleanup()

	// Test creating server with all options
	srv := httpserver.NewServer(
		httpserver.Address(":8099"),
		httpserver.Timeout(30*time.Second),
		httpserver.Middleware(
			middleware.Recovery(),
			middleware.Logging(),
		),
	)

	if srv == nil {
		t.Fatal("Expected server to be created, got nil")
	}

	// Test route registration - these should not panic
	srv.Route("GET", "/test", func(ctx context.Context, req any) (any, error) {
		return map[string]string{"status": "ok"}, nil
	})

	srv.Route("GET", "/users/:id", func(ctx context.Context, req any) (any, error) {
		return map[string]interface{}{"user_id": 1}, nil
	})

	srv.Route("POST", "/create", func(ctx context.Context, req any) (any, error) {
		return map[string]interface{}{"created": true}, nil
	})

	// Test route group
	api := srv.Group("/api/v1")
	api.Use(middleware.Recovery())
	api.Route("GET", "/status", func(ctx context.Context, req any) (any, error) {
		return map[string]string{"status": "operational"}, nil
	})

	// Test that Endpoint() returns error before start
	_, err := srv.Endpoint()
	if err == nil {
		t.Error("Expected error when getting endpoint before server starts")
	}

	// Start server
	ctx := context.Background()
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test that Endpoint() works after start
	endpoint, err := srv.Endpoint()
	if err != nil {
		t.Errorf("Expected no error after server starts, got: %v", err)
	}
	if endpoint == nil {
		t.Error("Expected endpoint to be non-nil")
	}

	// Stop server
	if err := srv.Stop(ctx); err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}
}

// TestAppWithServer tests the App can manage an HTTP server.
func TestAppWithServer(t *testing.T) {
	cleanup := log.New(&log.Config{
		FileName:   "test_app_with_server.log",
		MaxSize:    10,
		MaxBackups: 1,
		Level:      "debug",
	})
	defer cleanup()

	// Create HTTP server
	srv := httpserver.NewServer(
		httpserver.Address(":8102"),
		httpserver.Middleware(middleware.Recovery()),
	)

	srv.Route("GET", "/health", func(ctx context.Context, req any) (any, error) {
		return map[string]string{"status": "healthy"}, nil
	})

	// Create application
	application := app.New(
		app.Name("test-app"),
		app.Logger(log.L()),
		app.Server(srv),
		app.StopTimeout(2*time.Second),
	)

	// Verify app configuration
	if application.Name() != "test-app" {
		t.Errorf("Expected name 'test-app', got '%s'", application.Name())
	}

	// Verify metadata is empty by default (may be nil)
	if application.Metadata() != nil && len(application.Metadata()) != 0 {
		t.Error("Expected metadata to be empty")
	}
}

// TestMiddlewareChain tests that middleware chain works correctly.
func TestMiddlewareChain(t *testing.T) {
	cleanup := log.New(&log.Config{
		FileName:   "test_middleware_chain.log",
		MaxSize:    10,
		MaxBackups: 1,
		Level:      "debug",
	})
	defer cleanup()

	// Test Chain function
	recovery := middleware.Recovery()
	logging := middleware.Logging()

	chain := middleware.Chain(recovery, logging)

	if chain == nil {
		t.Fatal("Expected chain to be non-nil")
	}

	// Test that chain wraps a handler without panicking
	handler := func(ctx context.Context, req any) (any, error) {
		return map[string]string{"result": "success"}, nil
	}

	wrapped := chain(handler)

	if wrapped == nil {
		t.Error("Expected wrapped handler to be non-nil")
	}
}

// TestRouteGroup tests route grouping functionality.
func TestRouteGroup(t *testing.T) {
	cleanup := log.New(&log.Config{
		FileName:   "test_route_group.log",
		MaxSize:    10,
		MaxBackups: 1,
		Level:      "debug",
	})
	defer cleanup()

	srv := httpserver.NewServer(
		httpserver.Address(":8103"),
		httpserver.Middleware(middleware.Recovery()),
	)

	// Create nested route groups
	v1 := srv.Group("/api/v1")
	{
		users := v1.Group("/users")
		users.Route("GET", "", func(ctx context.Context, req any) (any, error) {
			return map[string]interface{}{"users": []string{}}, nil
		})
		users.Route("GET", "/:id", func(ctx context.Context, req any) (any, error) {
			return map[string]interface{}{}, nil
		})
	}

	orders := v1.Group("/orders")
	orders.Route("GET", "", func(ctx context.Context, req any) (any, error) {
		return map[string]interface{}{"orders": []string{}}, nil
	})

	// Test that all routes registered without panic
	// The actual routes are registered in the Gin router
	t.Log("Route group registration completed without panic")
}

// TestMultipleServers tests that multiple servers can be managed.
func TestMultipleServers(t *testing.T) {
	cleanup := log.New(&log.Config{
		FileName:   "test_multiple_servers.log",
		MaxSize:    10,
		MaxBackups: 1,
		Level:      "debug",
	})
	defer cleanup()

	// Create multiple servers
	srv1 := httpserver.NewServer(
		httpserver.Address(":8104"),
	)
	srv1.Route("GET", "/health", func(ctx context.Context, req any) (any, error) {
		return map[string]string{"server": "1"}, nil
	})

	srv2 := httpserver.NewServer(
		httpserver.Address(":8105"),
	)
	srv2.Route("GET", "/health", func(ctx context.Context, req any) (any, error) {
		return map[string]string{"server": "2"}, nil
	})

	// Create application with multiple servers
	application := app.New(
		app.Name("multi-server-test"),
		app.Server(srv1, srv2),
		app.StopTimeout(2*time.Second),
	)

	// Verify both servers are registered
	_ = application // Just verify it can be created without panic
}
