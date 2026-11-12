// Package middleware exposes the part of Firefly's middleware system that
// services living outside this module can use.
//
// Everything under internal/ is importable only from inside
// github.com/zhangpeihaoks/firefly/..., so a service in its own repository
// cannot reach the middleware constructors directly. This package is the
// supported surface: the Handler/Middleware types plus the middleware that
// every HTTP service needs.
//
// The constructors deliberately accept no options. Option types are internal
// and would leak into the public API; a service that needs a different
// timeout or a custom logger configures it through its own middleware.
package middleware

import (
	"context"
	"time"

	internal "github.com/zhangpeihaoks/firefly/internal/middleware"
)

// Handler is the request handler signature shared by every Firefly
// transport: it receives the request context and the transport-specific
// request value, and returns the response body or an error.
//
// For HTTP routes registered through pkg/server the req argument is the
// original *http.Request, so handlers that need the raw body, query string,
// or headers type-assert it.
type Handler = internal.Handler

// Middleware wraps a Handler to add behaviour that runs before and/or after
// it. The first middleware in a chain runs first.
type Middleware = internal.Middleware

// Chain composes middlewares into one, applying them in the order given:
// Chain(a, b, c)(h) runs a, then b, then c, then h.
func Chain(ms ...Middleware) Middleware {
	return internal.Chain(ms...)
}

// Recovery turns a panic inside any later handler into an internal error
// response instead of crashing the process.
func Recovery() Middleware {
	return internal.Recovery()
}

// Logging records one structured log line per request, including method,
// path, status code and latency.
func Logging() Middleware {
	return internal.Logging()
}

// RequestID reads X-Request-ID from the request (or generates one), stores
// it in the context, and echoes it back on the response.
func RequestID() Middleware {
	return internal.RequestID()
}

// Timeout cancels the request context after d. The handler is expected to
// return once its context is done.
func Timeout(d time.Duration) Middleware {
	return internal.Timeout(d)
}

// RequestIDFromContext returns the request id RequestID stored in ctx, or an
// empty string when the middleware did not run for this request.
func RequestIDFromContext(ctx context.Context) string {
	return internal.RequestIDFromContext(ctx)
}
