// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"

	"github.com/zhangpeihaoks/firefly/internal/transport"
)

// requestIDOptions holds the configuration for RequestID middleware.
type requestIDOptions struct {
	logger       *slog.Logger
	headerName   string
	generateFunc func() string
}

// RequestIDOption is a configuration option for RequestID middleware.
type RequestIDOption func(*requestIDOptions)

// WithRequestIDLogger sets a custom logger for the RequestID middleware.
func WithRequestIDLogger(logger *slog.Logger) RequestIDOption {
	return func(o *requestIDOptions) {
		o.logger = logger
	}
}

// WithRequestIDHeader sets a custom header name for the request ID.
// Default: "X-Request-Id"
func WithRequestIDHeader(headerName string) RequestIDOption {
	return func(o *requestIDOptions) {
		o.headerName = headerName
	}
}

// WithRequestIDGenerator sets a custom function for generating request IDs.
// Default: UUID v4 (random)
func WithRequestIDGenerator(fn func() string) RequestIDOption {
	return func(o *requestIDOptions) {
		o.generateFunc = fn
	}
}

// requestIDContextKey is the key type for storing request ID in context.
type requestIDContextKey struct{}

// requestIDKey is the context key for request ID values.
var requestIDKey = requestIDContextKey{}

// NewContextWithRequestID returns a new context with the given request ID attached.
func NewContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext returns the request ID stored in the context.
// Returns empty string if no request ID is found.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// defaultRequestID generates a UUID v4 (random) style request ID.
func defaultRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	// Set version 4 bits
	b[6] = (b[6] & 0x0f) | 0x40
	// Set variant bits
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

const (
	// DefaultRequestIDHeader is the default header name for request ID.
	DefaultRequestIDHeader = "X-Request-Id"
)

// RequestID returns a middleware that ensures every request has a unique identifier.
// It:
//  1. Extracts the request ID from the incoming request header (if present)
//  2. Generates a new UUID v4 if no request ID is found
//  3. Injects the request ID into the context
//  4. Sets the request ID in the response header
//
// Example:
//
//	// With default options
//	middleware.RequestID()
//
//	// With custom header name
//	middleware.RequestID(middleware.WithRequestIDHeader("X-Trace-Id"))
func RequestID(opts ...RequestIDOption) Middleware {
	options := &requestIDOptions{
		logger:       slog.Default(),
		headerName:   DefaultRequestIDHeader,
		generateFunc: defaultRequestID,
	}

	for _, opt := range opts {
		opt(options)
	}

	return func(next Handler) Handler {
		return func(ctx context.Context, req any) (resp any, err error) {
			var requestID string

			// Try to extract from request header
			if tr := transport.FromContext(ctx); tr != nil {
				requestID = tr.RequestHeader().Get(options.headerName)
			}

			// Generate new ID if not found
			if requestID == "" {
				requestID = options.generateFunc()
			}

			// Inject into context
			ctx = NewContextWithRequestID(ctx, requestID)

			// Set response header
			if tr := transport.FromContext(ctx); tr != nil {
				tr.ReplyHeader().Set(options.headerName, requestID)
			}

			// Call the next handler
			return next(ctx, req)
		}
	}
}
