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
// Property 17: Route Registration Correctness (路由注册正确性)
// =============================================================================
// Validates: Requirements 7.3, 9.1, 9.5
// For any HTTP method and route, after registration it should be correctly accessible.

// TestProperty17RouteRegistration_PBT tests route registration for all HTTP methods.
// Feature: backend-server-framework, Property 17: 路由注册正确性
//
// The route registration should work correctly for any HTTP method and path.
func TestProperty17RouteRegistration_PBT(t *testing.T) {
	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		handlerBody    string
	}{
		// GET method tests
		{
			name:           "GET simple path",
			method:         http.MethodGet,
			path:           "/test-get",
			expectedStatus: http.StatusOK,
			handlerBody:    "GET OK",
		},
		{
			name:           "GET nested path",
			method:         http.MethodGet,
			path:           "/api/users/list",
			expectedStatus: http.StatusOK,
			handlerBody:    "GET users list",
		},
		// POST method tests
		{
			name:           "POST simple path",
			method:         http.MethodPost,
			path:           "/test-post",
			expectedStatus: http.StatusOK,
			handlerBody:    "POST OK",
		},
		{
			name:           "POST nested path",
			method:         http.MethodPost,
			path:           "/api/users/create",
			expectedStatus: http.StatusOK,
			handlerBody:    "POST create user",
		},
		// PUT method tests
		{
			name:           "PUT simple path",
			method:         http.MethodPut,
			path:           "/test-put",
			expectedStatus: http.StatusOK,
			handlerBody:    "PUT OK",
		},
		{
			name:           "PUT nested path",
			method:         http.MethodPut,
			path:           "/api/users/update",
			expectedStatus: http.StatusOK,
			handlerBody:    "PUT update user",
		},
		// DELETE method tests
		{
			name:           "DELETE simple path",
			method:         http.MethodDelete,
			path:           "/test-delete",
			expectedStatus: http.StatusOK,
			handlerBody:    "DELETE OK",
		},
		{
			name:           "DELETE nested path",
			method:         http.MethodDelete,
			path:           "/api/users/delete",
			expectedStatus: http.StatusOK,
			handlerBody:    "DELETE delete user",
		},
		// PATCH method tests
		{
			name:           "PATCH simple path",
			method:         http.MethodPatch,
			path:           "/test-patch",
			expectedStatus: http.StatusOK,
			handlerBody:    "PATCH OK",
		},
		{
			name:           "PATCH nested path",
			method:         http.MethodPatch,
			path:           "/api/users/patch",
			expectedStatus: http.StatusOK,
			handlerBody:    "PATCH patch user",
		},
		// Additional edge cases
		{
			name:           "GET root path",
			method:         http.MethodGet,
			path:           "/",
			expectedStatus: http.StatusOK,
			handlerBody:    "root",
		},
		{
			name:           "GET path with hyphens",
			method:         http.MethodGet,
			path:           "/api/my-endpoint",
			expectedStatus: http.StatusOK,
			handlerBody:    "hyphen path",
		},
		{
			name:           "GET path with numbers",
			method:         http.MethodGet,
			path:           "/api/v2/users",
			expectedStatus: http.StatusOK,
			handlerBody:    "versioned path",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			// Create server with random port
			srv := NewServer(
				Address(":0"),
				Timeout(time.Second),
			)

			// Register handler for each method
			handler := func(ctx context.Context, req any) (any, error) {
				return map[string]string{"message": tc.handlerBody}, nil
			}
			srv.Route(tc.method, tc.path, handler)

			// Start server
			err := srv.Start(ctx)
			if err != nil {
				t.Fatalf("failed to start server: %v", err)
			}
			defer srv.Stop(ctx)

			// Give server time to start
			time.Sleep(50 * time.Millisecond)

			// Get server endpoint
			endpoint, err := srv.Endpoint()
			if err != nil {
				t.Fatalf("failed to get endpoint: %v", err)
			}

			// Make request
			req, err := http.NewRequest(tc.method, endpoint.String()+tc.path, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			defer resp.Body.Close()

			// Verify status code
			if resp.StatusCode != tc.expectedStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("expected status %d, got %d, body: %s", tc.expectedStatus, resp.StatusCode, string(body))
			}

			// Verify response body
			if resp.StatusCode == http.StatusOK {
				var result map[string]string
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				if result["message"] != tc.handlerBody {
					t.Errorf("expected message %s, got %v", tc.handlerBody, result)
				}
			}
		})
	}
}

// TestProperty17MultipleRoutes_PBT tests that multiple routes can be registered and work correctly.
// Feature: backend-server-framework, Property 17: 路由注册正确性
func TestProperty17MultipleRoutes_PBT(t *testing.T) {
	ctx := context.Background()

	// Create server
	srv := NewServer(
		Address(":0"),
		Timeout(time.Second),
	)

	// Register multiple routes with different methods
	routes := []struct {
		method string
		path   string
		result string
	}{
		{http.MethodGet, "/users", "get users"},
		{http.MethodPost, "/users", "create user"},
		{http.MethodGet, "/users/123", "get user 123"},
		{http.MethodPut, "/users/123", "update user 123"},
		{http.MethodDelete, "/users/123", "delete user 123"},
		{http.MethodPatch, "/users/123/patch", "patch user 123"},
	}

	// Register all routes
	for _, route := range routes {
		handler := func(ctx context.Context, req any) (any, error) {
			return map[string]string{"action": route.result}, nil
		}
		srv.Route(route.method, route.path, handler)
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

	// Test each route
	client := &http.Client{}
	for _, route := range routes {
		req, _ := http.NewRequest(route.method, endpoint.String()+route.path, nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("failed to make %s request to %s: %v", route.method, route.path, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("route %s %s: expected status 200, got %d", route.method, route.path, resp.StatusCode)
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Errorf("route %s %s: failed to decode: %v", route.method, route.path, err)
			continue
		}

		if result["action"] != route.result {
			t.Errorf("route %s %s: expected %s, got %v", route.method, route.path, route.result, result)
		}

		resp.Body.Close()
	}
}

// =============================================================================
// Property 19: Dynamic Route Parameter Parsing (动态路由参数解析)
// =============================================================================
// Validates: Requirements 7.7, 9.2
// For any dynamic route and parameter value, the parameter should be correctly parsed.

// TestProperty19DynamicRouteParams_PBT tests dynamic route parameter parsing.
// Feature: backend-server-framework, Property 19: 动态路由参数解析
//
// Dynamic route parameters should be correctly parsed for any route pattern and value.
func TestProperty19DynamicRouteParams_PBT(t *testing.T) {
	testCases := []struct {
		name             string
		route            string
		requestPath      string
		expectedParams   map[string]string
		expectedIntParam map[string]int64
	}{
		// Single parameter tests
		{
			name:        "single param :id",
			route:       "/users/:id",
			requestPath: "/users/123",
			expectedParams: map[string]string{
				"id": "123",
			},
			expectedIntParam: map[string]int64{
				"id": 123,
			},
		},
		{
			name:        "single param :user_id",
			route:       "/users/:user_id",
			requestPath: "/users/456",
			expectedParams: map[string]string{
				"user_id": "456",
			},
			expectedIntParam: map[string]int64{
				"user_id": 456,
			},
		},
		// Multiple parameter tests
		{
			name:        "multiple params :org_id :user_id",
			route:       "/orgs/:org_id/users/:user_id",
			requestPath: "/orgs/100/users/200",
			expectedParams: map[string]string{
				"org_id":  "100",
				"user_id": "200",
			},
			expectedIntParam: map[string]int64{
				"org_id":  100,
				"user_id": 200,
			},
		},
		{
			name:        "multiple params :category :item_id",
			route:       "/:category/:item_id",
			requestPath: "/products/abc123",
			expectedParams: map[string]string{
				"category": "products",
				"item_id":  "abc123",
			},
		},
		// Edge cases with special values
		{
			name:        "param with zero",
			route:       "/items/:id",
			requestPath: "/items/0",
			expectedParams: map[string]string{
				"id": "0",
			},
			expectedIntParam: map[string]int64{
				"id": 0,
			},
		},
		{
			name:        "param with large number",
			route:       "/records/:id",
			requestPath: "/records/9999999999",
			expectedParams: map[string]string{
				"id": "9999999999",
			},
			expectedIntParam: map[string]int64{
				"id": 9999999999,
			},
		},
		{
			name:        "param with alphanumeric",
			route:       "/codes/:code",
			requestPath: "/codes/abc123xyz",
			expectedParams: map[string]string{
				"code": "abc123xyz",
			},
		},
		{
			name:        "param with hyphens",
			route:       "/files/:filename",
			requestPath: "/files/my-file-name.txt",
			expectedParams: map[string]string{
				"filename": "my-file-name.txt",
			},
		},
		// More complex route patterns
		{
			name:        "nested dynamic route",
			route:       "/api/v1/:version/users/:id/profile",
			requestPath: "/api/v1/v2.0/users/789/profile",
			expectedParams: map[string]string{
				"version": "v2.0",
				"id":      "789",
			},
			expectedIntParam: map[string]int64{
				"id": 789,
			},
		},
		{
			name:        "action parameter",
			route:       "/users/:id/:action",
			requestPath: "/users/555/edit",
			expectedParams: map[string]string{
				"id":     "555",
				"action": "edit",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			// Create server
			srv := NewServer(
				Address(":0"),
				Timeout(time.Second),
			)

			// Register handler that extracts parameters
			handler := func(ctx context.Context, req any) (any, error) {
				result := make(map[string]interface{})

				// Extract string params
				for key := range tc.expectedParams {
					if val, ok := GetPathParam(ctx, key); ok {
						result[key] = val
					}
				}

				// Extract int params if expected
				for key := range tc.expectedIntParam {
					if val, ok := GetPathParamInt64(ctx, key); ok {
						result[key+"_int"] = val
					}
				}

				return result, nil
			}
			srv.Route(http.MethodGet, tc.route, handler)

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
			resp, err := http.Get(endpoint.String() + tc.requestPath)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("expected status 200, got %d, body: %s", resp.StatusCode, string(body))
				return
			}

			// Parse response
			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			// Verify string parameters
			for key, expectedVal := range tc.expectedParams {
				if result[key] != expectedVal {
					t.Errorf("param %s: expected %s, got %v", key, expectedVal, result[key])
				}
			}

			// Verify int parameters
			for key, expectedVal := range tc.expectedIntParam {
				intKey := key + "_int"
				if intVal, ok := result[intKey]; !ok {
					t.Errorf("param %s not found in result", intKey)
				} else if intVal != float64(expectedVal) {
					t.Errorf("param %s: expected %d, got %v", key, expectedVal, intVal)
				}
			}
		})
	}
}

// TestProperty19DynamicRouteWithQuery_PBT tests dynamic routes combined with query parameters.
// Feature: backend-server-framework, Property 19: 动态路由参数解析
func TestProperty19DynamicRouteWithQuery_PBT(t *testing.T) {
	ctx := context.Background()

	// Create server
	srv := NewServer(
		Address(":0"),
		Timeout(time.Second),
	)

	// Handler that uses both path and query params
	handler := func(ctx context.Context, req any) (any, error) {
		userID, _ := GetPathParam(ctx, "user_id")
		action, _ := GetQueryParam(ctx, "action")
		page, _ := GetQueryParamInt(ctx, "page")

		return map[string]interface{}{
			"user_id": userID,
			"action":  action,
			"page":    page,
		}, nil
	}

	srv.Route(http.MethodGet, "/users/:user_id/posts", handler)

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

	// Test cases for path + query params
	testCases := []struct {
		name         string
		path         string
		expectedID   string
		expectedAct  string
		expectedPage int
	}{
		{
			name:         "path with query params",
			path:         "/users/123/posts?action=recent&page=2",
			expectedID:   "123",
			expectedAct:  "recent",
			expectedPage: 2,
		},
		{
			name:         "path with only path param",
			path:         "/users/456/posts",
			expectedID:   "456",
			expectedAct:  "",
			expectedPage: 0,
		},
		{
			name:         "path with empty query value",
			path:         "/users/789/posts?action=",
			expectedID:   "789",
			expectedAct:  "",
			expectedPage: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(endpoint.String() + tc.path)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status 200, got %d", resp.StatusCode)
				return
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("failed to decode: %v", err)
			}

			if result["user_id"] != tc.expectedID {
				t.Errorf("user_id: expected %s, got %v", tc.expectedID, result["user_id"])
			}
			if result["action"] != tc.expectedAct {
				t.Errorf("action: expected %s, got %v", tc.expectedAct, result["action"])
			}
			if int(result["page"].(float64)) != tc.expectedPage {
				t.Errorf("page: expected %d, got %v", tc.expectedPage, result["page"])
			}
		})
	}
}

// TestProperty19RouteNotFound_PBT tests that unmatched routes return 404.
// Feature: backend-server-framework, Property 19: 动态路由参数解析
func TestProperty19RouteNotFound_PBT(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		Address(":0"),
		Timeout(time.Second),
	)

	// Register a simple route
	srv.Route(http.MethodGet, "/users/:id", func(ctx context.Context, req any) (any, error) {
		return map[string]string{"ok": "true"}, nil
	})

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

	// Test that non-matching routes return 404
	notFoundCases := []string{
		"/users",          // Missing required param
		"/posts",          // Completely different path
		"/users/123/edit", // Extra segment
		"/api/unknown",    // Unknown API path
	}

	for _, path := range notFoundCases {
		resp, err := http.Get(endpoint.String() + path)
		if err != nil {
			t.Errorf("path %s: request failed: %v", path, err)
			continue
		}

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("path %s: expected 404, got %d", path, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

// TestProperty19MethodNotAllowed_PBT tests that wrong method returns appropriate error.
// Note: Gin's behavior for unregistered methods varies - it may return 404 or 405.
// Both are acceptable error responses for this test.
func TestProperty19MethodNotAllowed_PBT(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		Address(":0"),
		Timeout(time.Second),
	)

	// Register only GET handler
	srv.Route(http.MethodGet, "/resource", func(ctx context.Context, req any) (any, error) {
		return map[string]string{"method": "get"}, nil
	})

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

	// Test that wrong methods return error (404 or 405 are both acceptable)
	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		req, _ := http.NewRequest(method, endpoint.String()+"/resource", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("method %s: request failed: %v", method, err)
			continue
		}

		// Accept both 404 (Not Found) and 405 (Method Not Allowed) as valid error responses
		if resp.StatusCode != http.StatusNotFound && resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("method %s: expected 404 or 405, got %d", method, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

// TestProperty19MiddlewareWithParams_PBT tests that middleware can access dynamic parameters.
// Feature: backend-server-framework, Property 19: 动态路由参数解析
func TestProperty19MiddlewareWithParams_PBT(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		Address(":0"),
		Timeout(time.Second),
	)

	// Middleware that accesses path params
	paramLogger := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			// Try to access params in middleware
			id, idOk := GetPathParam(ctx, "id")
			action, actionOk := GetPathParam(ctx, "action")

			// Add metadata about param access
			result, err := next(ctx, req)
			if m, ok := result.(map[string]interface{}); ok {
				m["middleware_id"] = id
				m["middleware_id_found"] = idOk
				m["middleware_action"] = action
				m["middleware_action_found"] = actionOk
			}
			return result, err
		}
	}

	// Handler that uses params
	handler := func(ctx context.Context, req any) (any, error) {
		id, _ := GetPathParam(ctx, "id")
		action, _ := GetPathParam(ctx, "action")
		return map[string]interface{}{
			"handler_id":     id,
			"handler_action": action,
		}, nil
	}

	srv.Route(http.MethodGet, "/:id/:action", handler, paramLogger)

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

	// Test request
	resp, err := http.Get(endpoint.String() + "/123/edit")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	// Verify handler has access to params
	if result["handler_id"] != "123" {
		t.Errorf("handler_id: expected 123, got %v", result["handler_id"])
	}
	if result["handler_action"] != "edit" {
		t.Errorf("handler_action: expected edit, got %v", result["handler_action"])
	}

	// Verify middleware also has access to params
	if result["middleware_id"] != "123" {
		t.Errorf("middleware_id: expected 123, got %v", result["middleware_id"])
	}
	if result["middleware_action"] != "edit" {
		t.Errorf("middleware_action: expected edit, got %v", result["middleware_action"])
	}
}

// =============================================================================
// Combined Tests for Multiple Properties
// =============================================================================

// TestProperty17And19Combined_PBT tests both route registration and dynamic params together.
// Feature: backend-server-framework, Properties 17 & 19
func TestProperty17And19Combined_PBT(t *testing.T) {
	ctx := context.Background()

	srv := NewServer(
		Address(":0"),
		Timeout(time.Second),
	)

	// Define routes with all HTTP methods and dynamic params
	routes := []struct {
		method string
		path   string
	}{
		// Static routes
		{http.MethodGet, "/static"},
		{http.MethodPost, "/static"},
		// Dynamic routes
		{http.MethodGet, "/users/:id"},
		{http.MethodPut, "/users/:id"},
		{http.MethodDelete, "/users/:id"},
		// Nested dynamic
		{http.MethodGet, "/orgs/:org_id/users/:user_id"},
	}

	for _, route := range routes {
		handler := func(ctx context.Context, req any) (any, error) {
			return map[string]string{
				"method": route.method,
				"path":   route.path,
			}, nil
		}
		srv.Route(route.method, route.path, handler)
	}

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

	// Test static routes
	staticTests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/static"},
		{http.MethodPost, "/static"},
	}

	for _, test := range staticTests {
		req, _ := http.NewRequest(test.method, endpoint.String()+test.path, nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("%s %s: request failed: %v", test.method, test.path, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s %s: expected 200, got %d", test.method, test.path, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// Test dynamic routes
	dynamicTests := []struct {
		method     string
		path       string
		requestURL string
	}{
		{http.MethodGet, "/users/:id", "/users/42"},
		{http.MethodPut, "/users/:id", "/users/42"},
		{http.MethodDelete, "/users/:id", "/users/42"},
		{http.MethodGet, "/orgs/:org_id/users/:user_id", "/orgs/10/users/20"},
	}

	for _, test := range dynamicTests {
		req, _ := http.NewRequest(test.method, endpoint.String()+test.requestURL, nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("%s %s: request failed: %v", test.method, test.requestURL, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s %s: expected 200, got %d", test.method, test.requestURL, resp.StatusCode)
		}

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Errorf("%s %s: decode failed: %v", test.method, test.requestURL, err)
			continue
		}

		if result["method"] != test.method {
			t.Errorf("%s %s: method mismatch, got %v", test.method, test.requestURL, result)
		}
		resp.Body.Close()
	}
}
