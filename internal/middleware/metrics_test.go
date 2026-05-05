// Package middleware provides middleware abstractions for the Firefly framework.
package middleware

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zhangpeihaoks/firefly/internal/errors"
	"github.com/zhangpeihaoks/firefly/internal/transport"
)

func TestMetrics(t *testing.T) {
	t.Run("records successful request metrics", func(t *testing.T) {
		// Create a new registry for this test
		registry := prometheus.NewRegistry()

		// Create metrics middleware with custom registry
		m := Metrics(WithRegistry(registry))

		// Create a handler that returns success
		handler := func(ctx context.Context, req any) (any, error) {
			return "success", nil
		}

		// Create context with transport info
		tr := &mockTransporter{
			kind:      transport.KindHTTP,
			endpoint:  "localhost:8080",
			operation: "/api/test",
		}
		ctx := transport.NewContext(context.Background(), tr)

		// Execute middleware
		wrapped := m(handler)
		resp, err := wrapped(ctx, nil)

		// Verify response
		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
	})

	t.Run("records error request metrics", func(t *testing.T) {
		// Create a new registry for this test
		registry := prometheus.NewRegistry()

		// Create metrics middleware with custom registry
		m := Metrics(WithRegistry(registry))

		// Create a handler that returns an error
		handler := func(ctx context.Context, req any) (any, error) {
			return nil, errors.New(errors.CodeNotFound, "NOT_FOUND", "resource not found")
		}

		// Create context with transport info
		tr := &mockTransporter{
			kind:      transport.KindHTTP,
			endpoint:  "localhost:8080",
			operation: "/api/users/:id",
		}
		ctx := transport.NewContext(context.Background(), tr)

		// Execute middleware
		wrapped := m(handler)
		resp, err := wrapped(ctx, nil)

		// Verify error response
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("works without transport context", func(t *testing.T) {
		// Create a new registry for this test
		registry := prometheus.NewRegistry()

		// Create metrics middleware with custom registry
		m := Metrics(WithRegistry(registry))

		// Create a handler that returns success
		handler := func(ctx context.Context, req any) (any, error) {
			return "success", nil
		}

		// Execute middleware without transport context
		wrapped := m(handler)
		resp, err := wrapped(context.Background(), nil)

		// Verify response
		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
	})

	t.Run("uses custom buckets", func(t *testing.T) {
		// Create a new registry for this test
		registry := prometheus.NewRegistry()

		// Custom buckets
		customBuckets := []float64{0.1, 0.5, 1.0}

		// Create metrics middleware with custom buckets
		m := Metrics(
			WithRegistry(registry),
			WithBuckets(customBuckets),
		)

		// Create a handler
		handler := func(ctx context.Context, req any) (any, error) {
			return "success", nil
		}

		// Create context with transport info
		tr := &mockTransporter{
			kind:      transport.KindHTTP,
			endpoint:  "localhost:8080",
			operation: "/api/test",
		}
		ctx := transport.NewContext(context.Background(), tr)

		// Execute middleware
		wrapped := m(handler)
		_, err := wrapped(ctx, nil)

		assert.NoError(t, err)
	})

	t.Run("records multiple requests", func(t *testing.T) {
		// Create a new registry for this test
		registry := prometheus.NewRegistry()

		// Create metrics middleware with custom registry
		m := Metrics(WithRegistry(registry))

		// Create a handler that returns success
		handler := func(ctx context.Context, req any) (any, error) {
			return "success", nil
		}

		// Create context with transport info
		tr := &mockTransporter{
			kind:      transport.KindHTTP,
			endpoint:  "localhost:8080",
			operation: "/api/test",
		}
		ctx := transport.NewContext(context.Background(), tr)

		// Execute middleware multiple times
		wrapped := m(handler)
		for i := 0; i < 5; i++ {
			_, err := wrapped(ctx, nil)
			require.NoError(t, err)
		}
	})

	t.Run("records gRPC transport metrics", func(t *testing.T) {
		// Create a new registry for this test
		registry := prometheus.NewRegistry()

		// Create metrics middleware with custom registry
		m := Metrics(WithRegistry(registry))

		// Create a handler that returns success
		handler := func(ctx context.Context, req any) (any, error) {
			return "success", nil
		}

		// Create context with gRPC transport info
		tr := &mockTransporter{
			kind:      transport.KindGRPC,
			endpoint:  "localhost:9090",
			operation: "/api.UserService/GetUser",
		}
		ctx := transport.NewContext(context.Background(), tr)

		// Execute middleware
		wrapped := m(handler)
		resp, err := wrapped(ctx, nil)

		// Verify response
		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
	})

	t.Run("records generic error", func(t *testing.T) {
		// Create a new registry for this test
		registry := prometheus.NewRegistry()

		// Create metrics middleware with custom registry
		m := Metrics(WithRegistry(registry))

		// Create a handler that returns a generic error
		handler := func(ctx context.Context, req any) (any, error) {
			return nil, errors.New(errors.CodeInternal, "INTERNAL_ERROR", "generic error")
		}

		// Create context with transport info
		tr := &mockTransporter{
			kind:      transport.KindHTTP,
			endpoint:  "localhost:8080",
			operation: "/api/test",
		}
		ctx := transport.NewContext(context.Background(), tr)

		// Execute middleware
		wrapped := m(handler)
		resp, err := wrapped(ctx, nil)

		// Verify error response
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestMetricsWithCustomMetrics(t *testing.T) {
	t.Run("uses custom counter", func(t *testing.T) {
		// Create a new registry for this test
		registry := prometheus.NewRegistry()

		// Create custom counter
		customCounter := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "custom",
				Name:      "requests_total",
				Help:      "Custom request counter",
			},
			[]string{"kind", "method", "path", "status"},
		)
		registry.MustRegister(customCounter)

		// Create metrics middleware with custom counter
		m := Metrics(
			WithRegistry(registry),
			WithCounter(customCounter),
		)

		// Create a handler
		handler := func(ctx context.Context, req any) (any, error) {
			return "success", nil
		}

		// Create context with transport info
		tr := &mockTransporter{
			kind:      transport.KindHTTP,
			endpoint:  "localhost:8080",
			operation: "/api/test",
		}
		ctx := transport.NewContext(context.Background(), tr)

		// Execute middleware
		wrapped := m(handler)
		_, err := wrapped(ctx, nil)

		assert.NoError(t, err)

		// Verify custom counter was used
		count := testutil.ToFloat64(customCounter)
		assert.Equal(t, float64(1), count)
	})

	t.Run("uses custom latency histogram", func(t *testing.T) {
		// Create a new registry for this test
		registry := prometheus.NewRegistry()

		// Create custom latency histogram
		customLatency := prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "custom",
				Name:      "request_duration_seconds",
				Help:      "Custom request latency",
				Buckets:   []float64{0.1, 0.5, 1.0},
			},
			[]string{"kind", "method", "path"},
		)
		registry.MustRegister(customLatency)

		// Create metrics middleware with custom latency
		m := Metrics(
			WithRegistry(registry),
			WithLatency(customLatency),
		)

		// Create a handler
		handler := func(ctx context.Context, req any) (any, error) {
			return "success", nil
		}

		// Create context with transport info
		tr := &mockTransporter{
			kind:      transport.KindHTTP,
			endpoint:  "localhost:8080",
			operation: "/api/test",
		}
		ctx := transport.NewContext(context.Background(), tr)

		// Execute middleware
		wrapped := m(handler)
		_, err := wrapped(ctx, nil)

		assert.NoError(t, err)
	})

	t.Run("uses custom error counter", func(t *testing.T) {
		// Create a new registry for this test
		registry := prometheus.NewRegistry()

		// Create custom error counter
		customErrorCount := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "custom",
				Name:      "errors_total",
				Help:      "Custom error counter",
			},
			[]string{"kind", "method", "path", "code"},
		)
		registry.MustRegister(customErrorCount)

		// Create metrics middleware with custom error counter
		m := Metrics(
			WithRegistry(registry),
			WithErrorCount(customErrorCount),
		)

		// Create a handler that returns an error
		handler := func(ctx context.Context, req any) (any, error) {
			return nil, errors.New(errors.CodeInternal, "INTERNAL_ERROR", "internal error")
		}

		// Create context with transport info
		tr := &mockTransporter{
			kind:      transport.KindHTTP,
			endpoint:  "localhost:8080",
			operation: "/api/test",
		}
		ctx := transport.NewContext(context.Background(), tr)

		// Execute middleware
		wrapped := m(handler)
		_, err := wrapped(ctx, nil)

		assert.Error(t, err)

		// Verify custom error counter was used
		count := testutil.ToFloat64(customErrorCount)
		assert.Equal(t, float64(1), count)
	})
}

func TestDefaultBuckets(t *testing.T) {
	// Verify default buckets are reasonable
	assert.Contains(t, DefaultBuckets, 0.001) // 1ms
	assert.Contains(t, DefaultBuckets, 0.01)  // 10ms
	assert.Contains(t, DefaultBuckets, 0.1)   // 100ms
	assert.Contains(t, DefaultBuckets, 1.0)   // 1s
	assert.Contains(t, DefaultBuckets, 10.0)  // 10s
	assert.Greater(t, len(DefaultBuckets), 0)
}
