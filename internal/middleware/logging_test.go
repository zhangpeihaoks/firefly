// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"log/slog"
	"testing"

	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/transport"
)

// mockTransporter implements transport.Transporter for testing.
type mockTransporter struct {
	kind          transport.Kind
	endpoint      string
	operation     string
	requestHeader transport.Header
	replyHeader   transport.Header
	pathParams    map[string]string
	queryParams   map[string][]string
}

func (m *mockTransporter) Kind() transport.Kind {
	return m.kind
}

func (m *mockTransporter) Endpoint() string {
	return m.endpoint
}

func (m *mockTransporter) Operation() string {
	return m.operation
}

func (m *mockTransporter) RequestHeader() transport.Header {
	return m.requestHeader
}

func (m *mockTransporter) ReplyHeader() transport.Header {
	return m.replyHeader
}

func (m *mockTransporter) PathParams() map[string]string {
	if m.pathParams == nil {
		return make(map[string]string)
	}
	return m.pathParams
}

func (m *mockTransporter) QueryParams() map[string][]string {
	if m.queryParams == nil {
		return make(map[string][]string)
	}
	return m.queryParams
}

// mockHeader implements transport.Header for testing.
type mockHeader struct {
	values map[string]string
}

func newMockHeader() *mockHeader {
	return &mockHeader{values: make(map[string]string)}
}

func (m *mockHeader) Get(key string) string {
	return m.values[key]
}

func (m *mockHeader) Set(key string, value string) {
	m.values[key] = value
}

func (m *mockHeader) Keys() []string {
	keys := make([]string, 0, len(m.values))
	for k := range m.values {
		keys = append(keys, k)
	}
	return keys
}

// TestLoggingMiddleware tests the basic functionality of Logging middleware.
func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		opts           []LoggingOption
		ctx            context.Context
		req            any
		handler        Handler
		wantErr        bool
		wantStatusCode int
	}{
		{
			name:    "successful request",
			opts:    nil,
			ctx:     context.Background(),
			req:     map[string]string{"key": "value"},
			handler: func(ctx context.Context, req any) (any, error) { return "ok", nil },
			wantErr: false,
		},
		{
			name: "request with error",
			opts: nil,
			ctx:  context.Background(),
			req:  nil,
			handler: func(ctx context.Context, req any) (any, error) {
				return nil, errors.New(400, "BAD_REQUEST", "invalid request")
			},
			wantErr: true,
		},
		{
			name: "request with transport context",
			opts: nil,
			ctx: transport.NewContext(context.Background(), &mockTransporter{
				kind:      transport.KindHTTP,
				endpoint:  "http://localhost:8080",
				operation: "GET /api/users",
			}),
			req:     nil,
			handler: func(ctx context.Context, req any) (any, error) { return "ok", nil },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create middleware
			middleware := Logging(tt.opts...)
			handler := middleware(tt.handler)

			// Execute handler
			resp, err := handler(tt.ctx, tt.req)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("Logging middleware error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check response
			if !tt.wantErr && resp == nil {
				t.Error("Logging middleware returned nil response")
			}
		})
	}
}

// TestLoggingMiddlewareWithOptions tests Logging middleware with various options.
func TestLoggingMiddlewareWithOptions(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	// Create transport context with headers
	reqHeader := newMockHeader()
	reqHeader.Set("Content-Type", "application/json")
	reqHeader.Set("X-Request-ID", "test-123")

	ctx := transport.NewContext(context.Background(), &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "http://localhost:8080",
		operation:     "POST /api/users",
		requestHeader: reqHeader,
	})

	tests := []struct {
		name                string
		opts                []LoggingOption
		req                 any
		checkLogContains    []string
		checkLogNotContains []string
	}{
		{
			name: "with request header logging",
			opts: []LoggingOption{
				WithLoggingLogger(logger),
				WithRequestHeader(true),
			},
			req:              nil,
			checkLogContains: []string{"request_header", "Content-Type", "X-Request-ID"},
		},
		{
			name: "with request body logging",
			opts: []LoggingOption{
				WithLoggingLogger(logger),
				WithRequestBody(true),
			},
			req:              map[string]string{"name": "test"},
			checkLogContains: []string{"request_body", "name", "test"},
		},
		{
			name: "with response body logging",
			opts: []LoggingOption{
				WithLoggingLogger(logger),
				WithResponseBody(true),
			},
			req:              nil,
			checkLogContains: []string{"response_body"},
		},
		{
			name: "without request header logging",
			opts: []LoggingOption{
				WithLoggingLogger(logger),
				WithRequestHeader(false),
			},
			req:                 nil,
			checkLogNotContains: []string{"request_header"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear buffer
			buf.Reset()

			// Create middleware
			middleware := Logging(tt.opts...)
			handler := middleware(func(ctx context.Context, req any) (any, error) {
				return map[string]string{"result": "success"}, nil
			})

			// Execute handler
			_, _ = handler(ctx, tt.req)

			// Get log output
			logOutput := buf.String()

			// Check log contains expected strings
			for _, s := range tt.checkLogContains {
				if !bytes.Contains([]byte(logOutput), []byte(s)) {
					t.Errorf("Log output should contain %q, got: %s", s, logOutput)
				}
			}

			// Check log does not contain unexpected strings
			for _, s := range tt.checkLogNotContains {
				if bytes.Contains([]byte(logOutput), []byte(s)) {
					t.Errorf("Log output should not contain %q, got: %s", s, logOutput)
				}
			}
		})
	}
}

// TestLoggingMiddlewareErrorStatus tests that the middleware correctly extracts status codes from errors.
func TestLoggingMiddlewareErrorStatus(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
	}{
		{
			name:         "framework error with 400",
			err:          errors.New(400, "BAD_REQUEST", "bad request"),
			expectedCode: 400,
		},
		{
			name:         "framework error with 500",
			err:          errors.New(500, "INTERNAL_ERROR", "internal error"),
			expectedCode: 500,
		},
		{
			name:         "standard error",
			err:          stderrors.New("standard error"),
			expectedCode: 500,
		},
		{
			name:         "nil error",
			err:          nil,
			expectedCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := getStatusCode(tt.err)
			if code != tt.expectedCode {
				t.Errorf("getStatusCode(%v) = %d, want %d", tt.err, code, tt.expectedCode)
			}
		})
	}
}

// TestLoggingMiddlewareWithPanic tests that Logging middleware works correctly with Recovery middleware.
func TestLoggingMiddlewareWithPanic(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	// Create combined middleware chain: Recovery -> Logging
	chain := Chain(
		Recovery(WithRecoveryLogger(logger)),
		Logging(WithLoggingLogger(logger)),
	)

	// Create handler that panics
	handler := chain(func(ctx context.Context, req any) (any, error) {
		panic("test panic")
	})

	// Execute handler
	ctx := transport.NewContext(context.Background(), &mockTransporter{
		kind:      transport.KindHTTP,
		endpoint:  "http://localhost:8080",
		operation: "GET /api/panic",
	})

	resp, err := handler(ctx, nil)

	// Should have caught the panic and returned an error
	if err == nil {
		t.Error("Expected error from panic recovery")
	}

	// Response should be nil (error case)
	if resp != nil {
		t.Error("Expected nil response from panic recovery")
	}

	// Check log output contains panic info
	logOutput := buf.String()
	if !bytes.Contains([]byte(logOutput), []byte("panic")) {
		t.Errorf("Log should contain 'panic', got: %s", logOutput)
	}
}

// TestLoggingMiddlewareLogFormat tests that the log output is properly formatted.
func TestLoggingMiddlewareLogFormat(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	// Create middleware with all options enabled
	middleware := Logging(
		WithLoggingLogger(logger),
		WithRequestHeader(true),
		WithRequestBody(true),
		WithResponseBody(true),
	)

	// Create transport context
	reqHeader := newMockHeader()
	reqHeader.Set("Content-Type", "application/json")

	ctx := transport.NewContext(context.Background(), &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "http://localhost:8080",
		operation:     "POST /api/users",
		requestHeader: reqHeader,
	})

	// Create handler
	handler := middleware(func(ctx context.Context, req any) (any, error) {
		return map[string]string{"id": "123"}, nil
	})

	// Execute handler
	_, _ = handler(ctx, map[string]string{"name": "test"})

	// Parse log output
	logOutput := buf.String()
	var logEntry map[string]any
	if err := json.Unmarshal([]byte(logOutput), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	// Check required fields
	requiredFields := []string{"kind", "operation", "endpoint", "status", "latency"}
	for _, field := range requiredFields {
		if _, ok := logEntry[field]; !ok {
			t.Errorf("Log entry missing required field: %s", field)
		}
	}

	// Check kind value
	if logEntry["kind"] != "http" {
		t.Errorf("Expected kind 'http', got: %v", logEntry["kind"])
	}

	// Check operation value
	if logEntry["operation"] != "POST /api/users" {
		t.Errorf("Expected operation 'POST /api/users', got: %v", logEntry["operation"])
	}

	// Check status value
	if logEntry["status"] != float64(200) {
		t.Errorf("Expected status 200, got: %v", logEntry["status"])
	}
}

// TestFormatBody tests the formatBody helper function.
func TestFormatBody(t *testing.T) {
	tests := []struct {
		name     string
		body     any
		maxLen   int
		expected string
	}{
		{
			name:     "nil body",
			body:     nil,
			maxLen:   100,
			expected: "",
		},
		{
			name:     "simple map",
			body:     map[string]string{"key": "value"},
			maxLen:   100,
			expected: `{"key":"value"}`,
		},
		{
			name:     "truncated body",
			body:     map[string]string{"key": "this is a very long value that should be truncated"},
			maxLen:   20,
			expected: `{"key":"this is a ve... (truncated)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBody(tt.body, tt.maxLen)
			if result != tt.expected {
				t.Errorf("formatBody() = %q, want %q", result, tt.expected)
			}
		})
	}
}
