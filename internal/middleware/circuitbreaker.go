// Package middleware provides middleware abstractions for the Firefly framework.
// This file implements the circuit breaker middleware.
package middleware

import (
	"context"
	"log/slog"

	"github.com/zhangpeihaoks/firefly/internal/circuitbreaker"
	"github.com/zhangpeihaoks/firefly/internal/errors"
)

// circuitBreakerOptions holds the configuration for CircuitBreaker middleware.
type circuitBreakerOptions struct {
	logger    *slog.Logger
	onReject  func(ctx context.Context, key string)
}

// CircuitBreakerOption is a configuration option for CircuitBreaker middleware.
type CircuitBreakerOption func(*circuitBreakerOptions)

// WithCircuitBreakerLogger sets a custom logger.
func WithCircuitBreakerLogger(logger *slog.Logger) CircuitBreakerOption {
	return func(o *circuitBreakerOptions) {
		o.logger = logger
	}
}

// WithCircuitBreakerOnReject registers a callback invoked whenever a request
// is rejected because the circuit is open (e.g. for metrics).
func WithCircuitBreakerOnReject(fn func(ctx context.Context, key string)) CircuitBreakerOption {
	return func(o *circuitBreakerOptions) {
		o.onReject = fn
	}
}

// CircuitBreaker returns a middleware that protects downstream handlers with a
// circuit breaker.
//
// While the breaker is Closed, requests pass through and their outcomes feed
// the breaker's failure statistics. When it trips Open, requests fail fast
// with 503 CIRCUIT_OPEN instead of hitting the (degraded) handler.
//
// Example:
//
//	b := circuitbreaker.New(
//	    circuitbreaker.WithFailureCount(5),
//	    circuitbreaker.WithFailureRatio(0.5),
//	    circuitbreaker.WithMinRequests(20),
//	    circuitbreaker.WithTimeout(30*time.Second),
//	)
//	server := httpserver.NewServer(
//	    httpserver.Middleware(middleware.CircuitBreaker(b)),
//	)
func CircuitBreaker(b *circuitbreaker.Breaker, opts ...CircuitBreakerOption) Middleware {
	options := &circuitBreakerOptions{logger: slog.Default()}
	for _, opt := range opts {
		opt(options)
	}

	return func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			cb, err := b.Allow()
			if err != nil {
				// Circuit open: fail fast.
				if options.onReject != nil {
					options.onReject(ctx, "")
				}
				options.logger.Warn("circuit breaker rejected request",
					"state", b.State().String(),
				)
				return nil, errors.New(errors.CodeServiceUnavailable, "CIRCUIT_OPEN", "服务熔断已开启，请稍后重试")
			}

			resp, handlerErr := next(ctx, req)
			cb(handlerErr == nil)
			return resp, handlerErr
		}
	}
}
