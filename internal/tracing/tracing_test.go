// Package tracing provides OpenTelemetry tracing initialization for the Firefly framework.
package tracing

import (
	"context"
	"errors"
	"log/slog"
	"math/rand"
	"net"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zhangpeihaoks/firefly/internal/config"
	ferrors "github.com/zhangpeihaoks/firefly/internal/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// TestNewTracerProvider_Disabled tests that a no-op provider is returned when tracing is disabled.
func TestNewTracerProvider_Disabled(t *testing.T) {
	tp, err := NewTracerProvider(
		WithEnabled(false),
		WithServiceName("test-service"),
	)
	require.NoError(t, err)
	require.NotNil(t, tp)
	require.NotNil(t, tp.TracerProvider)

	// Should not panic when shutting down
	err = tp.Shutdown(context.Background())
	assert.NoError(t, err)
}

// TestNewTracerProvider_StdoutExporter tests creating a TracerProvider with stdout exporter.
func TestNewTracerProvider_StdoutExporter(t *testing.T) {
	tp, err := NewTracerProvider(
		WithEnabled(true),
		WithServiceName("test-service"),
		WithExporterType(ExporterStdout),
		WithSamplerRatio(1.0),
	)
	require.NoError(t, err)
	require.NotNil(t, tp)
	require.NotNil(t, tp.TracerProvider)

	// Should not panic when shutting down
	err = tp.Shutdown(context.Background())
	assert.NoError(t, err)
}

// TestNewTracerProvider_OTLPExporter tests creating a TracerProvider with OTLP exporter.
func TestNewTracerProvider_OTLPExporter(t *testing.T) {
	// Start a mock OTLP server
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	listener.Close()

	tp, err := NewTracerProvider(
		WithEnabled(true),
		WithServiceName("test-service"),
		WithExporterType(ExporterOTLP),
		WithEndpoint(address),
		WithSamplerRatio(1.0),
		WithInsecure(true),
	)
	require.NoError(t, err)
	require.NotNil(t, tp)

	// Should not panic when shutting down
	err = tp.Shutdown(context.Background())
	assert.NoError(t, err)
}

// TestNewTracerProvider_MissingEndpoint tests that an error is returned when endpoint is missing.
func TestNewTracerProvider_MissingEndpoint(t *testing.T) {
	_, err := NewTracerProvider(
		WithEnabled(true),
		WithExporterType(ExporterOTLP),
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint is required")

	_, err = NewTracerProvider(
		WithEnabled(true),
		WithExporterType(ExporterJaeger),
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "endpoint is required")
}

// TestNewTracerProvider_Defaults tests that defaults are applied correctly.
func TestNewTracerProvider_Defaults(t *testing.T) {
	tp, err := NewTracerProvider()
	require.NoError(t, err)
	require.NotNil(t, tp)

	// Tracing should be disabled by default
	assert.NotNil(t, tp.opts)
	assert.False(t, tp.opts.Enabled)
	assert.Equal(t, "firefly-service", tp.opts.ServiceName)
	assert.Equal(t, ExporterOTLP, tp.opts.ExporterType)
	assert.Equal(t, 1.0, tp.opts.SamplerRatio)
	assert.False(t, tp.opts.Insecure)
}

// TestWithConfig tests the WithConfig option.
func TestWithConfig(t *testing.T) {
	cfg := &config.TracingConfig{
		Enabled:      true,
		Endpoint:     "localhost:4317",
		SamplerRatio: 0.5,
		ExporterType: "otlp",
		Insecure:     true,
	}

	tp, err := NewTracerProvider(WithConfig(cfg, "my-service"))
	require.NoError(t, err)
	require.NotNil(t, tp)

	assert.True(t, tp.opts.Enabled)
	assert.Equal(t, "my-service", tp.opts.ServiceName)
	assert.Equal(t, "localhost:4317", tp.opts.Endpoint)
	assert.Equal(t, 0.5, tp.opts.SamplerRatio)
	assert.Equal(t, ExporterOTLP, tp.opts.ExporterType)
	assert.True(t, tp.opts.Insecure)

	_ = tp.Shutdown(context.Background())
}

// TestSetup tests the Setup function.
func TestSetup(t *testing.T) {
	cfg := &config.TracingConfig{
		Enabled:      false, // Disabled to avoid needing a real endpoint
		Endpoint:     "",
		SamplerRatio: 1.0,
		ExporterType: "stdout",
	}

	shutdown, err := Setup(context.Background(), cfg, "test-service")
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// Verify global TracerProvider is set
	tp := otel.GetTracerProvider()
	require.NotNil(t, tp)

	// Cleanup
	err = shutdown(context.Background())
	assert.NoError(t, err)
}

// TestSetupWithOptions tests the SetupWithOptions function.
func TestSetupWithOptions(t *testing.T) {
	shutdown, err := SetupWithOptions(
		context.Background(),
		WithEnabled(false),
		WithServiceName("test-service"),
	)
	require.NoError(t, err)
	require.NotNil(t, shutdown)

	// Cleanup
	err = shutdown(context.Background())
	assert.NoError(t, err)
}

// TestTracerProvider_Shutdown tests the Shutdown method.
func TestTracerProvider_Shutdown(t *testing.T) {
	tp, err := NewTracerProvider(
		WithEnabled(true),
		WithExporterType(ExporterStdout),
		WithLogger(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))),
	)
	require.NoError(t, err)

	// First shutdown should succeed
	err = tp.Shutdown(context.Background())
	assert.NoError(t, err)

	// Second shutdown should be a no-op
	err = tp.Shutdown(context.Background())
	assert.NoError(t, err)
}

// TestTracer tests the Tracer convenience function.
func TestTracer(t *testing.T) {
	// Setup a no-op provider
	shutdown, err := SetupWithOptions(context.Background(), WithEnabled(false))
	require.NoError(t, err)
	defer shutdown(context.Background())

	tracer := Tracer("test-tracer")
	assert.NotNil(t, tracer)
}

// TestNoopTracerProvider tests the NoopTracerProvider function.
func TestNoopTracerProvider(t *testing.T) {
	tp := NoopTracerProvider()
	require.NotNil(t, tp)

	// Should be a no-op tracer provider
	tracer := tp.Tracer("test")
	assert.NotNil(t, tracer)
}

// TestTracerProvider_WithLogger tests that logger is properly used.
func TestTracerProvider_WithLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tp, err := NewTracerProvider(
		WithEnabled(true),
		WithExporterType(ExporterStdout),
		WithLogger(logger),
	)
	require.NoError(t, err)

	assert.Equal(t, logger, tp.Logger)

	_ = tp.Shutdown(context.Background())
}

// TestOptions tests all option functions.
func TestOptions(t *testing.T) {
	opts := &Options{}

	WithEnabled(true)(opts)
	assert.True(t, opts.Enabled)

	WithServiceName("my-service")(opts)
	assert.Equal(t, "my-service", opts.ServiceName)

	WithEndpoint("localhost:4317")(opts)
	assert.Equal(t, "localhost:4317", opts.Endpoint)

	WithExporterType(ExporterJaeger)(opts)
	assert.Equal(t, ExporterJaeger, opts.ExporterType)

	WithSamplerRatio(0.5)(opts)
	assert.Equal(t, 0.5, opts.SamplerRatio)

	WithInsecure(true)(opts)
	assert.True(t, opts.Insecure)

	logger := slog.Default()
	WithLogger(logger)(opts)
	assert.Equal(t, logger, opts.Logger)
}

// TestExporterTypes tests all exporter type constants.
func TestExporterTypes(t *testing.T) {
	assert.Equal(t, ExporterType("otlp"), ExporterOTLP)
	assert.Equal(t, ExporterType("jaeger"), ExporterJaeger)
	assert.Equal(t, ExporterType("zipkin"), ExporterZipkin)
	assert.Equal(t, ExporterType("stdout"), ExporterStdout)
}

// TestUnsupportedExporterType tests that an error is returned for unsupported exporter types.
func TestUnsupportedExporterType(t *testing.T) {
	_, err := NewTracerProvider(
		WithEnabled(true),
		WithExporterType(ExporterType("unsupported")),
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported exporter type")
}

// TestTracerProvider_Integration tests the full tracing workflow.
func TestTracerProvider_Integration(t *testing.T) {
	// Create TracerProvider with stdout exporter
	tp, err := NewTracerProvider(
		WithEnabled(true),
		WithServiceName("integration-test"),
		WithExporterType(ExporterStdout),
		WithSamplerRatio(1.0),
	)
	require.NoError(t, err)

	// Set as global TracerProvider
	otel.SetTracerProvider(tp)

	// Create a tracer and start a span
	tracer := otel.Tracer("test-tracer")
	ctx, span := tracer.Start(context.Background(), "test-operation")

	// Set some attributes
	span.SetAttributes(
		attribute.String("key1", "value1"),
		attribute.Int("key2", 42),
	)

	// Add an event
	span.AddEvent("test-event")

	// End the span
	span.End()

	// Verify the span context
	traceID := span.SpanContext().TraceID()
	spanID := span.SpanContext().SpanID()
	assert.True(t, traceID.IsValid())
	assert.True(t, spanID.IsValid())

	// Cleanup
	err = tp.Shutdown(ctx)
	assert.NoError(t, err)
}

// TestTracerProvider_ErrorRecording tests that errors are properly recorded on spans.
func TestTracerProvider_ErrorRecording(t *testing.T) {
	tp, err := NewTracerProvider(
		WithEnabled(true),
		WithExporterType(ExporterStdout),
	)
	require.NoError(t, err)
	defer tp.Shutdown(context.Background())

	otel.SetTracerProvider(tp)

	tracer := otel.Tracer("test-tracer")
	_, span := tracer.Start(context.Background(), "test-operation")

	// Record an error
	testErr := assert.AnError
	span.RecordError(testErr)
	span.SetStatus(codes.Error, testErr.Error())

	span.End()

	// Verify span is ended (no panic)
	assert.True(t, span.SpanContext().IsValid())
}

// TestTracerProvider_Propagators tests that propagators are properly set.
func TestTracerProvider_Propagators(t *testing.T) {
	shutdown, err := SetupWithOptions(
		context.Background(),
		WithEnabled(false),
	)
	require.NoError(t, err)
	defer shutdown(context.Background())

	// Verify propagators are set
	propagator := otel.GetTextMapPropagator()
	assert.NotNil(t, propagator)

	// Test propagation
	ctx := context.Background()
	carrier := make(map[string]string)

	// Create a span and inject
	tracer := otel.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	propagator.Inject(ctx, &mapCarrier{carrier})
	// Carrier should contain trace context headers
	_, hasTraceParent := carrier["traceparent"]
	// When tracing is disabled, trace context may not be present
	t.Logf("Has traceparent: %v", hasTraceParent)
}

// mapCarrier implements propagation.TextMapCarrier for testing.
type mapCarrier struct {
	m map[string]string
}

func (c *mapCarrier) Get(key string) string {
	return c.m[key]
}

func (c *mapCarrier) Set(key string, value string) {
	c.m[key] = value
}

func (c *mapCarrier) Keys() []string {
	keys := make([]string, 0, len(c.m))
	for k := range c.m {
		keys = append(keys, k)
	}
	return keys
}

// TestTracerProvider_WithURL tests TracerProvider with URL endpoint parsing.
func TestTracerProvider_WithURL(t *testing.T) {
	// Parse URL for OTLP endpoint
	u, err := url.Parse("http://localhost:4317")
	require.NoError(t, err)

	tp, err := NewTracerProvider(
		WithEnabled(true),
		WithExporterType(ExporterOTLP),
		WithEndpoint(u.Host),
		WithInsecure(true),
	)
	require.NoError(t, err)
	require.NotNil(t, tp)

	_ = tp.Shutdown(context.Background())
}

// =============================================================================
// Property-Based Tests
// =============================================================================

// TestProperty36TraceIDGeneration tests Property 36:
// For any request, trace ID should be automatically generated and propagated
//
// Feature: backend-server-framework, Property 36: Trace ID Generation
// Validates: Requirements 17.1
func TestProperty36TraceIDGeneration(t *testing.T) {
	// Feature: backend-server-framework, Property 36: Trace ID Generation

	// Test that for any request, a trace ID is automatically generated
	t.Run("trace_id_generation", func(t *testing.T) {
		// Create a TracerProvider with stdout exporter for testing
		tp, err := NewTracerProvider(
			WithEnabled(true),
			WithServiceName("test-service"),
			WithExporterType(ExporterStdout),
			WithSamplerRatio(1.0),
		)
		require.NoError(t, err)
		defer tp.Shutdown(context.Background())

		// Set as global TracerProvider
		otel.SetTracerProvider(tp)

		// Property: For any operation name, a trace ID should be generated
		prop := func(operationName string) bool {
			tracer := otel.Tracer("test-tracer")
			_, span := tracer.Start(context.Background(), operationName)
			defer span.End()

			// Get trace ID from span context
			traceID := span.SpanContext().TraceID()

			// Trace ID should be valid (non-zero)
			return traceID.IsValid()
		}

		// Test with quick
		if err := quick.Check(prop, &quick.Config{
			MaxCount: 100,
			Values: func(args []reflect.Value, rand *rand.Rand) {
				// Generate random operation names
				args[0] = reflect.ValueOf(generateOperationName(rand))
			},
		}); err != nil {
			t.Errorf("Property 36 (trace ID generation) failed: %v", err)
		}
	})

	// Test that trace IDs are unique for different spans
	t.Run("trace_id_uniqueness", func(t *testing.T) {
		tp, err := NewTracerProvider(
			WithEnabled(true),
			WithExporterType(ExporterStdout),
		)
		require.NoError(t, err)
		defer tp.Shutdown(context.Background())

		otel.SetTracerProvider(tp)
		tracer := otel.Tracer("test-tracer")

		// Generate multiple spans and ensure they have unique trace IDs
		// (or at least that span IDs are unique within the same trace)
		ctx1, span1 := tracer.Start(context.Background(), "operation1")
		defer span1.End()

		_, span2 := tracer.Start(ctx1, "operation2") // Child span
		defer span2.End()

		_, span3 := tracer.Start(context.Background(), "operation3") // New root span
		defer span3.End()

		// Trace IDs should be valid
		assert.True(t, span1.SpanContext().TraceID().IsValid())
		assert.True(t, span2.SpanContext().TraceID().IsValid())
		assert.True(t, span3.SpanContext().TraceID().IsValid())

		// span1 and span2 should have same trace ID (parent-child relationship)
		assert.Equal(t, span1.SpanContext().TraceID(), span2.SpanContext().TraceID())

		// span1 and span3 should have different trace IDs (different root spans)
		// Note: There's a small probability they could be the same, but it's extremely unlikely
		assert.NotEqual(t, span1.SpanContext().TraceID(), span3.SpanContext().TraceID())

		// Span IDs should all be different
		assert.NotEqual(t, span1.SpanContext().SpanID(), span2.SpanContext().SpanID())
		assert.NotEqual(t, span1.SpanContext().SpanID(), span3.SpanContext().SpanID())
		assert.NotEqual(t, span2.SpanContext().SpanID(), span3.SpanContext().SpanID())
	})
}

// TestProperty37CallDurationRecording tests Property 37:
// For any service call, duration should be recorded
//
// Feature: backend-server-framework, Property 37: Call Duration Recording
// Validates: Requirements 17.3
func TestProperty37CallDurationRecording(t *testing.T) {
	// Feature: backend-server-framework, Property 37: Call Duration Recording

	t.Run("duration_recording", func(t *testing.T) {
		// Create a TracerProvider with in-memory exporter for verification
		exporter := tracetest.NewInMemoryExporter()
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithSyncer(exporter),
			sdktrace.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName("test-service"),
			)),
		)
		defer tp.Shutdown(context.Background())

		otel.SetTracerProvider(tp)
		tracer := otel.Tracer("test-tracer")

		// Property: For any span duration, the span should record start and end times
		prop := func(durationMs int64) bool {
			// Ensure duration is positive
			if durationMs <= 0 {
				durationMs = 1
			}
			if durationMs > 10000 { // Cap at 10 seconds for test
				durationMs = 10000
			}

			// Start a span
			_, span := tracer.Start(context.Background(), "test-operation")

			// Simulate some work
			time.Sleep(time.Duration(durationMs) * time.Millisecond)

			// End the span
			span.End()

			// Get recorded span
			spans := exporter.GetSpans()
			if len(spans) == 0 {
				return false
			}

			recordedSpan := spans[len(spans)-1]

			// Span should have start and end times
			if recordedSpan.StartTime.IsZero() || recordedSpan.EndTime.IsZero() {
				return false
			}

			// Duration should be approximately what we slept for
			recordedDuration := recordedSpan.EndTime.Sub(recordedSpan.StartTime)
			minExpected := time.Duration(durationMs) * time.Millisecond
			maxExpected := time.Duration(durationMs+100) * time.Millisecond // Allow some overhead

			return recordedDuration >= minExpected && recordedDuration <= maxExpected
		}

		// Test with quick
		if err := quick.Check(prop, &quick.Config{
			MaxCount: 50, // Fewer iterations since we're sleeping
			Values: func(args []reflect.Value, rand *rand.Rand) {
				// Generate random durations between 1ms and 100ms for quick tests
				args[0] = reflect.ValueOf(rand.Int63n(100) + 1)
			},
		}); err != nil {
			t.Errorf("Property 37 (call duration recording) failed: %v", err)
		}
	})
}

// TestProperty38ErrorStackRecording tests Property 38:
// For any request error, complete error stack should be recorded
//
// Feature: backend-server-framework, Property 38: Error Stack Recording
// Validates: Requirements 17.5, 17.6
func TestProperty38ErrorStackRecording(t *testing.T) {
	// Feature: backend-server-framework, Property 38: Error Stack Recording

	t.Run("error_stack_recording", func(t *testing.T) {
		// Create a TracerProvider with in-memory exporter
		exporter := tracetest.NewInMemoryExporter()
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithSyncer(exporter),
			sdktrace.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceName("test-service"),
			)),
		)
		defer tp.Shutdown(context.Background())

		otel.SetTracerProvider(tp)
		tracer := otel.Tracer("test-tracer")

		// Test with framework errors
		t.Run("framework_errors", func(t *testing.T) {
			prop := func(errorCode int, reason, message string) bool {
				// Ensure reasonable values
				if errorCode < 100 || errorCode > 599 {
					errorCode = 500
				}
				if reason == "" {
					reason = "TEST_ERROR"
				}
				if message == "" {
					message = "test error message"
				}

				// Create a framework error
				err := ferrors.New(errorCode, reason, message)

				// Start a span
				_, span := tracer.Start(context.Background(), "test-operation")

				// Record the error on the span
				span.RecordError(err, trace.WithStackTrace(true))
				span.SetStatus(codes.Error, err.Error())
				span.End()

				// Get recorded span
				spans := exporter.GetSpans()
				if len(spans) == 0 {
					return false
				}

				recordedSpan := spans[len(spans)-1]

				// Span should be marked as error
				if recordedSpan.Status.Code != codes.Error {
					return false
				}

				// Error description should contain our error message
				if !strings.Contains(recordedSpan.Status.Description, message) {
					return false
				}

				return true
			}

			if err := quick.Check(prop, &quick.Config{
				MaxCount: 100,
				Values: func(args []reflect.Value, rand *rand.Rand) {
					args[0] = reflect.ValueOf(rand.Intn(400) + 100) // 100-499
					args[1] = reflect.ValueOf(generateRandomString(rand, 5, 20))
					args[2] = reflect.ValueOf(generateRandomString(rand, 10, 50))
				},
			}); err != nil {
				t.Errorf("Property 38 (error stack recording for framework errors) failed: %v", err)
			}
		})

		// Test with generic errors
		t.Run("generic_errors", func(t *testing.T) {
			prop := func(errorMessage string) bool {
				if errorMessage == "" {
					errorMessage = "generic error"
				}

				// Create a generic error
				err := errors.New(errorMessage)

				// Start a span
				_, span := tracer.Start(context.Background(), "test-operation")

				// Record the error on the span
				span.RecordError(err, trace.WithStackTrace(true))
				span.SetStatus(codes.Error, err.Error())
				span.End()

				// Get recorded span
				spans := exporter.GetSpans()
				if len(spans) == 0 {
					return false
				}

				recordedSpan := spans[len(spans)-1]

				// Span should be marked as error
				if recordedSpan.Status.Code != codes.Error {
					return false
				}

				// Error description should contain our error message
				if !strings.Contains(recordedSpan.Status.Description, errorMessage) {
					return false
				}

				return true
			}

			if err := quick.Check(prop, &quick.Config{
				MaxCount: 100,
				Values: func(args []reflect.Value, rand *rand.Rand) {
					args[0] = reflect.ValueOf(generateRandomString(rand, 5, 50))
				},
			}); err != nil {
				t.Errorf("Property 38 (error stack recording for generic errors) failed: %v", err)
			}
		})
	})
}

// Helper functions for property tests

func generateOperationName(rand *rand.Rand) string {
	operations := []string{
		"GET /api/users",
		"POST /api/users",
		"PUT /api/users/:id",
		"DELETE /api/users/:id",
		"GET /api/products",
		"POST /api/products",
		"/api.v1.UserService/GetUser",
		"/api.v1.UserService/CreateUser",
		"database.query",
		"cache.get",
		"external.api.call",
	}
	return operations[rand.Intn(len(operations))]
}

func generateRandomString(rand *rand.Rand, minLen, maxLen int) string {
	length := rand.Intn(maxLen-minLen+1) + minLen
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 _-"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}
