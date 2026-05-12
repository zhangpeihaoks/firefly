// Package admin provides management endpoints for the Firefly framework.
// Mount it on an HTTP server to expose health, metrics, config, and pprof.
//
// Usage:
//
//	admin.Mount(srv.Router(), admin.WithHealthChecker(checker))
package admin

import (
	"encoding/json"
	"net/http"
	"net/http/pprof"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/zhangpeihaoks/firefly/internal/health"
	"github.com/zhangpeihaoks/firefly/internal/metrics"
)

// Mount registers admin routes on the given Gin router under /admin.
func Mount(router gin.IRouter, opts ...Option) {
	cfg := &adminConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	admin := router.Group("/admin")
	admin.GET("/health", cfg.healthHandler())
	admin.GET("/metrics", cfg.metricsHandler())

	if cfg.configProvider != nil {
		admin.GET("/config", cfg.configHandler())
	}

	pprofGroup := admin.Group("/debug/pprof")
	pprofGroup.GET("/", gin.WrapF(pprof.Index))
	pprofGroup.GET("/cmdline", gin.WrapF(pprof.Cmdline))
	pprofGroup.GET("/profile", gin.WrapF(pprof.Profile))
	pprofGroup.GET("/symbol", gin.WrapF(pprof.Symbol))
	pprofGroup.GET("/trace", gin.WrapF(pprof.Trace))
	pprofGroup.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
	pprofGroup.GET("/block", gin.WrapH(pprof.Handler("block")))
	pprofGroup.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
	pprofGroup.GET("/heap", gin.WrapH(pprof.Handler("heap")))
	pprofGroup.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
	pprofGroup.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
}

type adminConfig struct {
	healthChecker  *health.Checker
	configProvider func() any
}

type Option func(*adminConfig)

func WithHealthChecker(checker *health.Checker) Option {
	return func(c *adminConfig) { c.healthChecker = checker }
}

func WithConfigProvider(provider func() any) Option {
	return func(c *adminConfig) { c.configProvider = provider }
}

var sensitiveKeys = regexp.MustCompile(`(?i)(password|secret|token|key|credential|dsn)`)

func (c *adminConfig) healthHandler() gin.HandlerFunc {
	return func(gc *gin.Context) {
		if c.healthChecker == nil {
			gc.JSON(http.StatusOK, gin.H{"status": "no checks configured"})
			return
		}
		status, results := c.healthChecker.Check(gc.Request.Context())
		httpStatus := http.StatusOK
		if status != health.StatusHealthy {
			httpStatus = http.StatusServiceUnavailable
		}
		// Convert error map to string map for JSON
		output := make(map[string]string, len(results))
		for name, err := range results {
			if err != nil {
				output[name] = err.Error()
			} else {
				output[name] = "ok"
			}
		}
		gc.JSON(httpStatus, gin.H{
			"status":  string(status),
			"details": output,
		})
	}
}

func (c *adminConfig) metricsHandler() gin.HandlerFunc {
	return func(gc *gin.Context) {
		metrics.Handler().ServeHTTP(gc.Writer, gc.Request)
	}
}

func (c *adminConfig) configHandler() gin.HandlerFunc {
	return func(gc *gin.Context) {
		masked := maskSensitive(c.configProvider())
		gc.JSON(http.StatusOK, masked)
	}
}

func maskSensitive(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, vv := range val {
			if sensitiveKeys.MatchString(k) {
				result[k] = "***"
			} else {
				result[k] = maskSensitive(vv)
			}
		}
		return result
	case map[string]string:
		result := make(map[string]string, len(val))
		for k, vv := range val {
			if sensitiveKeys.MatchString(k) {
				result[k] = "***"
			} else {
				result[k] = vv
			}
		}
		return result
	default:
		return v
	}
}

var _ = json.Encoder{}
