// Package tracing provides OpenTelemetry tracing initialization for the Firefly framework.
// It supports multiple exporters (OTLP, Jaeger, stdout) and provides configuration-based setup.
package tracing

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/zhangpeihaoks/firefly/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// ExporterType defines the type of trace exporter.
type ExporterType string

const (
	// ExporterOTLP uses OTLP protocol for trace export.
	ExporterOTLP ExporterType = "otlp"
	// ExporterJaeger uses Jaeger exporter for trace export.
	ExporterJaeger ExporterType = "jaeger"
	// ExporterZipkin uses Zipkin exporter for trace export.
	ExporterZipkin ExporterType = "zipkin"
	// ExporterStdout writes traces to stdout for testing.
	ExporterStdout ExporterType = "stdout"
)

// Options holds the configuration for tracing initialization.
type Options struct {
	// Enabled determines if tracing is enabled.
	Enabled bool
	// ServiceName is the name of the service.
	ServiceName string
	// Endpoint is the tracing endpoint (e.g., "localhost:4317" for OTLP).
	Endpoint string
	// ExporterType specifies the type of exporter to use.
	ExporterType ExporterType
	// SamplerRatio is the sampling ratio (0.0 to 1.0).
	SamplerRatio float64
	// Insecure determines whether to use insecure connection for OTLP.
	Insecure bool
	// Logger is the logger to use for tracing-related logs.
	Logger *slog.Logger
}

// Option is a configuration option function for tracing initialization.
type Option func(*Options)

// WithEnabled enables or disables tracing.
func WithEnabled(enabled bool) Option {
	return func(o *Options) {
		o.Enabled = enabled
	}
}

// WithServiceName sets the service name for tracing.
func WithServiceName(name string) Option {
	return func(o *Options) {
		o.ServiceName = name
	}
}

// WithEndpoint sets the tracing endpoint.
func WithEndpoint(endpoint string) Option {
	return func(o *Options) {
		o.Endpoint = endpoint
	}
}

// WithExporterType sets the exporter type.
func WithExporterType(t ExporterType) Option {
	return func(o *Options) {
		o.ExporterType = t
	}
}

// WithSamplerRatio sets the sampling ratio.
func WithSamplerRatio(ratio float64) Option {
	return func(o *Options) {
		o.SamplerRatio = ratio
	}
}

// WithInsecure sets whether to use insecure connection.
func WithInsecure(insecure bool) Option {
	return func(o *Options) {
		o.Insecure = insecure
	}
}

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) Option {
	return func(o *Options) {
		o.Logger = logger
	}
}

// WithConfig creates options from a TracingConfig.
func WithConfig(cfg *config.TracingConfig, serviceName string) Option {
	return func(o *Options) {
		o.Enabled = cfg.Enabled
		o.Endpoint = cfg.Endpoint
		o.SamplerRatio = cfg.SamplerRatio
		o.ExporterType = ExporterType(cfg.ExporterType)
		o.ServiceName = serviceName
		o.Insecure = cfg.Insecure
	}
}

// TracerProvider wraps the OpenTelemetry TracerProvider with shutdown capability.
type TracerProvider struct {
	*sdktrace.TracerProvider
	exporter sdktrace.SpanExporter
	opts     *Options
	Logger   *slog.Logger
	mu       sync.Mutex
	stopped  bool
}

// Shutdown gracefully shuts down the TracerProvider.
// It flushes any remaining spans and releases resources.
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	if tp.stopped {
		return nil
	}
	tp.stopped = true

	logger := tp.Logger
	if logger == nil {
		logger = slog.Default()
	}

	logger.Info("shutting down tracer provider")

	// Shutdown the TracerProvider
	if err := tp.TracerProvider.Shutdown(ctx); err != nil {
		logger.Error("failed to shutdown tracer provider", "error", err)
		return fmt.Errorf("tracing: failed to shutdown tracer provider: %w", err)
	}

	logger.Info("tracer provider shutdown complete")

	return nil
}

// NewTracerProvider creates a new TracerProvider with the given options.
// It initializes the appropriate exporter based on the configuration.
//
// Example:
//
//	// Create TracerProvider with OTLP exporter
//	tp, err := tracing.NewTracerProvider(
//	    tracing.WithServiceName("my-service"),
//	    tracing.WithEndpoint("localhost:4317"),
//	    tracing.WithExporterType(tracing.ExporterOTLP),
//	    tracing.WithSamplerRatio(1.0),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer tp.Shutdown(context.Background())
//
//	// Set as global TracerProvider
//	otel.SetTracerProvider(tp)
func NewTracerProvider(opts ...Option) (*TracerProvider, error) {
	options := &Options{
		Enabled:      false,
		ServiceName:  "firefly-service",
		ExporterType: ExporterOTLP,
		SamplerRatio: 1.0,
		Insecure:     false,
		Logger:       slog.Default(),
	}

	for _, opt := range opts {
		opt(options)
	}

	// If tracing is disabled, return a no-op TracerProvider
	if !options.Enabled {
		if options.Logger != nil {
			options.Logger.Info("tracing is disabled, using no-op tracer provider")
		}
		return &TracerProvider{
			TracerProvider: sdktrace.NewTracerProvider(),
			opts:           options,
			Logger:         options.Logger,
		}, nil
	}

	// Create the exporter
	exporter, err := createExporter(options)
	if err != nil {
		return nil, fmt.Errorf("tracing: failed to create exporter: %w", err)
	}

	// Create the resource with service information
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(options.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: failed to create resource: %w", err)
	}

	// Create the sampler
	sampler := sdktrace.TraceIDRatioBased(options.SamplerRatio)

	// Create the TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	if options.Logger != nil {
		options.Logger.Info("tracer provider initialized",
			"service_name", options.ServiceName,
			"exporter_type", options.ExporterType,
			"endpoint", options.Endpoint,
			"sampler_ratio", options.SamplerRatio,
		)
	}

	return &TracerProvider{
		TracerProvider: tp,
		exporter:       exporter,
		opts:           options,
		Logger:         options.Logger,
	}, nil
}

// createExporter creates a trace exporter based on the configuration.
func createExporter(opts *Options) (sdktrace.SpanExporter, error) {
	ctx := context.Background()

	switch opts.ExporterType {
	case ExporterOTLP:
		return createOTLPExporter(ctx, opts)
	case ExporterJaeger:
		return createJaegerExporter(opts)
	case ExporterZipkin:
		return createZipkinExporter(opts)
	case ExporterStdout:
		return createStdoutExporter(opts)
	default:
		return nil, fmt.Errorf("tracing: unsupported exporter type: %s", opts.ExporterType)
	}
}

// createOTLPExporter creates an OTLP trace exporter.
// It uses gRPC by default, falling back to HTTP if gRPC fails.
func createOTLPExporter(ctx context.Context, opts *Options) (sdktrace.SpanExporter, error) {
	if opts.Endpoint == "" {
		return nil, fmt.Errorf("tracing: endpoint is required for OTLP exporter")
	}

	// Try gRPC first
	if opts.Insecure {
		client := otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(opts.Endpoint),
			otlptracegrpc.WithInsecure(),
		)
		exporter, err := otlptrace.New(ctx, client)
		if err != nil {
			return nil, fmt.Errorf("tracing: failed to create OTLP gRPC exporter: %w", err)
		}
		return exporter, nil
	}

	// Use secure gRPC connection
	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(opts.Endpoint),
	)
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		// Fall back to HTTP if gRPC fails
		if opts.Logger != nil {
			opts.Logger.Debug("OTLP gRPC failed, trying HTTP", "error", err)
		}

		httpClient := otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(opts.Endpoint),
		)
		exporter, err = otlptrace.New(ctx, httpClient)
		if err != nil {
			return nil, fmt.Errorf("tracing: failed to create OTLP HTTP exporter: %w", err)
		}
	}

	return exporter, nil
}

// createJaegerExporter creates a Jaeger trace exporter.
// Note: Jaeger accepts OTLP protocol, so we use OTLP exporter for Jaeger.
func createJaegerExporter(opts *Options) (sdktrace.SpanExporter, error) {
	if opts.Endpoint == "" {
		return nil, fmt.Errorf("tracing: endpoint is required for Jaeger exporter")
	}

	// Jaeger accepts OTLP protocol, so we use OTLP exporter
	// Default Jaeger OTLP endpoint is typically localhost:4317
	ctx := context.Background()

	// For Jaeger, we can use gRPC or HTTP
	// Using gRPC by default as it's more efficient
	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(opts.Endpoint),
		otlptracegrpc.WithInsecure(), // Jaeger typically runs without TLS in development
	)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		// Fall back to HTTP if gRPC fails
		if opts.Logger != nil {
			opts.Logger.Debug("Jaeger OTLP gRPC failed, trying HTTP", "error", err)
		}

		httpClient := otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(opts.Endpoint),
		)
		exporter, err = otlptrace.New(ctx, httpClient)
		if err != nil {
			return nil, fmt.Errorf("tracing: failed to create Jaeger OTLP exporter: %w", err)
		}
	}

	if opts.Logger != nil {
		opts.Logger.Info("using Jaeger trace exporter (via OTLP)",
			"endpoint", opts.Endpoint)
	}

	return exporter, nil
}

// createZipkinExporter creates a Zipkin trace exporter.
// Note: Zipkin accepts OTLP protocol, so we use OTLP exporter for Zipkin.
func createZipkinExporter(opts *Options) (sdktrace.SpanExporter, error) {
	if opts.Endpoint == "" {
		return nil, fmt.Errorf("tracing: endpoint is required for Zipkin exporter")
	}

	// Zipkin accepts OTLP protocol, so we use OTLP exporter
	// Default Zipkin OTLP endpoint is typically http://localhost:9411/api/v2/spans
	// but OTLP uses different endpoint format
	ctx := context.Background()

	// For Zipkin, we typically use HTTP protocol
	client := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(opts.Endpoint),
		otlptracehttp.WithInsecure(), // Zipkin typically runs without TLS in development
	)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("tracing: failed to create Zipkin OTLP exporter: %w", err)
	}

	if opts.Logger != nil {
		opts.Logger.Info("using Zipkin trace exporter (via OTLP)",
			"endpoint", opts.Endpoint)
	}

	return exporter, nil
}

// createStdoutExporter creates a stdout trace exporter for testing.
func createStdoutExporter(opts *Options) (sdktrace.SpanExporter, error) {
	exporter, err := stdouttrace.New(
		stdouttrace.WithWriter(os.Stdout),
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: failed to create stdout exporter: %w", err)
	}

	if opts.Logger != nil {
		opts.Logger.Info("using stdout trace exporter (for testing only)")
	}

	return exporter, nil
}

// Setup initializes the global TracerProvider and propagators.
// It returns a shutdown function that should be called on application termination.
//
// Example:
//
//	shutdown, err := tracing.Setup(ctx, cfg, "my-service")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer shutdown()
func Setup(ctx context.Context, cfg *config.TracingConfig, serviceName string) (func(context.Context) error, error) {
	tp, err := NewTracerProvider(WithConfig(cfg, serviceName))
	if err != nil {
		return nil, err
	}

	// Set global TracerProvider
	otel.SetTracerProvider(tp)

	// Set global propagators for trace context propagation
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(propagator)

	// Return shutdown function
	return func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}, nil
}

// SetupWithOptions initializes the global TracerProvider with the given options.
// It returns a shutdown function that should be called on application termination.
func SetupWithOptions(ctx context.Context, opts ...Option) (func(context.Context) error, error) {
	tp, err := NewTracerProvider(opts...)
	if err != nil {
		return nil, err
	}

	// Set global TracerProvider
	otel.SetTracerProvider(tp)

	// Set global propagators for trace context propagation
	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(propagator)

	// Return shutdown function
	return func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}, nil
}

// Tracer returns a tracer with the given name from the global TracerProvider.
// This is a convenience function for otel.Tracer.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// NoopTracerProvider returns a no-op TracerProvider.
// Use this for testing or when tracing is disabled.
func NoopTracerProvider() trace.TracerProvider {
	return trace.NewNoopTracerProvider()
}
