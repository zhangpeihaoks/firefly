// Package observability is the first-class observability entry point of the
// Firefly framework. It wires logging, distributed tracing and metrics into
// one config-driven SDK, so services get production observability with a
// single Init call.
//
// Key behaviors:
//   - Init configures slog (JSON, rotation) and the OpenTelemetry tracer
//     provider from one Config, returning a shutdown function.
//   - The slog handler automatically injects trace_id/span_id/request_id
//     from the request context (see internal/log.ContextHandler), so every
//     log line correlates with its trace.
//   - Convenience APIs (StartSpan/TraceID/SpanID) keep business code free
//     of OpenTelemetry imports.
package observability

import (
	"context"
	"log/slog"

	"github.com/zhangpeihaoks/firefly/internal/log"
	"github.com/zhangpeihaoks/firefly/internal/middleware"
	"github.com/zhangpeihaoks/firefly/internal/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// tracerName is used for the framework-level convenience tracer.
const tracerName = "github.com/zhangpeihaoks/firefly/observability"

// TracingConfig configures the OpenTelemetry tracer provider.
type TracingConfig struct {
	// Enabled enables distributed tracing (default false: no-op provider).
	Enabled bool
	// Endpoint is the trace exporter endpoint (e.g. OTLP "localhost:4317").
	Endpoint string
	// ExporterType selects the exporter: otlp (default), jaeger, zipkin, stdout.
	ExporterType tracing.ExporterType
	// SamplerRatio is the sampling ratio 0..1 (default 1.0).
	SamplerRatio float64
	// Insecure disables TLS for OTLP.
	Insecure bool
}

// Config configures the observability SDK.
type Config struct {
	// ServiceName is reported as the OpenTelemetry service.name resource.
	ServiceName string
	// Log configures the framework logger; nil keeps the current logger.
	Log *log.Config
	// Tracing configures distributed tracing.
	Tracing TracingConfig
}

// Observability is the initialized observability SDK handle. It carries the
// configured tracer provider and convenience methods for wiring servers.
type Observability struct {
	serviceName string
	shutdown    func()
}

// Init wires logging and distributed tracing from the config and installs
// them as globals. Call Shutdown on application exit (flush spans, close log
// file).
func Init(ctx context.Context, cfg Config) (*Observability, error) {
	// 1. Logger (JSON + rotation + context correlation).
	var logCleanup func()
	if cfg.Log != nil {
		logCleanup = log.New(cfg.Log)
	}

	// 2. Tracer provider (no-op when disabled).
	traceOpts := []tracing.Option{
		tracing.WithEnabled(cfg.Tracing.Enabled),
		tracing.WithServiceName(cfg.ServiceName),
		tracing.WithEndpoint(cfg.Tracing.Endpoint),
		tracing.WithExporterType(cfg.Tracing.ExporterType),
		tracing.WithInsecure(cfg.Tracing.Insecure),
	}
	// 0 means "use the provider default (1.0)"; explicit 0 would sample nothing.
	if cfg.Tracing.SamplerRatio > 0 {
		traceOpts = append(traceOpts, tracing.WithSamplerRatio(cfg.Tracing.SamplerRatio))
	}

	tp, err := tracing.NewTracerProvider(traceOpts...)
	if err != nil {
		if logCleanup != nil {
			logCleanup()
		}
		return nil, err
	}

	// Install globals so framework middleware and the convenience APIs below
	// pick them up automatically.
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	obs := &Observability{serviceName: cfg.ServiceName}
	obs.shutdown = func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			slog.Warn("observability: tracer shutdown failed", "error", err)
		}
		if logCleanup != nil {
			logCleanup()
		}
	}
	return obs, nil
}

// Shutdown flushes traces and closes the log file. Safe to call once.
func (o *Observability) Shutdown() {
	if o != nil && o.shutdown != nil {
		o.shutdown()
	}
}

// ServiceName returns the configured service name.
func (o *Observability) ServiceName() string {
	return o.serviceName
}

// HTTPMiddleware returns the default observability middleware chain for
// wiring into HTTP/gRPC servers (see middleware.DefaultObservabilityChain).
func (o *Observability) HTTPMiddleware() []middleware.Middleware {
	return middleware.DefaultObservabilityChain()
}

// StartSpan starts a span as a child of the span in ctx and returns a context
// carrying the new span. Use it in business code to create sub-spans; the
// span must be ended with defer span.End().
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, name, opts...)
}

// TraceID returns the trace ID of the span in ctx, or "" when no active span.
// The ID is reported regardless of sampling — local logs stay correlated with
// traces even when the trace is not exported.
func TraceID(ctx context.Context) string {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		return sc.TraceID().String()
	}
	return ""
}

// SpanID returns the span ID of the span in ctx, or "" when no active span.
func SpanID(ctx context.Context) string {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		return sc.SpanID().String()
	}
	return ""
}
