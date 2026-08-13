// Package log provides structured logging for the Firefly framework.
package log

import (
	"context"
	"log/slog"

	"github.com/zhangpeihaoks/firefly/internal/middleware"
	"go.opentelemetry.io/otel/trace"
)

// ContextHandler is a slog.Handler wrapper that automatically extracts
// request_id, trace_id and span_id from the context and injects them into
// every log record. This correlates every log line with its request and
// distributed-trace context, so logs and traces form a single picture.
type ContextHandler struct {
	inner slog.Handler
}

// NewContextHandler wraps the given handler with context-aware correlation
// ID injection (request_id + trace_id + span_id).
func NewContextHandler(inner slog.Handler) *ContextHandler {
	return &ContextHandler{inner: inner}
}

// Enabled reports whether the handler handles records at the given level.
// Delegates to the inner handler.
func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle processes a log record. It extracts request_id and the OpenTelemetry
// span context from ctx and adds them as attributes before delegating to the
// inner handler.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := middleware.RequestIDFromContext(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}

	// Inject distributed-trace correlation IDs when a span is active.
	// The ID is injected regardless of sampling so local logs stay correlated
	// with traces even when the trace is not exported.
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}

	return h.inner.Handle(ctx, r)
}

// WithAttrs returns a new ContextHandler with the given attributes
// pre-added to the inner handler.
func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{inner: h.inner.WithAttrs(attrs)}
}

// WithGroup returns a new ContextHandler with the given group name
// applied to the inner handler.
func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{inner: h.inner.WithGroup(name)}
}
