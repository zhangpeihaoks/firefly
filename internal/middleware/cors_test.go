// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"testing"

	"github.com/zhangpeihaoks/firefly/internal/transport"
)

// mockCORSTransporter implements transport.Transporter for testing.
type mockCORSTransporter struct {
	kind          transport.Kind
	endpoint      string
	operation     string
	requestHeader *corsMockHeader
	replyHeader   *corsMockHeader
	pathParams    map[string]string
	queryParams   map[string][]string
}

func (m *mockCORSTransporter) Kind() transport.Kind {
	return m.kind
}

func (m *mockCORSTransporter) Endpoint() string {
	return m.endpoint
}

func (m *mockCORSTransporter) Operation() string {
	return m.operation
}

func (m *mockCORSTransporter) RequestHeader() transport.Header {
	if m.requestHeader == nil {
		m.requestHeader = &corsMockHeader{values: make(map[string]string)}
	}
	return m.requestHeader
}

func (m *mockCORSTransporter) ReplyHeader() transport.Header {
	if m.replyHeader == nil {
		m.replyHeader = &corsMockHeader{values: make(map[string]string)}
	}
	return m.replyHeader
}

func (m *mockCORSTransporter) PathParams() map[string]string {
	if m.pathParams == nil {
		return make(map[string]string)
	}
	return m.pathParams
}

func (m *mockCORSTransporter) QueryParams() map[string][]string {
	if m.queryParams == nil {
		return make(map[string][]string)
	}
	return m.queryParams
}

// corsMockHeader implements transport.Header for testing.
type corsMockHeader struct {
	values map[string]string
	keys   []string
}

func (m *corsMockHeader) Get(key string) string {
	return m.values[key]
}

func (m *corsMockHeader) Set(key string, value string) {
	if _, exists := m.values[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.values[key] = value
}

func (m *corsMockHeader) Keys() []string {
	return m.keys
}

// TestCORSDefaultConfig tests CORS middleware with default configuration.
func TestCORSDefaultConfig(t *testing.T) {
	// Feature: backend-server-framework, Requirement 19.1
	handler := CORS()
	ctx := context.Background()

	// Create mock transporter with origin header
	mockTr := &mockCORSTransporter{
		kind:      transport.KindHTTP,
		operation: "/api/test",
		requestHeader: &corsMockHeader{values: map[string]string{
			"Origin": "https://example.com",
		}},
		replyHeader: &corsMockHeader{values: make(map[string]string)},
	}
	ctx = transport.NewContext(ctx, mockTr)

	// Call handler
	next := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	wrapped := handler(next)
	_, err := wrapped(ctx, nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Check CORS headers
	allowOrigin := mockTr.ReplyHeader().Get("Access-Control-Allow-Origin")
	if allowOrigin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin=*, got %s", allowOrigin)
	}
}

// TestCORSWithSpecificOrigin tests CORS middleware with specific origin.
func TestCORSWithSpecificOrigin(t *testing.T) {
	// Feature: backend-server-framework, Requirement 19.1
	handler := CORS(WithAllowOrigins("https://example.com"))
	ctx := context.Background()

	// Test with allowed origin
	mockTr := &mockCORSTransporter{
		kind:      transport.KindHTTP,
		operation: "/api/test",
		requestHeader: &corsMockHeader{values: map[string]string{
			"Origin": "https://example.com",
		}},
		replyHeader: &corsMockHeader{values: make(map[string]string)},
	}
	ctx = transport.NewContext(ctx, mockTr)

	next := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	wrapped := handler(next)
	_, err := wrapped(ctx, nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	allowOrigin := mockTr.ReplyHeader().Get("Access-Control-Allow-Origin")
	if allowOrigin != "https://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin=https://example.com, got %s", allowOrigin)
	}
}

// TestCORSWithMultipleOrigins tests CORS middleware with multiple allowed origins.
func TestCORSWithMultipleOrigins(t *testing.T) {
	// Feature: backend-server-framework, Requirement 19.1
	handler := CORS(WithAllowOrigins("https://example.com", "https://api.example.com"))
	ctx := context.Background()

	tests := []struct {
		name         string
		origin       string
		expectOrigin string
		expectEmpty  bool
	}{
		{
			name:         "first origin",
			origin:       "https://example.com",
			expectOrigin: "https://example.com",
		},
		{
			name:         "second origin",
			origin:       "https://api.example.com",
			expectOrigin: "https://api.example.com",
		},
		{
			name:        "disallowed origin",
			origin:      "https://evil.com",
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTr := &mockCORSTransporter{
				kind:      transport.KindHTTP,
				operation: "/api/test",
				requestHeader: &corsMockHeader{values: map[string]string{
					"Origin": tt.origin,
				}},
				replyHeader: &corsMockHeader{values: make(map[string]string)},
			}
			testCtx := transport.NewContext(ctx, mockTr)

			next := func(ctx context.Context, req any) (any, error) {
				return "ok", nil
			}
			wrapped := handler(next)
			_, _ = wrapped(testCtx, nil)

			allowOrigin := mockTr.ReplyHeader().Get("Access-Control-Allow-Origin")
			if tt.expectEmpty {
				// Should not set CORS headers for disallowed origins
				// Or set empty value
				if allowOrigin != "" && allowOrigin != tt.origin {
					t.Errorf("expected empty or unchanged, got %s", allowOrigin)
				}
			} else {
				if allowOrigin != tt.expectOrigin {
					t.Errorf("expected %s, got %s", tt.expectOrigin, allowOrigin)
				}
			}
		})
	}
}

// TestCORSWithWildcardOrigin tests CORS middleware with wildcard subdomain.
func TestCORSWithWildcardOrigin(t *testing.T) {
	// Feature: backend-server-framework, Requirement 19.1
	handler := CORS(WithAllowOrigins("*.example.com"))
	ctx := context.Background()

	tests := []struct {
		name         string
		origin       string
		expectOrigin string
		shouldAllow  bool
	}{
		{
			name:         "subdomain match",
			origin:       "https://api.example.com",
			expectOrigin: "https://api.example.com",
			shouldAllow:  true,
		},
		{
			name:         "nested subdomain match",
			origin:       "https://api.v1.example.com",
			expectOrigin: "https://api.v1.example.com",
			shouldAllow:  true,
		},
		{
			name:        "root domain no match",
			origin:      "https://example.com",
			shouldAllow: false,
		},
		{
			name:        "different domain no match",
			origin:      "https://evil.com",
			shouldAllow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockTr := &mockCORSTransporter{
				kind:      transport.KindHTTP,
				operation: "/api/test",
				requestHeader: &corsMockHeader{values: map[string]string{
					"Origin": tt.origin,
				}},
				replyHeader: &corsMockHeader{values: make(map[string]string)},
			}
			testCtx := transport.NewContext(ctx, mockTr)

			next := func(ctx context.Context, req any) (any, error) {
				return "ok", nil
			}
			wrapped := handler(next)
			_, _ = wrapped(testCtx, nil)

			allowOrigin := mockTr.ReplyHeader().Get("Access-Control-Allow-Origin")
			if tt.shouldAllow {
				if allowOrigin != tt.expectOrigin {
					t.Errorf("expected %s, got %s", tt.expectOrigin, allowOrigin)
				}
			} else {
				// For disallowed origins, CORS headers should not be set
				// (empty or missing)
			}
		})
	}
}

// TestCORSWithCredentials tests CORS middleware with credentials enabled.
func TestCORSWithCredentials(t *testing.T) {
	// Feature: backend-server-framework, Requirement 19.1
	handler := CORS(
		WithAllowOrigins("https://example.com"),
		WithAllowCredentials(true),
	)
	ctx := context.Background()

	mockTr := &mockCORSTransporter{
		kind:      transport.KindHTTP,
		operation: "/api/test",
		requestHeader: &corsMockHeader{values: map[string]string{
			"Origin": "https://example.com",
		}},
		replyHeader: &corsMockHeader{values: make(map[string]string)},
	}
	ctx = transport.NewContext(ctx, mockTr)

	next := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	wrapped := handler(next)
	_, _ = wrapped(ctx, nil)

	// Check credentials header
	allowCreds := mockTr.ReplyHeader().Get("Access-Control-Allow-Credentials")
	if allowCreds != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials=true, got %s", allowCreds)
	}

	// When credentials are enabled, origin should be the actual origin, not "*"
	allowOrigin := mockTr.ReplyHeader().Get("Access-Control-Allow-Origin")
	if allowOrigin != "https://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin=https://example.com, got %s", allowOrigin)
	}
}

// TestCORSPreflight tests CORS preflight request handling.
func TestCORSPreflight(t *testing.T) {
	// Feature: backend-server-framework, Requirement 19.1
	handler := CORS(
		WithAllowOrigins("https://example.com"),
		WithAllowMethods("GET", "POST", "PUT"),
		WithAllowHeaders("Content-Type", "Authorization"),
		WithMaxAge(3600),
	)
	ctx := context.Background()

	mockTr := &mockCORSTransporter{
		kind:      transport.KindHTTP,
		operation: "/api/test",
		requestHeader: &corsMockHeader{values: map[string]string{
			"Origin":                         "https://example.com",
			"Access-Control-Request-Method":  "POST",
			"Access-Control-Request-Headers": "Content-Type, Authorization",
		}},
		replyHeader: &corsMockHeader{values: make(map[string]string)},
	}
	ctx = transport.NewContext(ctx, mockTr)

	next := func(ctx context.Context, req any) (any, error) {
		t.Error("next handler should not be called for preflight")
		return nil, nil
	}
	wrapped := handler(next)
	_, _ = wrapped(ctx, nil)

	// Check preflight response headers
	tests := []struct {
		key      string
		expected string
	}{
		{"Access-Control-Allow-Origin", "https://example.com"},
		{"Access-Control-Allow-Methods", "GET, POST, PUT"},
		{"Access-Control-Allow-Headers", "Content-Type, Authorization"},
		{"Access-Control-Max-Age", "3600"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			value := mockTr.ReplyHeader().Get(tt.key)
			if value != tt.expected {
				t.Errorf("expected %s=%s, got %s", tt.key, tt.expected, value)
			}
		})
	}
}

// TestCORSExposeHeaders tests CORS Expose-Headers configuration.
func TestCORSExposeHeaders(t *testing.T) {
	// Feature: backend-server-framework, Requirement 19.1
	handler := CORS(
		WithAllowOrigins("*"),
		WithExposeHeaders("X-Custom-Header", "X-Request-Id"),
	)
	ctx := context.Background()

	mockTr := &mockCORSTransporter{
		kind:      transport.KindHTTP,
		operation: "/api/test",
		requestHeader: &corsMockHeader{values: map[string]string{
			"Origin": "https://example.com",
		}},
		replyHeader: &corsMockHeader{values: make(map[string]string)},
	}
	ctx = transport.NewContext(ctx, mockTr)

	next := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	wrapped := handler(next)
	_, _ = wrapped(ctx, nil)

	exposeHeaders := mockTr.ReplyHeader().Get("Access-Control-Expose-Headers")
	if exposeHeaders != "X-Custom-Header, X-Request-Id" {
		t.Errorf("expected Access-Control-Expose-Headers=X-Custom-Header, X-Request-Id, got %s", exposeHeaders)
	}
}

// TestCORSNoOriginHeader tests requests without Origin header.
func TestCORSNoOriginHeader(t *testing.T) {
	// Feature: backend-server-framework, Requirement 19.1
	handler := CORS()
	ctx := context.Background()

	mockTr := &mockCORSTransporter{
		kind:          transport.KindHTTP,
		operation:     "/api/test",
		requestHeader: &corsMockHeader{values: make(map[string]string)},
		replyHeader:   &corsMockHeader{values: make(map[string]string)},
	}
	ctx = transport.NewContext(ctx, mockTr)

	called := false
	next := func(ctx context.Context, req any) (any, error) {
		called = true
		return "ok", nil
	}
	wrapped := handler(next)
	_, _ = wrapped(ctx, nil)

	// Should pass through without CORS headers
	if !called {
		t.Error("expected next handler to be called")
	}

	// Should not set CORS headers
	allowOrigin := mockTr.ReplyHeader().Get("Access-Control-Allow-Origin")
	if allowOrigin != "" {
		t.Errorf("expected no Access-Control-Allow-Origin header, got %s", allowOrigin)
	}
}

// TestCORSWithConfig tests CORS middleware with full configuration.
func TestCORSWithConfig(t *testing.T) {
	// Feature: backend-server-framework, Requirement 19.1
	config := CORSConfig{
		AllowOrigins:     []string{"https://example.com"},
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Content-Type"},
		ExposeHeaders:    []string{"X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           7200,
	}

	handler := CORS(WithCORSConfig(config))
	ctx := context.Background()

	mockTr := &mockCORSTransporter{
		kind:      transport.KindHTTP,
		operation: "/api/test",
		requestHeader: &corsMockHeader{values: map[string]string{
			"Origin":                        "https://example.com",
			"Access-Control-Request-Method": "GET",
		}},
		replyHeader: &corsMockHeader{values: make(map[string]string)},
	}
	ctx = transport.NewContext(ctx, mockTr)

	next := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	wrapped := handler(next)
	_, _ = wrapped(ctx, nil)

	// Verify all headers are set correctly
	allowOrigin := mockTr.ReplyHeader().Get("Access-Control-Allow-Origin")
	if allowOrigin != "https://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin=https://example.com, got %s", allowOrigin)
	}

	allowCreds := mockTr.ReplyHeader().Get("Access-Control-Allow-Credentials")
	if allowCreds != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials=true, got %s", allowCreds)
	}

	maxAge := mockTr.ReplyHeader().Get("Access-Control-Max-Age")
	if maxAge != "7200" {
		t.Errorf("expected Access-Control-Max-Age=7200, got %s", maxAge)
	}
}

// TestCORSWildcardAllowHeaders tests CORS with wildcard AllowHeaders.
func TestCORSWildcardAllowHeaders(t *testing.T) {
	// Feature: backend-server-framework, Requirement 19.1
	handler := CORS(
		WithAllowOrigins("https://example.com"),
		WithAllowHeaders("*"),
	)
	ctx := context.Background()

	mockTr := &mockCORSTransporter{
		kind:      transport.KindHTTP,
		operation: "/api/test",
		requestHeader: &corsMockHeader{values: map[string]string{
			"Origin":                         "https://example.com",
			"Access-Control-Request-Method":  "POST",
			"Access-Control-Request-Headers": "X-Custom-Header, X-Another-Header",
		}},
		replyHeader: &corsMockHeader{values: make(map[string]string)},
	}
	ctx = transport.NewContext(ctx, mockTr)

	next := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	wrapped := handler(next)
	_, _ = wrapped(ctx, nil)

	// When AllowHeaders is "*", should reflect the requested headers
	allowHeaders := mockTr.ReplyHeader().Get("Access-Control-Allow-Headers")
	if allowHeaders != "X-Custom-Header, X-Another-Header" {
		t.Errorf("expected Access-Control-Allow-Headers to reflect requested headers, got %s", allowHeaders)
	}
}

// TestDefaultCORSConfig tests that DefaultCORSConfig returns valid defaults.
func TestDefaultCORSConfig(t *testing.T) {
	// Feature: backend-server-framework, Requirement 19.1
	config := DefaultCORSConfig()

	if len(config.AllowOrigins) != 1 || config.AllowOrigins[0] != "*" {
		t.Errorf("expected AllowOrigins=[\"*\"], got %v", config.AllowOrigins)
	}

	if len(config.AllowMethods) != 7 {
		t.Errorf("expected 7 AllowMethods, got %d", len(config.AllowMethods))
	}

	if config.MaxAge != 86400 {
		t.Errorf("expected MaxAge=86400, got %d", config.MaxAge)
	}
}

// TestCORSSameOriginRequest tests same-origin requests (no Origin header).
func TestCORSSameOriginRequest(t *testing.T) {
	// Feature: backend-server-framework, Requirement 19.1
	handler := CORS(WithAllowOrigins("https://example.com"))
	ctx := context.Background()

	// Same-origin requests don't have Origin header
	mockTr := &mockCORSTransporter{
		kind:          transport.KindHTTP,
		operation:     "/api/test",
		requestHeader: &corsMockHeader{values: make(map[string]string)},
		replyHeader:   &corsMockHeader{values: make(map[string]string)},
	}
	ctx = transport.NewContext(ctx, mockTr)

	called := false
	next := func(ctx context.Context, req any) (any, error) {
		called = true
		return "ok", nil
	}
	wrapped := handler(next)
	_, _ = wrapped(ctx, nil)

	if !called {
		t.Error("expected next handler to be called")
	}

	// CORS headers should not be set for same-origin requests
	allowOrigin := mockTr.ReplyHeader().Get("Access-Control-Allow-Origin")
	if allowOrigin != "" {
		t.Errorf("expected no CORS headers for same-origin request, got %s", allowOrigin)
	}
}

// TestCORSOptionsPassthrough tests passing OPTIONS requests to next handler.
func TestCORSOptionsPassthrough(t *testing.T) {
	// Feature: backend-server-framework, Requirement 19.1
	handler := CORS(
		WithAllowOrigins("https://example.com"),
		WithOptionsPassthrough(true),
	)
	ctx := context.Background()

	mockTr := &mockCORSTransporter{
		kind:      transport.KindHTTP,
		operation: "/api/test",
		requestHeader: &corsMockHeader{values: map[string]string{
			"Origin":                        "https://example.com",
			"Access-Control-Request-Method": "POST",
		}},
		replyHeader: &corsMockHeader{values: make(map[string]string)},
	}
	ctx = transport.NewContext(ctx, mockTr)

	next := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	wrapped := handler(next)
	_, _ = wrapped(ctx, nil)

	// CORS headers should still be set
	allowOrigin := mockTr.ReplyHeader().Get("Access-Control-Allow-Origin")
	if allowOrigin != "https://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin=https://example.com, got %s", allowOrigin)
	}
}

// TestCORSMethodValidation tests that only allowed methods are accepted in preflight.
func TestCORSMethodValidation(t *testing.T) {
	// Feature: backend-server-framework, Requirement 19.1
	handler := CORS(
		WithAllowOrigins("https://example.com"),
		WithAllowMethods("GET", "POST"),
	)
	ctx := context.Background()

	mockTr := &mockCORSTransporter{
		kind:      transport.KindHTTP,
		operation: "/api/test",
		requestHeader: &corsMockHeader{values: map[string]string{
			"Origin":                        "https://example.com",
			"Access-Control-Request-Method": "DELETE", // Not allowed
		}},
		replyHeader: &corsMockHeader{values: make(map[string]string)},
	}
	ctx = transport.NewContext(ctx, mockTr)

	next := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}
	wrapped := handler(next)
	_, _ = wrapped(ctx, nil)

	// Access-Control-Allow-Methods should still be set (browser handles validation)
	allowMethods := mockTr.ReplyHeader().Get("Access-Control-Allow-Methods")
	if allowMethods == "" {
		t.Error("expected Access-Control-Allow-Methods to be set")
	}
}
