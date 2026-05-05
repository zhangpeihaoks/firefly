package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/middleware"
)

// TestDynamicRoutingIntegration tests dynamic routing with actual HTTP requests
func TestDynamicRoutingIntegration(t *testing.T) {
	ctx := context.Background()

	// Create server with random port
	srv := NewServer(
		Address(":0"), // Use port 0 for random available port
		Timeout(time.Second),
	)

	// Register test handlers
	registerTestHandlers(srv)

	// Start server
	err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Stop(ctx)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Get server endpoint
	endpoint, err := srv.Endpoint()
	if err != nil {
		t.Fatalf("failed to get endpoint: %v", err)
	}

	baseURL := endpoint.String()

	// Test 1: Dynamic path parameter
	t.Run("PathParameter", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/users/123", baseURL))
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if result["id"] != float64(123) {
			t.Errorf("expected id 123, got %v", result["id"])
		}
		if result["name"] != "User 123" {
			t.Errorf("expected name 'User 123', got %v", result["name"])
		}
	})

	// Test 2: Query parameters
	t.Run("QueryParameters", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/search?q=test&page=2&limit=10", baseURL))
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if result["query"] != "test" {
			t.Errorf("expected query 'test', got %v", result["query"])
		}
		if result["page"] != float64(2) {
			t.Errorf("expected page 2, got %v", result["page"])
		}
		if result["limit"] != float64(10) {
			t.Errorf("expected limit 10, got %v", result["limit"])
		}
	})

	// Test 3: Multiple path parameters
	t.Run("MultiplePathParameters", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/orgs/456/users/789", baseURL))
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		if result["org_id"] != float64(456) {
			t.Errorf("expected org_id 456, got %v", result["org_id"])
		}
		if result["user_id"] != float64(789) {
			t.Errorf("expected user_id 789, got %v", result["user_id"])
		}
	})

	// Test 4: Missing required parameter
	t.Run("MissingParameter", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/users", baseURL))
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		defer resp.Body.Close()

		// Should return 404 since route doesn't match
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected status 404 for missing route, got %d", resp.StatusCode)
		}
	})

	// Test 5: Invalid parameter type
	t.Run("InvalidParameterType", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/users/abc", baseURL))
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		// Handler should handle invalid parameter gracefully
		if result["error"] == nil {
			t.Error("expected error for invalid parameter type")
		}
	})
}

func registerTestHandlers(srv *Server) {
	// User handler with path parameter
	userHandler := func(ctx context.Context, req any) (any, error) {
		id, ok := GetPathParamInt(ctx, "id")
		if !ok {
			return map[string]interface{}{
				"error": "id parameter not found or invalid",
			}, nil
		}
		return map[string]interface{}{
			"id":   id,
			"name": fmt.Sprintf("User %d", id),
		}, nil
	}

	// Search handler with query parameters
	searchHandler := func(ctx context.Context, req any) (any, error) {
		query, _ := GetQueryParam(ctx, "q")
		page, _ := GetQueryParamInt(ctx, "page")
		limit, _ := GetQueryParamInt(ctx, "limit")
		return map[string]interface{}{
			"query": query,
			"page":  page,
			"limit": limit,
		}, nil
	}

	// Handler with multiple path parameters
	orgUserHandler := func(ctx context.Context, req any) (any, error) {
		orgID, _ := GetPathParamInt(ctx, "org_id")
		userID, _ := GetPathParamInt(ctx, "user_id")
		return map[string]interface{}{
			"org_id":  orgID,
			"user_id": userID,
		}, nil
	}

	// Register routes
	srv.Route(http.MethodGet, "/users/:id", userHandler)
	srv.Route(http.MethodGet, "/search", searchHandler)
	srv.Route(http.MethodGet, "/orgs/:org_id/users/:user_id", orgUserHandler)
}

// TestMiddlewareWithParameters tests that middleware can access parameters
func TestMiddlewareWithParameters(t *testing.T) {
	ctx := context.Background()

	// Create server with random port
	srv := NewServer(
		Address(":0"),
		Timeout(time.Second),
	)

	// Create middleware that logs parameters
	loggingMiddleware := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			// Access path parameters in middleware
			id, ok := GetPathParam(ctx, "id")
			if ok {
				fmt.Printf("Middleware: path parameter id=%s\n", id)
			}

			// Access query parameters in middleware
			debug, _ := GetQueryParam(ctx, "debug")
			if debug == "true" {
				fmt.Printf("Middleware: debug mode enabled\n")
			}

			return next(ctx, req)
		}
	}

	// Handler that uses parameters
	handler := func(ctx context.Context, req any) (any, error) {
		id, _ := GetPathParamInt(ctx, "id")
		return map[string]interface{}{
			"id":      id,
			"message": "processed in middleware chain",
		}, nil
	}

	// Register route with middleware
	srv.Route(http.MethodGet, "/test/:id", handler, loggingMiddleware)

	// Start server
	err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Stop(ctx)

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Get server endpoint
	endpoint, err := srv.Endpoint()
	if err != nil {
		t.Fatalf("failed to get endpoint: %v", err)
	}

	// Make request
	resp, err := http.Get(fmt.Sprintf("%s/test/999?debug=true", endpoint.String()))
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if result["id"] != float64(999) {
		t.Errorf("expected id 999, got %v", result["id"])
	}
}
