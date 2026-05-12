// Package log provides structured logging for the Firefly framework.
package log

import (
	"context"
	"log/slog"

	"github.com/zhangpeihaoks/firefly/internal/middleware"
)

// ContextHandler is a slog.Handler wrapper that automatically extracts
// request_id from the context and injects it into every log record.
// This ensures all log entries produced during request processing carry
// the correlation ID for traceability.
type ContextHandler struct {
	inner slog.Handler
}

// NewContextHandler wraps the given handler with context-aware request ID injection.
func NewContextHandler(inner slog.Handler) *ContextHandler {
	return &ContextHandler{inner: inner}
}

// Enabled reports whether the handler handles records at the given level.
// Delegates to the inner handler.
func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle processes a log record. It extracts request_id from the context
// and adds it as an attribute before delegating to the inner handler.
func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := middleware.RequestIDFromContext(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
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
