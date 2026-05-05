// Package http provides HTTP server implementation for the Firefly framework.
package http

import (
	"context"

	"github.com/zhangpeihaoks/firefly/internal/transport"
)

// contextKey is a private type for context keys.
type contextKey struct{}

// transporterKey is the key for storing transporter in context.
var transporterKey = contextKey{}

// NewContext creates a new context with the HTTP transporter.
func NewContext(ctx context.Context, t transport.Transporter) context.Context {
	return context.WithValue(ctx, transporterKey, t)
}

// FromContext extracts the HTTP transporter from the context.
func FromContext(ctx context.Context) (transport.Transporter, bool) {
	t, ok := ctx.Value(transporterKey).(transport.Transporter)
	return t, ok
}

// MustFromContext extracts the HTTP transporter from the context, panicking if not found.
func MustFromContext(ctx context.Context) transport.Transporter {
	t, ok := FromContext(ctx)
	if !ok {
		panic("HTTP transporter not found in context")
	}
	return t
}
