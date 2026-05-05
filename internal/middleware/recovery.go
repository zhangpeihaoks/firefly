// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/zhangpeihaoks/firefly/internal/errors"
)

// recoveryOptions holds the configuration for Recovery middleware.
type recoveryOptions struct {
	logger  *slog.Logger
	handler func(ctx context.Context, req, err any)
}

// RecoveryOption is a configuration option for Recovery middleware.
type RecoveryOption func(*recoveryOptions)

// WithRecoveryHandler sets a custom panic handler function.
// The handler is called with the context, request, and panic value.
func WithRecoveryHandler(h func(ctx context.Context, req, err any)) RecoveryOption {
	return func(o *recoveryOptions) {
		o.handler = h
	}
}

// WithRecoveryLogger sets a custom logger for the Recovery middleware.
// If not set, the default slog logger is used.
func WithRecoveryLogger(logger *slog.Logger) RecoveryOption {
	return func(o *recoveryOptions) {
		o.logger = logger
	}
}

// Recovery returns a middleware that recovers from panics in the handler chain.
// When a panic occurs, it:
//  1. Logs the panic with stack trace
//  2. Calls the custom handler if configured
//  3. Returns a 500 Internal Server Error response
//
// Example:
//
//	// With default options
//	middleware.Recovery()
//
//	// With custom logger
//	middleware.Recovery(middleware.WithLogger(myLogger))
//
//	// With custom handler
//	middleware.Recovery(middleware.WithHandler(func(ctx context.Context, req, err any) {
//	    // Custom panic handling logic
//	    notifyMonitoring(ctx, err)
//	}))
func Recovery(opts ...RecoveryOption) Middleware {
	// Apply default options
	options := &recoveryOptions{
		logger: slog.Default(),
	}

	// Apply custom options
	for _, opt := range opts {
		opt(options)
	}

	return func(next Handler) Handler {
		return func(ctx context.Context, req any) (resp any, err error) {
			// Use defer to catch panics
			defer func() {
				if r := recover(); r != nil {
					// Get stack trace
					stack := string(debug.Stack())

					// Log the panic
					if options.logger != nil {
						options.logger.Error("panic recovered",
							"error", fmt.Sprintf("%v", r),
							"stack", stack,
						)
					}

					// Call custom handler if configured
					if options.handler != nil {
						options.handler(ctx, req, r)
					}

					// Convert panic to error response
					err = errors.New(errors.CodeInternal, "INTERNAL_ERROR", "内部服务错误")
				}
			}()

			// Call the next handler
			return next(ctx, req)
		}
	}
}
