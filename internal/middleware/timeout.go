// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/transport"
)

// timeoutOptions holds the configuration for Timeout middleware.
type timeoutOptions struct {
	logger      *slog.Logger
	defaultTimeout time.Duration
	pathTimeouts   map[string]time.Duration
}

// TimeoutOption is a configuration option for Timeout middleware.
type TimeoutOption func(*timeoutOptions)

// WithTimeoutLogger sets a custom logger for the Timeout middleware.
func WithTimeoutLogger(logger *slog.Logger) TimeoutOption {
	return func(o *timeoutOptions) {
		o.logger = logger
	}
}

// WithPathTimeout sets a specific timeout for the given request path.
// This allows per-route timeout configuration.
func WithPathTimeout(path string, timeout time.Duration) TimeoutOption {
	return func(o *timeoutOptions) {
		o.pathTimeouts[path] = timeout
	}
}

// Timeout returns a middleware that enforces a per-request timeout.
// When the request processing time exceeds the configured timeout,
// the context is cancelled and a DeadlineExceeded error is returned.
//
// Example:
//
//	// 5 second default timeout
//	middleware.Timeout(5 * time.Second)
//
//	// With per-path timeouts
//	middleware.Timeout(5 * time.Second,
//	    middleware.WithPathTimeout("/api/large-request", 30*time.Second),
//	)
func Timeout(defaultTimeout time.Duration, opts ...TimeoutOption) Middleware {
	options := &timeoutOptions{
		logger:        slog.Default(),
		defaultTimeout: defaultTimeout,
		pathTimeouts:   make(map[string]time.Duration),
	}

	for _, opt := range opts {
		opt(options)
	}

	return func(next Handler) Handler {
		return func(ctx context.Context, req any) (resp any, err error) {
			// Determine the effective timeout for this request
			timeout := options.defaultTimeout

			// Check if there is a path-specific timeout
			if tr := transport.FromContext(ctx); tr != nil {
				if pathTimeout, ok := options.pathTimeouts[tr.Operation()]; ok {
					timeout = pathTimeout
				}
			}

			// Create a new context with timeout
			timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// Create a channel to receive the result
			type result struct {
				resp any
				err  error
			}
			resultCh := make(chan result, 1)

			// Execute the handler in a goroutine
			go func() {
				r, e := next(timeoutCtx, req)
				resultCh <- result{r, e}
			}()

			// Wait for either the result or the timeout
			select {
			case <-timeoutCtx.Done():
				// Check if it was the parent context that was cancelled
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				// Timeout occurred
				operation := ""
				if tr := transport.FromContext(ctx); tr != nil {
					operation = tr.Operation()
				}
				options.logger.Warn("request timeout",
					"operation", operation,
					"timeout", timeout,
				)
				return nil, errors.ErrGatewayTimeout
			case r := <-resultCh:
				return r.resp, r.err
			}
		}
	}
}
