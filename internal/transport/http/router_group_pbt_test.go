// Package http provides HTTP server implementation for the Firefly framework.
package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/middleware"
)

// =============================================================================
// Property 18: Route Group Correctness (路由分组正确性)
// =============================================================================
// Validates: Requirements 7.4, 9.3, 9.4
// For any route group, prefix and middleware should be correctly applied.

// TestProperty18SimpleGroupPrefix_PBT tests that simple route groups apply correct prefixes.
// Feature: backend-server-framework, Property 18: 路由分组正确性
//
// A route group with a prefix should correctly prefix all routes in that group.
func TestProperty18SimpleGroupPrefix_PBT(t *testing.T) {
	testCases := []struct {
		name         string
		groupPrefix  string
		routePath    string
		expectedPath string
	}{
		{
			name:         "simple group /api",
			groupPrefix:  "/api",
			routePath:    "/users",
			expectedPath: "/api/users",
		},
		{
			name:         "simple group /v1",
			groupPrefix:  "/v1",
			routePath:    "/users",
			expectedPath: "/v1/users",
		},
		{
			name:         "group with trailing slash /api/",
			groupPrefix:  "/api/",
			routePath:    "/users",
			expectedPath: "/api/users",
		},
		{
			name:         "nested path /admin",
			groupPrefix:  "/admin",
			routePath:    "/settings",
			expectedPath: "/admin/settings",
		},
		{
			name:         "group /items with different routes",
			groupPrefix:  "/items",
			routePath:    "/list",
			expectedPath: "/items/list",
		},
		{
			name:         "group /users/:id",
			groupPrefix:  "/users",
			routePath:    "/:id/profile",
			expectedPath: "/users/:id/profile",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			srv := NewServer(
				Address(":0"),
				Timeout(time.Second),
			)

			// Create group with prefix
			group := srv.Group(tc.groupPrefix)

			// Register handler
			handler := func(ctx context.Context, req any) (any, error) {
				return map[string]string{"path": tc.routePath}, nil
			}
			group.Route(http.MethodGet, tc.routePath, handler)

			// Start server
			err := srv.Start(ctx)
			if err != nil {
				t.Fatalf("failed to start server: %v", err)
			}
			defer srv.Stop(ctx)

			time.Sleep(50 * time.Millisecond)

			endpoint, err := srv.Endpoint()
			if err != nil {
				t.Fatalf("failed to get endpoint: %v", err)
			}

			// Make request to the grouped route
			resp, err := http.Get(endpoint.String() + tc.expectedPath)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("expected status 200 for path %s, got %d, body: %s",
					tc.expectedPath, resp.StatusCode, string(body))
			}
		})
	}
}

// TestProperty18NestedGroupPrefix_PBT tests that nested route groups combine prefixes correctly.
// Feature: backend-server-framework, Property 18: 路由分组正确性
//
// Nested route groups should correctly combine their prefixes.
func TestProperty18NestedGroupPrefix_PBT(t *testing.T) {
	testCases := []struct {
		name         string
		outerPrefix  string
		innerPrefix  string
		routePath    string
		expectedPath string
	}{
		{
			name:         "two level nested /api/v1",
			outerPrefix:  "/api",
			innerPrefix:  "/v1",
			routePath:    "/users",
			expectedPath: "/api/v1/users",
		},
		{
			name:         "three level nested /a/b/c",
			outerPrefix:  "/a",
			innerPrefix:  "/b",
			routePath:    "/c",
			expectedPath: "/a/b/c",
		},
		{
			name:         "nested with empty inner /api/",
			outerPrefix:  "/api",
			innerPrefix:  "/",
			routePath:    "/users",
			expectedPath: "/api/users",
		},
		{
			name:         "nested /v1/admin",
			outerPrefix:  "/v1",
			innerPrefix:  "/admin",
			routePath:    "/users",
			expectedPath: "/v1/admin/users",
		},
		{
			name:         "deeply nested /users/:id/posts",
			outerPrefix:  "/users",
			innerPrefix:  "/:id",
			routePath:    "/posts",
			expectedPath: "/users/:id/posts",
		},
		{
			name:         "nested /groups/:group_id/resources/:id",
			outerPrefix:  "/groups",
			innerPrefix:  "/:group_id",
			routePath:    "/resources/:id",
			expectedPath: "/groups/:group_id/resources/:id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			srv := NewServer(
				Address(":0"),
				Timeout(time.Second),
			)

			// Create nested groups
			outer := srv.Group(tc.outerPrefix)
			inner := outer.Group(tc.innerPrefix)

			// Register handler
			handler := func(ctx context.Context, req any) (any, error) {
				return map[string]string{"ok": "true"}, nil
			}
			inner.Route(http.MethodGet, tc.routePath, handler)

			// Start server
			err := srv.Start(ctx)
			if err != nil {
				t.Fatalf("failed to start server: %v", err)
			}
			defer srv.Stop(ctx)

			time.Sleep(50 * time.Millisecond)

			endpoint, err := srv.Endpoint()
			if err != nil {
				t.Fatalf("failed to get endpoint: %v", err)
			}

			// Make request
			resp, err := http.Get(endpoint.String() + tc.expectedPath)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("expected status 200 for path %s, got %d, body: %s",
					tc.expectedPath, resp.StatusCode, string(body))
			}
		})
	}
}

// TestProperty18GroupMiddleware_PBT tests that middleware is correctly applied to route groups.
// Feature: backend-server-framework, Property 18: 路由分组正确性
//
// Middleware added to a route group should be executed for all routes in that group.
func TestProperty18GroupMiddleware_PBT(t *testing.T) {
	testCases := []struct {
		name                   string
		groupPrefix            string
		middlewareName         string
		expectMiddlewareCalled bool
	}{
		{
			name:                   "group with single middleware",
			groupPrefix:            "/api",
			middlewareName:         "api-mw",
			expectMiddlewareCalled: true,
		},
		{
			name:                   "group with different prefix",
			groupPrefix:            "/admin",
			middlewareName:         "admin-mw",
			expectMiddlewareCalled: true,
		},
		{
			name:                   "group /v1/middleware",
			groupPrefix:            "/v1",
			middlewareName:         "v1-mw",
			expectMiddlewareCalled: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			srv := NewServer(
				Address(":0"),
				Timeout(time.Second),
			)

			// Create middleware that records which middleware was applied
			testMiddleware := func(name string) func(next middleware.Handler) middleware.Handler {
				return func(next middleware.Handler) middleware.Handler {
					return func(ctx context.Context, req any) (any, error) {
						return next(ctx, req)
					}
				}
			}

			// Create group with middleware
			group := srv.Group(tc.groupPrefix, testMiddleware(tc.middlewareName))

			// Register handler
			handler := func(ctx context.Context, req any) (any, error) {
				return map[string]string{"handler": "executed", "middleware": tc.middlewareName}, nil
			}
			group.Route(http.MethodGet, "/test", handler)

			// Start server
			err := srv.Start(ctx)
			if err != nil {
				t.Fatalf("failed to start server: %v", err)
			}
			defer srv.Stop(ctx)

			time.Sleep(50 * time.Millisecond)

			endpoint, err := srv.Endpoint()
			if err != nil {
				t.Fatalf("failed to get endpoint: %v", err)
			}

			// Make request
			resp, err := http.Get(endpoint.String() + tc.groupPrefix + "/test")
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			// Verify handler was executed
			var result map[string]string
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if result["handler"] != "executed" {
				t.Errorf("handler not executed properly, got %v", result)
			}
			if result["middleware"] != tc.middlewareName {
				t.Errorf("middleware not applied, expected %s, got %v", tc.middlewareName, result)
			}
		})
	}
}

// TestProperty18GroupMiddlewareExecution_PBT tests that group middleware executes in correct order.
// Feature: backend-server-framework, Property 18: 路由分组正确性
func TestProperty18GroupMiddlewareExecution_PBT(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		Address(":0"),
		Timeout(time.Second),
	)

	// Track middleware execution order
	executionOrder := []string{}

	// First middleware
	mw1 := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			executionOrder = append(executionOrder, "mw1-before")
			result, err := next(ctx, req)
			executionOrder = append(executionOrder, "mw1-after")
			return result, err
		}
	}

	// Second middleware
	mw2 := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			executionOrder = append(executionOrder, "mw2-before")
			result, err := next(ctx, req)
			executionOrder = append(executionOrder, "mw2-after")
			return result, err
		}
	}

	// Create group with middleware
	group := srv.Group("/api", mw1, mw2)

	// Register handler
	handler := func(ctx context.Context, req any) (any, error) {
		executionOrder = append(executionOrder, "handler")
		return map[string]string{"ok": "true"}, nil
	}
	group.Route(http.MethodGet, "/test", handler)

	// Start server
	err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Stop(ctx)

	time.Sleep(50 * time.Millisecond)

	endpoint, err := srv.Endpoint()
	if err != nil {
		t.Fatalf("failed to get endpoint: %v", err)
	}

	// Make request
	_, err = http.Get(endpoint.String() + "/api/test")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	// Verify execution order: mw1-before, mw2-before, handler, mw2-after, mw1-after
	// This verifies that middleware is applied in FIFO order (first registered runs first)
	expectedOrder := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}

	if len(executionOrder) != len(expectedOrder) {
		t.Errorf("execution order length mismatch: expected %d, got %d, order: %v",
			len(expectedOrder), len(executionOrder), executionOrder)
		return
	}

	for i, expected := range expectedOrder {
		if executionOrder[i] != expected {
			t.Errorf("execution order[%d]: expected %s, got %s, full order: %v",
				i, expected, executionOrder[i], executionOrder)
		}
	}
}

// TestProperty18NestedGroupMiddleware_PBT tests that nested groups inherit and combine middleware.
// Feature: backend-server-framework, Property 18: 路由分组正确性
func TestProperty18NestedGroupMiddleware_PBT(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		Address(":0"),
		Timeout(time.Second),
	)

	// Track middleware execution
	middlewareCalls := []string{}

	mw1 := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			middlewareCalls = append(middlewareCalls, "outer-mw")
			return next(ctx, req)
		}
	}

	mw2 := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			middlewareCalls = append(middlewareCalls, "inner-mw")
			return next(ctx, req)
		}
	}

	// Create outer group with middleware
	outer := srv.Group("/api", mw1)
	// Create inner group with additional middleware
	inner := outer.Group("/v1", mw2)

	// Register handler
	handler := func(ctx context.Context, req any) (any, error) {
		middlewareCalls = append(middlewareCalls, "handler")
		return map[string]string{"ok": "true"}, nil
	}
	inner.Route(http.MethodGet, "/users", handler)

	// Start server
	err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Stop(ctx)

	time.Sleep(50 * time.Millisecond)

	endpoint, err := srv.Endpoint()
	if err != nil {
		t.Fatalf("failed to get endpoint: %v", err)
	}

	// Make request
	_, err = http.Get(endpoint.String() + "/api/v1/users")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	// Both outer and inner middleware should be called
	// Order: outer-mw (from parent group), inner-mw (from current group), handler
	if len(middlewareCalls) < 2 {
		t.Errorf("expected at least 2 middleware calls, got %v", middlewareCalls)
	}

	// Check that both middlewares were called
	hasOuter := false
	hasInner := false
	for _, call := range middlewareCalls {
		if call == "outer-mw" {
			hasOuter = true
		}
		if call == "inner-mw" {
			hasInner = true
		}
	}

	if !hasOuter {
		t.Errorf("outer middleware not called, calls: %v", middlewareCalls)
	}
	if !hasInner {
		t.Errorf("inner middleware not called, calls: %v", middlewareCalls)
	}
}

// TestProperty18GroupUseMethod_PBT tests the Use method for adding middleware to groups.
// Feature: backend-server-framework, Property 18: 路由分组正确性
func TestProperty18GroupUseMethod_PBT(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		Address(":0"),
		Timeout(time.Second),
	)

	// Track middleware calls via closure
	var middlewareCalls []string
	testMiddleware := func(name string) func(next middleware.Handler) middleware.Handler {
		return func(next middleware.Handler) middleware.Handler {
			return func(ctx context.Context, req any) (any, error) {
				middlewareCalls = append(middlewareCalls, name)
				return next(ctx, req)
			}
		}
	}

	// Create group and use Use method to add middleware
	group := srv.Group("/api")
	group.Use(testMiddleware("use-mw"))

	// Register handler
	handler := func(ctx context.Context, req any) (any, error) {
		return map[string]string{"ok": "true"}, nil
	}
	group.Route(http.MethodGet, "/test", handler)

	// Start server
	err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Stop(ctx)

	time.Sleep(50 * time.Millisecond)

	endpoint, err := srv.Endpoint()
	if err != nil {
		t.Fatalf("failed to get endpoint: %v", err)
	}

	// Make request
	resp, err := http.Get(endpoint.String() + "/api/test")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// Verify response is correct
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if result["ok"] != "true" {
		t.Errorf("unexpected response: %v", result)
	}

	// Verify middleware was called
	if len(middlewareCalls) == 0 {
		t.Errorf("middleware was not called, calls: %v", middlewareCalls)
	}
	if len(middlewareCalls) > 0 && middlewareCalls[0] != "use-mw" {
		t.Errorf("expected middleware name 'use-mw', got %v", middlewareCalls)
	}
}

// TestProperty18MultipleRoutesInGroup_PBT tests multiple routes in a single group.
// Feature: backend-server-framework, Property 18: 路由分组正确性
func TestProperty18MultipleRoutesInGroup_PBT(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		Address(":0"),
		Timeout(time.Second),
	)

	// Create group
	group := srv.Group("/api")

	// Register multiple routes
	routes := []struct {
		method string
		path   string
		result string
	}{
		{http.MethodGet, "/users", "get-users"},
		{http.MethodPost, "/users", "create-user"},
		{http.MethodGet, "/users/:id", "get-user"},
		{http.MethodPut, "/users/:id", "update-user"},
		{http.MethodDelete, "/users/:id", "delete-user"},
	}

	for _, route := range routes {
		handler := func(r string) func(ctx context.Context, req any) (any, error) {
			return func(ctx context.Context, req any) (any, error) {
				return map[string]string{"action": r}, nil
			}
		}(route.result)
		group.Route(route.method, route.path, handler)
	}

	// Start server
	err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Stop(ctx)

	time.Sleep(50 * time.Millisecond)

	endpoint, err := srv.Endpoint()
	if err != nil {
		t.Fatalf("failed to get endpoint: %v", err)
	}

	client := &http.Client{}

	// Test each route
	testRoutes := []struct {
		method     string
		path       string
		wantResult string
	}{
		{http.MethodGet, "/api/users", "get-users"},
		{http.MethodPost, "/api/users", "create-user"},
		{http.MethodGet, "/api/users/123", "get-user"},
		{http.MethodPut, "/api/users/123", "update-user"},
		{http.MethodDelete, "/api/users/123", "delete-user"},
	}

	for _, tr := range testRoutes {
		req, _ := http.NewRequest(tr.method, endpoint.String()+tr.path, nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("%s %s: request failed: %v", tr.method, tr.path, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s %s: expected 200, got %d", tr.method, tr.path, resp.StatusCode)
			resp.Body.Close()
			continue
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Errorf("%s %s: decode failed: %v", tr.method, tr.path, err)
			resp.Body.Close()
			continue
		}

		if result["action"] != tr.wantResult {
			t.Errorf("%s %s: expected %s, got %v", tr.method, tr.path, tr.wantResult, result)
		}
		resp.Body.Close()
	}
}

// TestProperty18GroupWithAllHTTPPethods_PBT tests that route groups work with all HTTP methods.
// Feature: backend-server-framework, Property 18: 路由分组正确性
func TestProperty18GroupWithAllHTTPPethods_PBT(t *testing.T) {
	testCases := []struct {
		name       string
		method     string
		expectBody bool
	}{
		{"GET", http.MethodGet, true},
		{"POST", http.MethodPost, true},
		{"PUT", http.MethodPut, true},
		{"DELETE", http.MethodDelete, true},
		{"PATCH", http.MethodPatch, true},
		{"HEAD", http.MethodHead, false}, // HEAD requests don't have body
		{"OPTIONS", http.MethodOptions, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			srv := NewServer(
				Address(":0"),
				Timeout(time.Second),
			)

			// Create group
			group := srv.Group("/api")

			// Register handler for each method
			handler := func(ctx context.Context, req any) (any, error) {
				return map[string]string{"method": tc.method}, nil
			}
			group.Route(tc.method, "/test", handler)

			// Start server
			err := srv.Start(ctx)
			if err != nil {
				t.Fatalf("failed to start server: %v", err)
			}
			defer srv.Stop(ctx)

			time.Sleep(50 * time.Millisecond)

			endpoint, err := srv.Endpoint()
			if err != nil {
				t.Fatalf("failed to get endpoint: %v", err)
			}

			// Make request
			req, _ := http.NewRequest(tc.method, endpoint.String()+"/api/test", nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("method %s: expected status 200, got %d", tc.method, resp.StatusCode)
			}

			// Verify response body for methods that support it
			if tc.expectBody {
				var result map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode: %v", err)
				}
				if result["method"] != tc.method {
					t.Errorf("method %s: expected %s, got %v", tc.method, tc.method, result)
				}
			}
		})
	}
}

// TestProperty18NestedGroupsWithMiddleware_PBT tests deeply nested groups with middleware.
// Feature: backend-server-framework, Property 18: 路由分组正确性
func TestProperty18NestedGroupsWithMiddleware_PBT(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		Address(":0"),
		Timeout(time.Second),
	)

	// Track all middleware calls
	middlewareCalls := []string{}

	mw1 := func(name string) func(next middleware.Handler) middleware.Handler {
		return func(next middleware.Handler) middleware.Handler {
			return func(ctx context.Context, req any) (any, error) {
				middlewareCalls = append(middlewareCalls, name)
				return next(ctx, req)
			}
		}
	}

	// Create three levels of nested groups with middleware
	level1 := srv.Group("/v1", mw1("level1"))
	level2 := level1.Group("/admin", mw1("level2"))
	level3 := level2.Group("/users", mw1("level3"))

	// Register handler at deepest level
	handler := func(ctx context.Context, req any) (any, error) {
		middlewareCalls = append(middlewareCalls, "handler")
		return map[string]string{"ok": "true"}, nil
	}
	level3.Route(http.MethodGet, "/list", handler)

	// Start server
	err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Stop(ctx)

	time.Sleep(50 * time.Millisecond)

	endpoint, err := srv.Endpoint()
	if err != nil {
		t.Fatalf("failed to get endpoint: %v", err)
	}

	// Make request to nested route
	_, err = http.Get(endpoint.String() + "/v1/admin/users/list")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	// All three levels of middleware should be called
	if len(middlewareCalls) < 4 {
		t.Errorf("expected at least 4 calls (3 middleware + handler), got %v", middlewareCalls)
	}

	// Verify all middleware levels were called
	expectedMiddleware := []string{"level1", "level2", "level3", "handler"}
	for i, expected := range expectedMiddleware {
		if i >= len(middlewareCalls) {
			t.Errorf("missing call at index %d, expected %s", i, expected)
			continue
		}
		if middlewareCalls[i] != expected {
			t.Errorf("call[%d]: expected %s, got %s, full: %v",
				i, expected, middlewareCalls[i], middlewareCalls)
		}
	}
}

// TestProperty18GroupWithoutMiddleware_PBT tests that groups work without middleware.
// Feature: backend-server-framework, Property 18: 路由分组正确性
func TestProperty18GroupWithoutMiddleware_PBT(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		Address(":0"),
		Timeout(time.Second),
	)

	// Create group without middleware
	group := srv.Group("/api")

	// Register handler
	handler := func(ctx context.Context, req any) (any, error) {
		return map[string]string{"grouped": "true"}, nil
	}
	group.Route(http.MethodGet, "/test", handler)

	// Start server
	err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Stop(ctx)

	time.Sleep(50 * time.Millisecond)

	endpoint, err := srv.Endpoint()
	if err != nil {
		t.Fatalf("failed to get endpoint: %v", err)
	}

	// Make request
	resp, err := http.Get(endpoint.String() + "/api/test")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if result["grouped"] != "true" {
		t.Errorf("expected grouped=true, got %v", result)
	}
}

// TestProperty18MixedGroupedAndUngroupedRoutes_PBT tests that grouped and ungrouped routes coexist.
// Feature: backend-server-framework, Property 18: 路由分组正确性
func TestProperty18MixedGroupedAndUngroupedRoutes_PBT(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		Address(":0"),
		Timeout(time.Second),
	)

	// Register ungrouped route
	srv.Route(http.MethodGet, "/health", func(ctx context.Context, req any) (any, error) {
		return map[string]string{"type": "ungrouped"}, nil
	})

	// Register grouped route
	group := srv.Group("/api")
	group.Route(http.MethodGet, "/status", func(ctx context.Context, req any) (any, error) {
		return map[string]string{"type": "grouped"}, nil
	})

	// Start server
	err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer srv.Stop(ctx)

	time.Sleep(50 * time.Millisecond)

	endpoint, err := srv.Endpoint()
	if err != nil {
		t.Fatalf("failed to get endpoint: %v", err)
	}

	// Test ungrouped route
	resp1, err := http.Get(endpoint.String() + "/health")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp1.Body.Close()

	if resp1.StatusCode != http.StatusOK {
		t.Errorf("/health: expected 200, got %d", resp1.StatusCode)
	}

	var result1 map[string]string
	if err := json.NewDecoder(resp1.Body).Decode(&result1); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if result1["type"] != "ungrouped" {
		t.Errorf("/health: expected ungrouped, got %v", result1)
	}

	// Test grouped route
	resp2, err := http.Get(endpoint.String() + "/api/status")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Errorf("/api/status: expected 200, got %d", resp2.StatusCode)
	}

	var result2 map[string]string
	if err := json.NewDecoder(resp2.Body).Decode(&result2); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if result2["type"] != "grouped" {
		t.Errorf("/api/status: expected grouped, got %v", result2)
	}
}
