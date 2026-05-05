// Package middleware provides middleware abstractions for the Firefly framework.
// It defines the Handler and Middleware types, and utilities for composing middleware chains.
package middleware

import (
	"context"
)

// Handler is the request handler function type.
// It takes a context and request, and returns a response and error.
type Handler func(ctx context.Context, req any) (any, error)

// Middleware is the middleware function type.
// It wraps a Handler and returns a new Handler with additional behavior.
type Middleware func(Handler) Handler

// Chain combines multiple middleware into a single middleware.
// Middleware is applied in the order they are provided (first middleware runs first).
// Example: Chain(m1, m2, m3)(handler) results in m1 -> m2 -> m3 -> handler
func Chain(middlewares ...Middleware) Middleware {
	return func(final Handler) Handler {
		// Apply middleware in reverse order so that the first middleware
		// in the list is the outermost (runs first)
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// Adapt converts a middleware function to work with a specific handler type.
// This is useful for adapting framework middleware to external handler types.
func Adapt(m Middleware, toHandler func(Handler) any, fromHandler func(any) Handler) Middleware {
	return func(h Handler) Handler {
		return fromHandler(toHandler(m(h)))
	}
}
