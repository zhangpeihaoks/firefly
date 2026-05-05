// Package metrics provides Prometheus metrics collection and exposure for the Firefly framework.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	io_prometheus_client "github.com/prometheus/client_model/go"
)

// Registry is the global Prometheus registry.
var Registry = prometheus.NewRegistry()

// DefaultRegistry returns the default Prometheus registry.
func DefaultRegistry() *prometheus.Registry {
	return Registry
}

// Handler returns an HTTP handler for the Prometheus metrics endpoint.
func Handler() http.Handler {
	return promhttp.HandlerFor(Registry, promhttp.HandlerOpts{
		Registry:          Registry,
		EnableOpenMetrics: true,
	})
}

// Register registers a collector with the global registry.
func Register(c prometheus.Collector) error {
	return Registry.Register(c)
}

// MustRegister registers a collector with the global registry, panicking on error.
func MustRegister(c prometheus.Collector) {
	Registry.MustRegister(c)
}

// Unregister unregisters a collector from the global registry.
func Unregister(c prometheus.Collector) bool {
	return Registry.Unregister(c)
}

// Gather gathers metrics from the global registry.
func Gather() ([]*io_prometheus_client.MetricFamily, error) {
	return Registry.Gather()
}

// CustomMetric is a wrapper for custom metrics that can be registered.
type CustomMetric struct {
	collector prometheus.Collector
}

// NewCounter creates a new custom counter metric.
func NewCounter(opts prometheus.CounterOpts) *CustomMetric {
	counter := prometheus.NewCounter(opts)
	return &CustomMetric{collector: counter}
}

// NewCounterVec creates a new custom counter vector metric.
func NewCounterVec(opts prometheus.CounterOpts, labelNames []string) *CustomMetric {
	counter := prometheus.NewCounterVec(opts, labelNames)
	return &CustomMetric{collector: counter}
}

// NewGauge creates a new custom gauge metric.
func NewGauge(opts prometheus.GaugeOpts) *CustomMetric {
	gauge := prometheus.NewGauge(opts)
	return &CustomMetric{collector: gauge}
}

// NewGaugeVec creates a new custom gauge vector metric.
func NewGaugeVec(opts prometheus.GaugeOpts, labelNames []string) *CustomMetric {
	gauge := prometheus.NewGaugeVec(opts, labelNames)
	return &CustomMetric{collector: gauge}
}

// NewHistogram creates a new custom histogram metric.
func NewHistogram(opts prometheus.HistogramOpts) *CustomMetric {
	histogram := prometheus.NewHistogram(opts)
	return &CustomMetric{collector: histogram}
}

// NewHistogramVec creates a new custom histogram vector metric.
func NewHistogramVec(opts prometheus.HistogramOpts, labelNames []string) *CustomMetric {
	histogram := prometheus.NewHistogramVec(opts, labelNames)
	return &CustomMetric{collector: histogram}
}

// NewSummary creates a new custom summary metric.
func NewSummary(opts prometheus.SummaryOpts) *CustomMetric {
	summary := prometheus.NewSummary(opts)
	return &CustomMetric{collector: summary}
}

// NewSummaryVec creates a new custom summary vector metric.
func NewSummaryVec(opts prometheus.SummaryOpts, labelNames []string) *CustomMetric {
	summary := prometheus.NewSummaryVec(opts, labelNames)
	return &CustomMetric{collector: summary}
}

// Collector returns the underlying Prometheus collector.
func (m *CustomMetric) Collector() prometheus.Collector {
	return m.collector
}

// Register registers the custom metric with the global registry.
func (m *CustomMetric) Register() error {
	return Register(m.collector)
}

// MustRegister registers the custom metric with the global registry, panicking on error.
func (m *CustomMetric) MustRegister() {
	MustRegister(m.collector)
}

// Unregister unregisters the custom metric from the global registry.
func (m *CustomMetric) Unregister() bool {
	return Unregister(m.collector)
}

// Initialize initializes the metrics system with default metrics.
func Initialize() {
	// Register default Go metrics
	Registry.MustRegister(prometheus.NewGoCollector())
	Registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
}
