package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------------------------------------------------------------------------
// Checker unit tests
// ---------------------------------------------------------------------------

func TestNewChecker(t *testing.T) {
	c := NewChecker()
	require.NotNil(t, c)
	assert.Empty(t, c.healthChecks)
	assert.Empty(t, c.readinessChecks)
}

func TestChecker_HealthyWhenAllChecksPass(t *testing.T) {
	c := NewChecker()
	c.AddCheck("db", func(ctx context.Context) error { return nil })
	c.AddCheck("redis", func(ctx context.Context) error { return nil })

	status, checks := c.Check(context.Background())

	assert.Equal(t, StatusHealthy, status)
	assert.Len(t, checks, 2)
	assert.NoError(t, checks["db"])
	assert.NoError(t, checks["redis"])
}

func TestChecker_UnhealthyWhenAnyCheckFails(t *testing.T) {
	c := NewChecker()
	c.AddCheck("db", func(ctx context.Context) error { return nil })
	c.AddCheck("redis", func(ctx context.Context) error {
		return errors.New("connection refused")
	})

	status, checks := c.Check(context.Background())

	assert.Equal(t, StatusUnhealthy, status)
	assert.NoError(t, checks["db"])
	assert.EqualError(t, checks["redis"], "connection refused")
}

func TestChecker_HealthyWithNoChecks(t *testing.T) {
	c := NewChecker()

	status, checks := c.Check(context.Background())

	assert.Equal(t, StatusHealthy, status)
	assert.Empty(t, checks)
}

func TestChecker_ReadinessChecks(t *testing.T) {
	c := NewChecker()
	c.AddReadinessCheck("cache-warm", func(ctx context.Context) error { return nil })
	c.AddReadinessCheck("migrations", func(ctx context.Context) error {
		return errors.New("pending migrations")
	})

	status, checks := c.CheckReadiness(context.Background())

	assert.Equal(t, StatusUnhealthy, status)
	assert.NoError(t, checks["cache-warm"])
	assert.EqualError(t, checks["migrations"], "pending migrations")
}

func TestChecker_ReadinessHealthyWhenAllPass(t *testing.T) {
	c := NewChecker()
	c.AddReadinessCheck("cache-warm", func(ctx context.Context) error { return nil })

	status, checks := c.CheckReadiness(context.Background())

	assert.Equal(t, StatusHealthy, status)
	assert.Len(t, checks, 1)
	assert.NoError(t, checks["cache-warm"])
}

func TestChecker_IsHealthy(t *testing.T) {
	c := NewChecker()
	c.AddCheck("ok", func(ctx context.Context) error { return nil })
	assert.True(t, c.IsHealthy(context.Background()))

	c.AddCheck("fail", func(ctx context.Context) error { return errors.New("down") })
	assert.False(t, c.IsHealthy(context.Background()))
}

func TestChecker_IsReady(t *testing.T) {
	c := NewChecker()
	c.AddReadinessCheck("ok", func(ctx context.Context) error { return nil })
	assert.True(t, c.IsReady(context.Background()))

	c.AddReadinessCheck("fail", func(ctx context.Context) error { return errors.New("not ready") })
	assert.False(t, c.IsReady(context.Background()))
}

func TestChecker_CustomCheckFunc(t *testing.T) {
	callCount := 0
	customCheck := func(ctx context.Context) error {
		callCount++
		return nil
	}

	c := NewChecker()
	c.AddCheck("custom", customCheck)

	_, _ = c.Check(context.Background())
	assert.Equal(t, 1, callCount)

	_, _ = c.Check(context.Background())
	assert.Equal(t, 2, callCount)
}

func TestChecker_HealthAndReadinessIndependent(t *testing.T) {
	c := NewChecker()
	c.AddCheck("health-ok", func(ctx context.Context) error { return nil })
	c.AddReadinessCheck("ready-fail", func(ctx context.Context) error {
		return errors.New("not ready")
	})

	assert.True(t, c.IsHealthy(context.Background()))
	assert.False(t, c.IsReady(context.Background()))
}

func TestChecker_ConcurrentChecks(t *testing.T) {
	var counter atomic.Int32

	c := NewChecker()
	for i := 0; i < 10; i++ {
		c.AddCheck("check-"+string(rune('a'+i)), func(ctx context.Context) error {
			counter.Add(1)
			return nil
		})
	}

	status, checks := c.Check(context.Background())

	assert.Equal(t, StatusHealthy, status)
	assert.Len(t, checks, 10)
	assert.Equal(t, int32(10), counter.Load())
}

// ---------------------------------------------------------------------------
// HTTP handler tests
// ---------------------------------------------------------------------------

func setupRouter(checker *Checker) *gin.Engine {
	r := gin.New()
	r.GET("/health", HealthHandler(checker))
	r.GET("/ready", ReadyHandler(checker))
	return r
}

func TestHealthHandler_Returns200WhenHealthy(t *testing.T) {
	c := NewChecker()
	c.AddCheck("db", func(ctx context.Context) error { return nil })

	router := setupRouter(c)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp healthResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "healthy", resp.Status)
	assert.Equal(t, "ok", resp.Checks["db"])
}

func TestHealthHandler_Returns503WhenUnhealthy(t *testing.T) {
	c := NewChecker()
	c.AddCheck("db", func(ctx context.Context) error { return nil })
	c.AddCheck("redis", func(ctx context.Context) error {
		return errors.New("connection refused")
	})

	router := setupRouter(c)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var resp healthResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "unhealthy", resp.Status)
	assert.Equal(t, "ok", resp.Checks["db"])
	assert.Equal(t, "error: connection refused", resp.Checks["redis"])
}

func TestHealthHandler_Returns200WhenNoChecks(t *testing.T) {
	c := NewChecker()

	router := setupRouter(c)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp healthResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "healthy", resp.Status)
}

func TestReadyHandler_Returns200WhenReady(t *testing.T) {
	c := NewChecker()
	c.AddReadinessCheck("cache", func(ctx context.Context) error { return nil })

	router := setupRouter(c)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ready", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp healthResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "healthy", resp.Status)
	assert.Equal(t, "ok", resp.Checks["cache"])
}

func TestReadyHandler_Returns503WhenNotReady(t *testing.T) {
	c := NewChecker()
	c.AddReadinessCheck("migrations", func(ctx context.Context) error {
		return errors.New("pending migrations")
	})

	router := setupRouter(c)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/ready", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var resp healthResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "unhealthy", resp.Status)
	assert.Equal(t, "error: pending migrations", resp.Checks["migrations"])
}

func TestHealthHandler_ResponseFormat(t *testing.T) {
	c := NewChecker()
	c.AddCheck("service-a", func(ctx context.Context) error { return nil })
	c.AddCheck("service-b", func(ctx context.Context) error {
		return errors.New("timeout")
	})

	router := setupRouter(c)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(w, req)

	// Verify JSON structure has expected fields.
	var raw map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &raw)
	require.NoError(t, err)

	_, hasStatus := raw["status"]
	_, hasChecks := raw["checks"]
	assert.True(t, hasStatus, "response should have 'status' field")
	assert.True(t, hasChecks, "response should have 'checks' field")
}
