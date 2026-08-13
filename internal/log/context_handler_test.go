package log

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/zhangpeihaoks/firefly/internal/middleware"
	"go.opentelemetry.io/otel/trace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// newTestLogger builds a logger over a buffer through ContextHandler.
func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	handler := NewContextHandler(slog.NewTextHandler(buf, nil))
	return slog.New(handler)
}

func TestContextHandler_NoSpanNoTraceID(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	logger.InfoContext(context.Background(), "no span here")

	if strings.Contains(buf.String(), "trace_id") {
		t.Errorf("unexpected trace_id without an active span: %s", buf.String())
	}
}

func TestContextHandler_InjectTraceID(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("test-tracer")
	ctx, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	logger.InfoContext(ctx, "inside a span")

	out := buf.String()
	if !strings.Contains(out, "trace_id=") {
		t.Errorf("expected trace_id in log output, got: %s", out)
	}
	if !strings.Contains(out, "span_id=") {
		t.Errorf("expected span_id in log output, got: %s", out)
	}
}

func TestContextHandler_InjectRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	ctx := middleware.NewContextWithRequestID(context.Background(), "req-12345")
	logger.InfoContext(ctx, "with request id")

	if !strings.Contains(buf.String(), "request_id=req-12345") {
		t.Errorf("expected request_id in log output, got: %s", buf.String())
	}
}

func TestContextHandler_BothCorrelationIDs(t *testing.T) {
	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	tp := sdktrace.NewTracerProvider()
	ctx, span := tp.Tracer("test").Start(
		middleware.NewContextWithRequestID(context.Background(), "req-abc"),
		"op",
	)
	defer span.End()

	logger.InfoContext(ctx, "full correlation")

	out := buf.String()
	for _, want := range []string{"request_id=req-abc", "trace_id=", "span_id="} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in log output, got: %s", want, out)
		}
	}
}

func TestContextHandler_DelegatesToInner(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError})
	h := NewContextHandler(inner)

	if h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("expected Info to be filtered by inner handler")
	}
	if !h.Enabled(context.Background(), slog.LevelError) {
		t.Error("expected Error to be enabled")
	}

	// WithAttrs/WithGroup delegation preserves behavior.
	_ = h.WithAttrs([]slog.Attr{slog.String("k", "v")})
	_ = h.WithGroup("g")
	_ = trace.SpanFromContext(context.Background()) // ensure import is used
}
