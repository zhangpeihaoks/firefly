package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/zhangpeihaoks/firefly/internal/health"
)

func init() { gin.SetMode(gin.TestMode) }

// setupRouter creates a test Gin router and mounts admin endpoints.
func setupRouter(opts ...Option) *gin.Engine {
	router := gin.New()
	Mount(router, opts...)
	return router
}

func TestHealthEndpoint_NoChecker(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest("GET", "/admin/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "no checks") {
		t.Errorf("expected 'no checks' message, got %s", w.Body.String())
	}
}

func TestHealthEndpoint_WithChecker(t *testing.T) {
	checker := health.NewChecker()
	checker.AddCheck("test-check", func(ctx context.Context) error { return nil })
	router := setupRouter(WithHealthChecker(checker))

	req := httptest.NewRequest("GET", "/admin/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestMetricsEndpoint(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest("GET", "/admin/metrics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestConfigEndpoint_NotProvided(t *testing.T) {
	router := setupRouter()

	req := httptest.NewRequest("GET", "/admin/config", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 when no config provider, got %d", w.Code)
	}
}

func TestConfigEndpoint_WithProvider(t *testing.T) {
	router := setupRouter(WithConfigProvider(func() any {
		return map[string]any{
			"name":     "test",
			"password": "secret123",
		}
	}))

	req := httptest.NewRequest("GET", "/admin/config", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "test") {
		t.Error("config should contain 'name'")
	}
	if strings.Contains(body, "secret123") {
		t.Error("password should be masked")
	}
	if !strings.Contains(body, "***") {
		t.Error("config should contain masked password")
	}
}

func TestPprofEndpoints(t *testing.T) {
	router := setupRouter()

	endpoints := []string{
		"/admin/debug/pprof/",
		"/admin/debug/pprof/heap",
		"/admin/debug/pprof/goroutine",
	}
	for _, ep := range endpoints {
		req := httptest.NewRequest("GET", ep, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("%s: expected 200, got %d", ep, w.Code)
		}
	}
}

func TestMaskSensitive(t *testing.T) {
	input := map[string]any{
		"name":     "myservice",
		"password": "secret",
		"database": map[string]any{
			"dsn":      "mysql://user:pass@host/db",
			"max_conn": 100,
		},
		"api_key": "key123",
	}

	result := maskSensitive(input).(map[string]any)

	if result["name"] != "myservice" {
		t.Errorf("name should not be masked, got %v", result["name"])
	}
	if result["password"] != "***" {
		t.Errorf("password should be masked, got %v", result["password"])
	}
	if result["api_key"] != "***" {
		t.Errorf("api_key should be masked, got %v", result["api_key"])
	}
	db := result["database"].(map[string]any)
	if db["dsn"] != "***" {
		t.Errorf("dsn should be masked, got %v", db["dsn"])
	}
	if db["max_conn"] != 100 {
		t.Errorf("max_conn should not be masked, got %v", db["max_conn"])
	}
}
