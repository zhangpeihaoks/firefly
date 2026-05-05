// Package metrics provides Prometheus metrics collection and exposure for the Firefly framework.
package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetrics(t *testing.T) {
	t.Run("handler returns metrics", func(t *testing.T) {
		// Create a test registry
		testRegistry := prometheus.NewRegistry()

		// Create a simple counter
		counter := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "test_counter",
			Help: "A test counter",
		})
		testRegistry.MustRegister(counter)
		counter.Inc()

		// Create handler with test registry
		handler := promhttp.HandlerFor(testRegistry, promhttp.HandlerOpts{
			Registry:          testRegistry,
			EnableOpenMetrics: true,
		})

		// Create test request
		req := httptest.NewRequest("GET", "/metrics", nil)
		rr := httptest.NewRecorder()

		// Serve the request
		handler.ServeHTTP(rr, req)

		// Check response
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "test_counter")
	})

	t.Run("global registry works", func(t *testing.T) {
		// Reset the global registry for this test
		Registry = prometheus.NewRegistry()

		// Create and register a counter
		counter := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "global_test_counter",
			Help: "A test counter in global registry",
		})
		MustRegister(counter)
		counter.Inc()

		// Get metrics from global registry
		metrics, err := Gather()
		require.NoError(t, err)
		assert.Greater(t, len(metrics), 0)
	})

	t.Run("custom metric registration", func(t *testing.T) {
		// Reset the global registry for this test
		Registry = prometheus.NewRegistry()

		// Create custom metric
		customMetric := NewCounter(prometheus.CounterOpts{
			Name: "custom_counter",
			Help: "A custom counter",
		})

		// Register it
		err := customMetric.Register()
		require.NoError(t, err)

		// Increment the counter
		if counter, ok := customMetric.collector.(prometheus.Counter); ok {
			counter.Inc()
		}

		// Verify it's registered
		metrics, err := Gather()
		require.NoError(t, err)
		assert.Greater(t, len(metrics), 0)
	})

	t.Run("must register panics on error", func(t *testing.T) {
		// Reset the global registry for this test
		Registry = prometheus.NewRegistry()

		// Create a metric
		counter := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "panic_test_counter",
			Help: "A test counter for panic test",
		})

		// This should not panic
		MustRegister(counter)

		// Try to register the same metric again - this should panic
		assert.Panics(t, func() {
			MustRegister(counter)
		})
	})

	t.Run("unregister works", func(t *testing.T) {
		// Reset the global registry for this test
		Registry = prometheus.NewRegistry()

		// Create and register a counter
		counter := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "unregister_test_counter",
			Help: "A test counter for unregister test",
		})
		MustRegister(counter)

		// Unregister it
		unregistered := Unregister(counter)
		assert.True(t, unregistered)

		// Try to unregister again - should return false
		unregistered = Unregister(counter)
		assert.False(t, unregistered)
	})

	t.Run("initialize registers default collectors", func(t *testing.T) {
		// Reset the global registry for this test
		Registry = prometheus.NewRegistry()

		// Initialize metrics
		Initialize()

		// Gather metrics
		metrics, err := Gather()
		require.NoError(t, err)

		// Should have some metrics from default collectors
		assert.Greater(t, len(metrics), 0)
	})
}
