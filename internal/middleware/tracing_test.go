// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ferrors "github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/transport"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// setupTracer creates a test tracer provider and returns the exporter for verification.
func setupTracer(t *testing.T) (*sdktrace.TracerProvider, *tracetest.InMemoryExporter) {
	exporter := tracetest.NewInMemoryExporter()

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("test-service"),
		)),
	)

	return tp, exporter
}

func TestTracing_BasicRequest(t *testing.T) {
	tp, exporter := setupTracer(t)
	defer tp.Shutdown(context.Background())

	// Create middleware with custom tracer provider
	middleware := Tracing(WithTracerProvider(tp))

	// Create a handler that returns successfully
	handler := middleware(func(ctx context.Context, req any) (any, error) {
		return "success", nil
	})

	// Create context with transport info
	reqHeader := newMockHeader()
	replyHeader := newMockHeader()
	tr := &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "localhost:8080",
		operation:     "GET /api/test",
		requestHeader: reqHeader,
		replyHeader:   replyHeader,
	}
	ctx := transport.NewContext(context.Background(), tr)

	// Execute handler
	resp, err := handler(ctx, map[string]string{"key": "value"})

	// Verify response
	assert.NoError(t, err)
	assert.Equal(t, "success", resp)

	// Verify span was created
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Contains(t, span.Name, "firefly.http.GET /api/test")
	assert.Equal(t, trace.SpanKindServer, span.SpanKind)

	// Verify attributes
	var hasTransportKind, hasOperation, hasEndpoint bool
	for _, attr := range span.Attributes {
		if attr.Key == "transport.kind" {
			hasTransportKind = true
			assert.Equal(t, "http", attr.Value.AsString())
		}
		if attr.Key == "transport.operation" {
			hasOperation = true
			assert.Equal(t, "GET /api/test", attr.Value.AsString())
		}
		if attr.Key == "transport.endpoint" {
			hasEndpoint = true
			assert.Equal(t, "localhost:8080", attr.Value.AsString())
		}
	}
	assert.True(t, hasTransportKind, "missing transport.kind attribute")
	assert.True(t, hasOperation, "missing transport.operation attribute")
	assert.True(t, hasEndpoint, "missing transport.endpoint attribute")
}

func TestTracing_WithError(t *testing.T) {
	tp, exporter := setupTracer(t)
	defer tp.Shutdown(context.Background())

	// Create middleware with custom tracer provider
	middleware := Tracing(WithTracerProvider(tp))

	// Create a handler that returns an error
	handler := middleware(func(ctx context.Context, req any) (any, error) {
		return nil, ferrors.New(ferrors.CodeInternal, "INTERNAL_ERROR", "something went wrong")
	})

	// Create context with transport info
	reqHeader := newMockHeader()
	replyHeader := newMockHeader()
	tr := &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "localhost:8080",
		operation:     "POST /api/test",
		requestHeader: reqHeader,
		replyHeader:   replyHeader,
	}
	ctx := transport.NewContext(context.Background(), tr)

	// Execute handler
	resp, err := handler(ctx, nil)

	// Verify error
	assert.Error(t, err)
	assert.Nil(t, resp)

	// Verify span was created with error
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, "Error", span.Status.Code.String())
	assert.Contains(t, span.Status.Description, "something went wrong")

	// Verify error attributes
	var hasErrorCode, hasErrorReason bool
	for _, attr := range span.Attributes {
		if attr.Key == "error.code" {
			hasErrorCode = true
			assert.Equal(t, int64(500), attr.Value.AsInt64())
		}
		if attr.Key == "error.reason" {
			hasErrorReason = true
			assert.Equal(t, "INTERNAL_ERROR", attr.Value.AsString())
		}
	}
	assert.True(t, hasErrorCode, "missing error.code attribute")
	assert.True(t, hasErrorReason, "missing error.reason attribute")
}

func TestTracing_WithMetadata(t *testing.T) {
	tp, exporter := setupTracer(t)
	defer tp.Shutdown(context.Background())

	// Create middleware with custom tracer provider
	middleware := Tracing(WithTracerProvider(tp))

	// Create a handler that returns an error with metadata
	handler := middleware(func(ctx context.Context, req any) (any, error) {
		return nil, ferrors.New(ferrors.CodeBadRequest, "BAD_REQUEST", "invalid request").
			WithMetadata(map[string]string{"field": "email", "reason": "invalid format"})
	})

	// Create context with transport info
	reqHeader := newMockHeader()
	replyHeader := newMockHeader()
	tr := &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "localhost:8080",
		operation:     "PUT /api/test",
		requestHeader: reqHeader,
		replyHeader:   replyHeader,
	}
	ctx := transport.NewContext(context.Background(), tr)

	// Execute handler
	_, err := handler(ctx, nil)

	// Verify error
	assert.Error(t, err)

	// Verify span was created with error metadata
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	span := spans[0]
	var hasMetadataField, hasMetadataReason bool
	for _, attr := range span.Attributes {
		if attr.Key == "error.metadata.field" {
			hasMetadataField = true
			assert.Equal(t, "email", attr.Value.AsString())
		}
		if attr.Key == "error.metadata.reason" {
			hasMetadataReason = true
			assert.Equal(t, "invalid format", attr.Value.AsString())
		}
	}
	assert.True(t, hasMetadataField, "missing error.metadata.field attribute")
	assert.True(t, hasMetadataReason, "missing error.metadata.reason attribute")
}

func TestTracing_ContextPropagation(t *testing.T) {
	tp, _ := setupTracer(t)
	defer tp.Shutdown(context.Background())

	// Create propagators
	propagators := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	// Create middleware with custom tracer provider and propagators
	middleware := Tracing(
		WithTracerProvider(tp),
		WithPropagators(propagators),
	)

	// Create a handler that checks trace context
	var receivedTraceID string
	handler := middleware(func(ctx context.Context, req any) (any, error) {
		receivedTraceID = GetTraceID(ctx)
		return "success", nil
	})

	// Create context with transport info
	reqHeader := newMockHeader()
	replyHeader := newMockHeader()
	tr := &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "localhost:8080",
		operation:     "GET /api/test",
		requestHeader: reqHeader,
		replyHeader:   replyHeader,
	}
	ctx := transport.NewContext(context.Background(), tr)

	// Execute handler
	_, err := handler(ctx, nil)

	// Verify no error
	assert.NoError(t, err)

	// Verify trace ID was generated
	assert.NotEmpty(t, receivedTraceID)

	// Verify trace context was injected into response headers
	traceParent := replyHeader.Get("traceparent")
	assert.NotEmpty(t, traceParent, "traceparent header should be set for propagation")
}

func TestTracing_ExtractIncomingTraceContext(t *testing.T) {
	tp, exporter := setupTracer(t)
	defer tp.Shutdown(context.Background())

	// Create propagators
	propagators := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)

	// Create middleware
	middleware := Tracing(
		WithTracerProvider(tp),
		WithPropagators(propagators),
	)

	// Create a handler that checks trace context
	var receivedTraceID string
	handler := middleware(func(ctx context.Context, req any) (any, error) {
		receivedTraceID = GetTraceID(ctx)
		return "success", nil
	})

	// Create context with incoming trace context in headers
	reqHeader := newMockHeader()
	// Set a traceparent header (format: version-traceid-spanid-flags)
	reqHeader.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	replyHeader := newMockHeader()
	tr := &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "localhost:8080",
		operation:     "GET /api/test",
		requestHeader: reqHeader,
		replyHeader:   replyHeader,
	}
	ctx := transport.NewContext(context.Background(), tr)

	// Execute handler
	_, err := handler(ctx, nil)

	// Verify no error
	assert.NoError(t, err)

	// Verify trace ID was extracted from incoming headers
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", receivedTraceID)

	// Verify span was created as child of incoming trace
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	// Check that the span context has the correct trace ID
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", spans[0].SpanContext.TraceID().String())
}

func TestTracing_NoTransport(t *testing.T) {
	tp, exporter := setupTracer(t)
	defer tp.Shutdown(context.Background())

	// Create middleware with custom tracer provider
	middleware := Tracing(WithTracerProvider(tp))

	// Create a handler
	handler := middleware(func(ctx context.Context, req any) (any, error) {
		return "success", nil
	})

	// Execute handler without transport info in context
	resp, err := handler(context.Background(), nil)

	// Verify response
	assert.NoError(t, err)
	assert.Equal(t, "success", resp)

	// Verify span was still created
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Contains(t, spans[0].Name, "firefly.unknown")
}

func TestTracing_GRPCTransport(t *testing.T) {
	tp, exporter := setupTracer(t)
	defer tp.Shutdown(context.Background())

	// Create middleware with custom tracer provider
	middleware := Tracing(WithTracerProvider(tp))

	// Create a handler
	handler := middleware(func(ctx context.Context, req any) (any, error) {
		return "grpc success", nil
	})

	// Create context with gRPC transport info
	reqHeader := newMockHeader()
	replyHeader := newMockHeader()
	tr := &mockTransporter{
		kind:          transport.KindGRPC,
		endpoint:      "localhost:9090",
		operation:     "/api.v1.UserService/GetUser",
		requestHeader: reqHeader,
		replyHeader:   replyHeader,
	}
	ctx := transport.NewContext(context.Background(), tr)

	// Execute handler
	resp, err := handler(ctx, nil)

	// Verify response
	assert.NoError(t, err)
	assert.Equal(t, "grpc success", resp)

	// Verify span was created with gRPC transport kind
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Contains(t, span.Name, "firefly.grpc./api.v1.UserService/GetUser")

	// Verify transport.kind attribute
	var hasTransportKind bool
	for _, attr := range span.Attributes {
		if attr.Key == "transport.kind" {
			hasTransportKind = true
			assert.Equal(t, "grpc", attr.Value.AsString())
		}
	}
	assert.True(t, hasTransportKind, "missing transport.kind attribute")
}

func TestGetTraceID(t *testing.T) {
	tp, _ := setupTracer(t)
	defer tp.Shutdown(context.Background())

	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test")

	// Test with valid span
	ctx, _ := tracer.Start(context.Background(), "test-span")
	traceID := GetTraceID(ctx)
	assert.NotEmpty(t, traceID)
	assert.Len(t, traceID, 32) // TraceID is 16 bytes = 32 hex chars

	// Test with no span
	emptyTraceID := GetTraceID(context.Background())
	assert.Empty(t, emptyTraceID)
}

func TestGetSpanID(t *testing.T) {
	tp, _ := setupTracer(t)
	defer tp.Shutdown(context.Background())

	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test")

	// Test with valid span
	_, span := tracer.Start(context.Background(), "test-span")
	ctx := trace.ContextWithSpan(context.Background(), span)
	spanID := GetSpanID(ctx)
	assert.NotEmpty(t, spanID)
	assert.Len(t, spanID, 16) // SpanID is 8 bytes = 16 hex chars

	// Test with no span
	emptySpanID := GetSpanID(context.Background())
	assert.Empty(t, emptySpanID)
}

func TestStartSpan(t *testing.T) {
	tp, _ := setupTracer(t)
	defer tp.Shutdown(context.Background())

	otel.SetTracerProvider(tp)

	// Start a parent span
	parentCtx, parentSpan := StartSpan(context.Background(), "parent-operation")
	defer parentSpan.End()

	// Verify parent span is valid
	assert.True(t, parentSpan.SpanContext().IsValid())

	// Start a child span
	childCtx, childSpan := StartSpan(parentCtx, "child-operation")
	defer childSpan.End()

	// Verify child span is valid
	assert.True(t, childSpan.SpanContext().IsValid())

	// Verify both spans have the same trace ID
	parentTraceID := GetTraceID(parentCtx)
	childTraceID := GetTraceID(childCtx)
	assert.Equal(t, parentTraceID, childTraceID)
}

func TestHeaderCarrier(t *testing.T) {
	header := newMockHeader()
	carrier := &headerCarrier{header: header}

	// Test Set and Get
	carrier.Set("key1", "value1")
	assert.Equal(t, "value1", carrier.Get("key1"))

	// Test Keys
	carrier.Set("key2", "value2")
	keys := carrier.Keys()
	assert.Len(t, keys, 2)

	// Test with nil header
	nilCarrier := &headerCarrier{header: nil}
	assert.Empty(t, nilCarrier.Get("key"))
	nilCarrier.Set("key", "value") // Should not panic
	assert.Nil(t, nilCarrier.Keys())
}

func TestTracing_GenericError(t *testing.T) {
	tp, exporter := setupTracer(t)
	defer tp.Shutdown(context.Background())

	// Create middleware with custom tracer provider
	middleware := Tracing(WithTracerProvider(tp))

	// Create a handler that returns a generic error (not framework error)
	handler := middleware(func(ctx context.Context, req any) (any, error) {
		return nil, errors.New("generic error")
	})

	// Create context with transport info
	reqHeader := newMockHeader()
	replyHeader := newMockHeader()
	tr := &mockTransporter{
		kind:          transport.KindHTTP,
		endpoint:      "localhost:8080",
		operation:     "DELETE /api/test",
		requestHeader: reqHeader,
		replyHeader:   replyHeader,
	}
	ctx := transport.NewContext(context.Background(), tr)

	// Execute handler
	_, err := handler(ctx, nil)

	// Verify error
	assert.Error(t, err)

	// Verify span was created with error
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	span := spans[0]
	assert.Equal(t, "Error", span.Status.Code.String())
	assert.Contains(t, span.Status.Description, "generic error")
}
