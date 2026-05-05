// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"log/slog"

	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/transport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// tracingOptions holds the configuration for Tracing middleware.
type tracingOptions struct {
	tracerProvider trace.TracerProvider
	propagators    propagation.TextMapPropagator
	tracer         trace.Tracer
}

// TracingOption is a configuration option for Tracing middleware.
type TracingOption func(*tracingOptions)

// WithTracerProvider sets a custom TracerProvider for the Tracing middleware.
// If not set, the global TracerProvider is used.
func WithTracerProvider(tp trace.TracerProvider) TracingOption {
	return func(o *tracingOptions) {
		o.tracerProvider = tp
	}
}

// WithPropagators sets custom propagators for trace context propagation.
// If not set, the global propagators are used.
func WithPropagators(propagators propagation.TextMapPropagator) TracingOption {
	return func(o *tracingOptions) {
		o.propagators = propagators
	}
}

const (
	// tracerName is the name of the tracer used by the middleware.
	tracerName = "github.com/zhangpeihaoks/firefly/middleware/tracing"
	// spanNamePrefix is the prefix for span names.
	spanNamePrefix = "firefly."
)

// Tracing returns a middleware that integrates with OpenTelemetry for distributed tracing.
// It:
//  1. Extracts trace context from incoming requests (if present)
//  2. Generates new trace IDs for requests without existing trace context
//  3. Creates spans for each request with proper attributes
//  4. Propagates trace context to downstream calls
//  5. Records error stack traces when requests fail
//
// Example:
//
//	// With default options (uses global TracerProvider and propagators)
//	middleware.Tracing()
//
//	// With custom TracerProvider
//	middleware.Tracing(middleware.WithTracerProvider(myTracerProvider))
//
//	// With custom propagators
//	middleware.Tracing(middleware.WithPropagators(propagation.NewCompositeTextMapPropagator(
//	    propagation.TraceContext{},
//	    propagation.Baggage{},
//	)))
func Tracing(opts ...TracingOption) Middleware {
	// Apply default options
	options := &tracingOptions{
		tracerProvider: otel.GetTracerProvider(),
		propagators:    otel.GetTextMapPropagator(),
	}

	// Apply custom options
	for _, opt := range opts {
		opt(options)
	}

	// Create tracer from provider
	options.tracer = options.tracerProvider.Tracer(tracerName)

	return func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			// Get transport info from context
			tr := transport.FromContext(ctx)

			// Build span name
			spanName := spanNamePrefix
			if tr != nil {
				spanName += string(tr.Kind()) + "." + tr.Operation()
			} else {
				spanName += "unknown"
			}

			// Extract trace context from incoming request headers if available
			var spanCtx context.Context
			if tr != nil && tr.RequestHeader() != nil {
				// Create a carrier from the request headers
				carrier := &headerCarrier{header: tr.RequestHeader()}
				ctx = options.propagators.Extract(ctx, carrier)
			}

			// Start a new span
			spanCtx, span := options.tracer.Start(
				ctx,
				spanName,
				trace.WithSpanKind(trace.SpanKindServer),
			)

			// Set span attributes from transport info
			if tr != nil {
				span.SetAttributes(
					attribute.String("transport.kind", string(tr.Kind())),
					attribute.String("transport.operation", tr.Operation()),
					attribute.String("transport.endpoint", tr.Endpoint()),
				)
			}

			// Inject trace context into response headers for propagation
			if tr != nil && tr.ReplyHeader() != nil {
				carrier := &headerCarrier{header: tr.ReplyHeader()}
				options.propagators.Inject(spanCtx, carrier)
			}

			// Call the next handler
			resp, err := next(spanCtx, req)

			// Record error if occurred
			if err != nil {
				recordError(span, err)
			}

			// End the span
			span.End()

			return resp, err
		}
	}
}

// recordError records an error on the span with stack trace information.
// It sets the span status to Error and adds error details as attributes.
func recordError(span trace.Span, err error) {
	if err == nil {
		return
	}

	// Set span status to error
	span.SetStatus(codes.Error, err.Error())

	// Record the error with stack trace
	span.RecordError(err, trace.WithStackTrace(true))

	// If it's a framework error, add additional attributes
	if e, ok := err.(*errors.Error); ok {
		span.SetAttributes(
			attribute.Int64("error.code", int64(e.Code)),
			attribute.String("error.reason", e.Reason),
		)
		if len(e.Metadata) > 0 {
			for k, v := range e.Metadata {
				span.SetAttributes(attribute.String("error.metadata."+k, v))
			}
		}
	}
}

// headerCarrier implements propagation.TextMapCarrier for Header interface.
// It allows OpenTelemetry to read and write trace context from/to headers.
type headerCarrier struct {
	header transport.Header
}

// Get returns the value for the given key from the header.
func (c *headerCarrier) Get(key string) string {
	if c.header == nil {
		return ""
	}
	return c.header.Get(key)
}

// Set stores the key-value pair in the header.
func (c *headerCarrier) Set(key, value string) {
	if c.header != nil {
		c.header.Set(key, value)
	}
}

// Keys returns all keys in the header.
func (c *headerCarrier) Keys() []string {
	if c.header == nil {
		return nil
	}
	return c.header.Keys()
}

// GetTraceID extracts the trace ID from the context.
// Returns an empty string if no trace is found.
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

// GetSpanID extracts the span ID from the context.
// Returns an empty string if no span is found.
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().SpanID().String()
}

// SpanFromContext returns the current span from the context.
// This is a convenience wrapper around trace.SpanFromContext.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// ContextWithSpan returns a new context with the given span attached.
// This is a convenience wrapper around trace.ContextWithSpan.
func ContextWithSpan(ctx context.Context, span trace.Span) context.Context {
	return trace.ContextWithSpan(ctx, span)
}

// StartSpan starts a new span as a child of the current span in the context.
// This is useful for creating child spans in business logic.
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	return tracer.Start(ctx, name, opts...)
}

// LogSpanInfo logs the current span information to the provided logger.
// This is useful for debugging and correlating logs with traces.
func LogSpanInfo(ctx context.Context, logger *slog.Logger, msg string, args ...any) {
	traceID := GetTraceID(ctx)
	spanID := GetSpanID(ctx)

	if traceID != "" {
		args = append(args, "trace_id", traceID, "span_id", spanID)
	}

	logger.Info(msg, args...)
}

// LogSpanError logs an error with span information to the provided logger.
// This is useful for error logging with trace correlation.
func LogSpanError(ctx context.Context, logger *slog.Logger, msg string, err error, args ...any) {
	traceID := GetTraceID(ctx)
	spanID := GetSpanID(ctx)

	args = append(args, "trace_id", traceID, "span_id", spanID, "error", err)

	logger.Error(msg, args...)
}
