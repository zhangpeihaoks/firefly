package observability

import (
	"context"
	"testing"

	"github.com/zhangpeihaoks/firefly/internal/tracing"
)

func TestInit_DisabledTracing(t *testing.T) {
	shutdown, err := Init(context.Background(), Config{
		ServiceName: "test-svc",
	})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer shutdown()

	// No tracer configured: convenience APIs return empty IDs.
	if id := TraceID(context.Background()); id != "" {
		t.Errorf("expected empty trace ID without span, got %q", id)
	}
	if id := SpanID(context.Background()); id != "" {
		t.Errorf("expected empty span ID without span, got %q", id)
	}
}

func TestInit_StdoutTracer(t *testing.T) {
	// stdout exporter avoids external dependencies while exercising the
	// full provider setup path.
	shutdown, err := Init(context.Background(), Config{
		ServiceName: "test-svc",
		Tracing: TracingConfig{
			Enabled:      true,
			ExporterType: tracing.ExporterStdout,
		},
	})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer shutdown()

	ctx, span := StartSpan(context.Background(), "test-operation")
	defer span.End()

	traceID := TraceID(ctx)
	if traceID == "" {
		t.Error("expected a trace ID from the global tracer")
	}
	if spanID := SpanID(ctx); spanID == "" {
		t.Error("expected a span ID from the global tracer")
	}

	// A child span inherits the parent trace.
	childCtx, child := StartSpan(ctx, "child")
	defer child.End()
	if childID := TraceID(childCtx); childID != traceID {
		t.Errorf("expected child to share trace %s, got %s", traceID, childID)
	}
}
