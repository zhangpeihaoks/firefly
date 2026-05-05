// Package metrics provides Prometheus metrics collection and exposure for the Firefly framework.
package metrics

import (
	"math/rand"
	"regexp"
	"testing"
	"testing/quick"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

// isValidMetricName checks if a metric name is valid according to Prometheus naming conventions.
func isValidMetricName(name string) bool {
	// Prometheus metric names must match [a-zA-Z_:][a-zA-Z0-9_:]*
	matched, _ := regexp.MatchString(`^[a-zA-Z_:][a-zA-Z0-9_:]*$`, name)
	return matched
}

// isValidLabelName checks if a label name is valid according to Prometheus naming conventions.
func isValidLabelName(name string) bool {
	// Prometheus label names must match [a-zA-Z_][a-zA-Z0-9_]*
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	return matched
}

// TestProperty39_MetricsCollection tests property 39: 指标收集
// For any request, metrics should automatically update request count, request latency, error rate, etc.
// **Validates: Requirements 18.2**
func TestProperty39_MetricsCollection(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(42)),
	}

	f := func(metricName string, helpText string, labelNames []string, labelValues []string, incrementValue float64) bool {
		// Reset global registry for test
		Registry = prometheus.NewRegistry()

		// Validate metric name (Prometheus naming convention)
		if !isValidMetricName(metricName) {
			return true // Skip invalid names
		}

		// Validate label names
		for _, label := range labelNames {
			if !isValidLabelName(label) {
				return true // Skip invalid label names
			}
		}

		// Ensure labelValues length matches labelNames length
		if len(labelValues) != len(labelNames) {
			// Adjust labelValues to match labelNames length
			if len(labelValues) > len(labelNames) {
				labelValues = labelValues[:len(labelNames)]
			} else {
				// Pad with empty strings
				for i := len(labelValues); i < len(labelNames); i++ {
					labelValues = append(labelValues, "")
				}
			}
		}

		// Create a counter vector metric
		counterOpts := prometheus.CounterOpts{
			Name: metricName,
			Help: helpText,
		}
		counter := prometheus.NewCounterVec(counterOpts, labelNames)

		// Register the metric
		err := Register(counter)
		if err != nil {
			t.Logf("failed to register metric: %v", err)
			return false
		}

		// Increment the metric with label values
		counter.WithLabelValues(labelValues...).Add(incrementValue)

		// Gather metrics
		metrics, err := Gather()
		if err != nil {
			t.Logf("failed to gather metrics: %v", err)
			return false
		}

		// Verify metric was collected
		found := false
		for _, mf := range metrics {
			if mf.GetName() == metricName {
				found = true
				// Verify it has the expected help text
				if mf.GetHelp() != helpText {
					t.Logf("expected help text %q, got %q", helpText, mf.GetHelp())
					return false
				}
				break
			}
		}

		if !found {
			t.Logf("metric %q not found in collected metrics", metricName)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 39 failed: %v", err)
	}
}

// TestProperty40_CustomMetricRegistration tests property 40: 自定义指标注册
// For any custom metric, it should be successfully registered.
// **Validates: Requirements 18.3**
func TestProperty40_CustomMetricRegistration(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(42)),
	}

	f := func(metricType int, metricName string, helpText string, labelNames []string) bool {
		// Reset global registry for test
		Registry = prometheus.NewRegistry()

		// Validate metric name
		if !isValidMetricName(metricName) {
			return true // Skip invalid names
		}

		// Validate label names
		for _, label := range labelNames {
			if !isValidLabelName(label) {
				return true // Skip invalid label names
			}
		}

		var customMetric *CustomMetric

		// Create different types of metrics based on metricType
		metricType = metricType % 4 // Ensure it's 0-3
		switch metricType {
		case 0: // Counter
			counterOpts := prometheus.CounterOpts{
				Name: metricName,
				Help: helpText,
			}
			if len(labelNames) > 0 {
				customMetric = NewCounterVec(counterOpts, labelNames)
			} else {
				customMetric = NewCounter(counterOpts)
			}

		case 1: // Gauge
			gaugeOpts := prometheus.GaugeOpts{
				Name: metricName,
				Help: helpText,
			}
			if len(labelNames) > 0 {
				customMetric = NewGaugeVec(gaugeOpts, labelNames)
			} else {
				customMetric = NewGauge(gaugeOpts)
			}

		case 2: // Histogram
			histogramOpts := prometheus.HistogramOpts{
				Name:    metricName,
				Help:    helpText,
				Buckets: []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
			}
			if len(labelNames) > 0 {
				customMetric = NewHistogramVec(histogramOpts, labelNames)
			} else {
				customMetric = NewHistogram(histogramOpts)
			}

		case 3: // Summary
			summaryOpts := prometheus.SummaryOpts{
				Name: metricName,
				Help: helpText,
				Objectives: map[float64]float64{
					0.5:  0.05,
					0.9:  0.01,
					0.99: 0.001,
				},
			}
			if len(labelNames) > 0 {
				customMetric = NewSummaryVec(summaryOpts, labelNames)
			} else {
				customMetric = NewSummary(summaryOpts)
			}
		}

		// Register the custom metric
		err := customMetric.Register()
		if err != nil {
			t.Logf("failed to register custom metric: %v", err)
			return false
		}

		// Verify the metric is registered
		metrics, err := Gather()
		if err != nil {
			t.Logf("failed to gather metrics: %v", err)
			return false
		}

		// Check if metric exists
		found := false
		for _, mf := range metrics {
			if mf.GetName() == metricName {
				found = true
				// Verify it has the expected help text
				if mf.GetHelp() != helpText {
					t.Logf("expected help text %q, got %q", helpText, mf.GetHelp())
					return false
				}
				break
			}
		}

		if !found {
			t.Logf("custom metric %q not found in registered metrics", metricName)
			return false
		}

		// Test unregistration
		unregistered := customMetric.Unregister()
		if !unregistered {
			t.Logf("failed to unregister custom metric")
			return false
		}

		// Verify metric is unregistered
		metrics, err = Gather()
		if err != nil {
			t.Logf("failed to gather metrics after unregister: %v", err)
			return false
		}

		// Check if metric no longer exists
		for _, mf := range metrics {
			if mf.GetName() == metricName {
				t.Logf("metric %q should have been unregistered but still exists", metricName)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 40 failed: %v", err)
	}
}

// TestMetricTypesSupported tests that all metric types are supported
// **Validates: Requirements 18.4**
func TestMetricTypesSupported(t *testing.T) {
	// Reset global registry for test
	Registry = prometheus.NewRegistry()

	// Test Counter
	counter := NewCounter(prometheus.CounterOpts{
		Name: "test_counter",
		Help: "Test counter",
	})
	assert.NoError(t, counter.Register())

	// Test Gauge
	gauge := NewGauge(prometheus.GaugeOpts{
		Name: "test_gauge",
		Help: "Test gauge",
	})
	assert.NoError(t, gauge.Register())

	// Test Histogram
	histogram := NewHistogram(prometheus.HistogramOpts{
		Name:    "test_histogram",
		Help:    "Test histogram",
		Buckets: []float64{0.1, 0.5, 1.0},
	})
	assert.NoError(t, histogram.Register())

	// Test Summary
	summary := NewSummary(prometheus.SummaryOpts{
		Name:       "test_summary",
		Help:       "Test summary",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01},
	})
	assert.NoError(t, summary.Register())

	// Verify all metrics are registered
	metrics, err := Gather()
	assert.NoError(t, err)

	expectedMetrics := []string{"test_counter", "test_gauge", "test_histogram", "test_summary"}
	metricNames := make(map[string]bool)
	for _, mf := range metrics {
		metricNames[mf.GetName()] = true
	}

	for _, expected := range expectedMetrics {
		assert.True(t, metricNames[expected], "expected metric %q not found", expected)
	}
}
