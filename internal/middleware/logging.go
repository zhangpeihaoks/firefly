// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/transport"
)

// loggingOptions holds the configuration for Logging middleware.
type loggingOptions struct {
	logger        *slog.Logger
	requestHeader bool
	requestBody   bool
	responseBody  bool
}

// LoggingOption is a configuration option for Logging middleware.
type LoggingOption func(*loggingOptions)

// WithLoggingLogger sets a custom logger for the Logging middleware.
// If not set, the default slog logger is used.
func WithLoggingLogger(logger *slog.Logger) LoggingOption {
	return func(o *loggingOptions) {
		o.logger = logger
	}
}

// WithRequestHeader enables or disables request header logging.
// When enabled, request headers will be included in the log output.
func WithRequestHeader(enabled bool) LoggingOption {
	return func(o *loggingOptions) {
		o.requestHeader = enabled
	}
}

// WithRequestBody enables or disables request body logging.
// When enabled, the request body will be included in the log output.
// Note: Be careful when enabling this for sensitive data.
func WithRequestBody(enabled bool) LoggingOption {
	return func(o *loggingOptions) {
		o.requestBody = enabled
	}
}

// WithResponseBody enables or disables response body logging.
// When enabled, the response body will be included in the log output.
// Note: Be careful when enabling this for sensitive data.
func WithResponseBody(enabled bool) LoggingOption {
	return func(o *loggingOptions) {
		o.responseBody = enabled
	}
}

// Logging returns a middleware that logs request and response information.
// It records:
//   - Request method and path (operation)
//   - Response status code
//   - Request latency
//   - Optional: request headers, request body, response body
//
// Example:
//
//	// With default options
//	middleware.Logging()
//
//	// With custom logger
//	middleware.Logging(middleware.WithLogger(myLogger))
//
//	// With request/response body logging
//	middleware.Logging(
//	    middleware.WithRequestHeader(true),
//	    middleware.WithRequestBody(true),
//	    middleware.WithResponseBody(true),
//	)
func Logging(opts ...LoggingOption) Middleware {
	// Apply default options
	options := &loggingOptions{
		logger:        slog.Default(),
		requestHeader: false,
		requestBody:   false,
		responseBody:  false,
	}

	// Apply custom options
	for _, opt := range opts {
		opt(options)
	}

	return func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			// Record start time
			startTime := time.Now()

			// Get transport info from context
			tr := transport.FromContext(ctx)

			// Build log attributes
			var attrs []any
			var kind, operation, endpoint string

			if tr != nil {
				kind = string(tr.Kind())
				operation = tr.Operation()
				endpoint = tr.Endpoint()
				attrs = append(attrs,
					slog.String("kind", kind),
					slog.String("operation", operation),
					slog.String("endpoint", endpoint),
				)

				// Log request headers if enabled
				if options.requestHeader && tr.RequestHeader() != nil {
					headers := make(map[string]string)
					for _, key := range tr.RequestHeader().Keys() {
						headers[key] = tr.RequestHeader().Get(key)
					}
					attrs = append(attrs, slog.Any("request_header", headers))
				}
			}

			// Log request body if enabled
			if options.requestBody && req != nil {
				if bodyBytes, err := json.Marshal(req); err == nil {
					attrs = append(attrs, slog.String("request_body", string(bodyBytes)))
				}
			}

			// Call the next handler
			resp, err := next(ctx, req)

			// Calculate latency
			latency := time.Since(startTime)

			// Determine status code from error
			statusCode := 200
			if err != nil {
				statusCode = getStatusCode(err)
			}

			// Add response info to attributes
			attrs = append(attrs,
				slog.Int("status", statusCode),
				slog.Duration("latency", latency),
			)

			// Log response body if enabled
			if options.responseBody && resp != nil {
				if bodyBytes, err := json.Marshal(resp); err == nil {
					attrs = append(attrs, slog.String("response_body", string(bodyBytes)))
				}
			}

			// Log the request
			if err != nil {
				attrs = append(attrs, slog.String("error", err.Error()))
				options.logger.Error("request completed with error", attrs...)
			} else {
				options.logger.Info("request completed", attrs...)
			}

			return resp, err
		}
	}
}

// getStatusCode extracts the HTTP status code from an error.
// If the error is a framework Error, it uses the error's code.
// Otherwise, it returns 500 (Internal Server Error).
func getStatusCode(err error) int {
	if err == nil {
		return 200
	}

	// Try to extract status code from framework error
	if e, ok := err.(*errors.Error); ok {
		return int(e.Code)
	}

	// Default to internal server error
	return 500
}

// formatBody safely formats a body for logging.
// It truncates the output if it exceeds maxLen.
func formatBody(body any, maxLen int) string {
	if body == nil {
		return ""
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Sprintf("<error marshaling: %v>", err)
	}

	if len(bodyBytes) > maxLen {
		return string(bodyBytes[:maxLen]) + "... (truncated)"
	}
	return string(bodyBytes)
}
