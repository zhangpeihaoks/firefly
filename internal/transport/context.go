package transport

import (
	"context"
)

// contextKey is the key type for storing Transporter in context.
type contextKey struct{}

// transporterKey is the context key for Transporter values.
var transporterKey = contextKey{}

// NewContext returns a new context with the given Transporter attached.
func NewContext(ctx context.Context, tr Transporter) context.Context {
	return context.WithValue(ctx, transporterKey, tr)
}

// FromContext returns the Transporter stored in the context.
// Returns nil if no Transporter is found.
func FromContext(ctx context.Context) Transporter {
	if tr, ok := ctx.Value(transporterKey).(Transporter); ok {
		return tr
	}
	return nil
}

// MustFromContext returns the Transporter stored in the context.
// Panics if no Transporter is found.
func MustFromContext(ctx context.Context) Transporter {
	tr := FromContext(ctx)
	if tr == nil {
		panic("transport: no Transporter found in context")
	}
	return tr
}
