// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/metrics"
	"github.com/zhangpeihaoks/firefly/internal/transport"
)

const (
	// namespace is the Prometheus namespace for all metrics.
	namespace = "firefly"
	// subsystem is the Prometheus subsystem for HTTP metrics.
	subsystem = "http"
)

// DefaultBuckets are the default histogram buckets for request latency.
// They cover a range from 1ms to 10s.
var DefaultBuckets = []float64{
	0.001, // 1ms
	0.005, // 5ms
	0.01,  // 10ms
	0.025, // 25ms
	0.05,  // 50ms
	0.1,   // 100ms
	0.25,  // 250ms
	0.5,   // 500ms
	1.0,   // 1s
	2.5,   // 2.5s
	5.0,   // 5s
	10.0,  // 10s
}

// metricsOptions holds the configuration for Metrics middleware.
type metricsOptions struct {
	registry   prometheus.Registerer
	buckets    []float64
	counter    *prometheus.CounterVec
	latency    *prometheus.HistogramVec
	errorCount *prometheus.CounterVec
}

// MetricsOption is a configuration option for Metrics middleware.
type MetricsOption func(*metricsOptions)

// WithRegistry sets a custom Prometheus registry for the Metrics middleware.
// If not set, the default Prometheus registry is used.
func WithRegistry(r prometheus.Registerer) MetricsOption {
	return func(o *metricsOptions) {
		o.registry = r
	}
}

// WithBuckets sets custom histogram buckets for request latency.
// If not set, DefaultBuckets are used.
func WithBuckets(buckets []float64) MetricsOption {
	return func(o *metricsOptions) {
		o.buckets = buckets
	}
}

// WithCounter sets a custom request counter metric.
// This allows for custom labeling or metric naming.
func WithCounter(counter *prometheus.CounterVec) MetricsOption {
	return func(o *metricsOptions) {
		o.counter = counter
	}
}

// WithLatency sets a custom request latency histogram metric.
// This allows for custom labeling or metric naming.
func WithLatency(latency *prometheus.HistogramVec) MetricsOption {
	return func(o *metricsOptions) {
		o.latency = latency
	}
}

// WithErrorCount sets a custom error counter metric.
// This allows for custom labeling or metric naming.
func WithErrorCount(errorCount *prometheus.CounterVec) MetricsOption {
	return func(o *metricsOptions) {
		o.errorCount = errorCount
	}
}

// Metrics returns a middleware that collects Prometheus metrics for requests.
// It records:
//   - Request count by method, path, and status code
//   - Request latency histogram by method and path
//   - Error count by method, path, and error type
//
// The middleware uses the following labels:
//   - kind: transport type (http/grpc)
//   - method: HTTP method or gRPC method
//   - path: request path or operation
//   - status: HTTP status code
//   - code: error code (for error counter)
//
// Example:
//
//	// With default options
//	middleware.Metrics()
//
//	// With custom buckets
//	middleware.Metrics(middleware.WithBuckets([]float64{0.1, 0.5, 1.0, 5.0}))
//
//	// With custom registry
//	middleware.Metrics(middleware.WithRegistry(myRegistry))
func Metrics(opts ...MetricsOption) Middleware {
	// Apply default options
	options := &metricsOptions{
		registry: metrics.DefaultRegistry(),
		buckets:  DefaultBuckets,
	}

	// Apply custom options
	for _, opt := range opts {
		opt(options)
	}

	// Create metrics if not provided
	factory := promauto.With(options.registry)

	if options.counter == nil {
		options.counter = factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "requests_total",
				Help:      "Total number of requests by method, path, and status.",
			},
			[]string{"kind", "method", "path", "status"},
		)
	}

	if options.latency == nil {
		options.latency = factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_duration_seconds",
				Help:      "Request latency in seconds by method and path.",
				Buckets:   options.buckets,
			},
			[]string{"kind", "method", "path"},
		)
	}

	if options.errorCount == nil {
		options.errorCount = factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "errors_total",
				Help:      "Total number of errors by method, path, and error code.",
			},
			[]string{"kind", "method", "path", "code"},
		)
	}

	return func(next Handler) Handler {
		return func(ctx context.Context, req any) (any, error) {
			// Record start time
			startTime := time.Now()

			// Get transport info from context
			tr := transport.FromContext(ctx)

			// Extract labels
			var kind, method, path string
			if tr != nil {
				kind = string(tr.Kind())
				method = tr.Operation()
				path = tr.Operation()
			} else {
				kind = "unknown"
				method = "unknown"
				path = "unknown"
			}

			// Call the next handler
			resp, err := next(ctx, req)

			// Calculate latency
			latency := time.Since(startTime).Seconds()

			// Determine status code
			statusCode := "200"
			if err != nil {
				statusCode = "500"
				if e, ok := err.(*errors.Error); ok {
					statusCode = e.Reason
				}
			}

			// Record metrics
			options.counter.WithLabelValues(kind, method, path, statusCode).Inc()
			options.latency.WithLabelValues(kind, method, path).Observe(latency)

			// Record error if occurred
			if err != nil {
				errorCode := "unknown"
				if e, ok := err.(*errors.Error); ok {
					errorCode = e.Reason
				}
				options.errorCount.WithLabelValues(kind, method, path, errorCode).Inc()
			}

			return resp, err
		}
	}
}
