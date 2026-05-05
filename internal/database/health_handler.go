// Package database provides database health check HTTP handlers for the Firefly framework.
package database

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthHandler returns a Gin handler for database health status endpoint.
// This provides a detailed view of all database connections and their health.
func HealthHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		results := manager.CheckHealth(c.Request.Context())

		// Determine overall status
		overallStatus := "healthy"
		response := make(map[string]interface{})
		checks := make(map[string]interface{})

		for name, status := range results {
			checks[name] = map[string]interface{}{
				"status":  status.Status,
				"message": status.Message,
				"latency": status.Latency.String(),
			}

			if status.Status != "healthy" {
				overallStatus = "unhealthy"
			}

			// Include stats if available
			if status.Stats != nil {
				checks[name].(map[string]interface{})["stats"] = status.Stats
			}
		}

		response["status"] = overallStatus
		response["checks"] = checks
		response["timestamp"] = time.Now().UTC().Format(time.RFC3339)

		code := http.StatusOK
		if overallStatus != "healthy" {
			code = http.StatusServiceUnavailable
		}

		c.JSON(code, response)
	}
}

// DetailedHealthHandler returns a Gin handler for detailed database health status.
// This includes connection pool statistics and latency metrics.
func DetailedHealthHandler(manager *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		results := manager.CheckHealth(c.Request.Context())

		// Determine overall status
		overallStatus := "healthy"
		response := make(map[string]interface{})
		databases := make(map[string]interface{})

		for name, status := range results {
			dbInfo := map[string]interface{}{
				"status":  status.Status,
				"message": status.Message,
				"latency": status.Latency.Milliseconds(),
			}

			// Include detailed stats
			if status.Stats != nil {
				dbInfo["pool"] = map[string]interface{}{
					"max_open_connections": status.Stats.MaxOpenConnections,
					"open_connections":     status.Stats.OpenConnections,
					"in_use":               status.Stats.InUse,
					"idle":                 status.Stats.Idle,
					"wait_count":           status.Stats.WaitCount,
					"wait_duration_ms":     status.Stats.WaitDuration.Milliseconds(),
				}

				// Include Redis-specific stats if available
				if status.Stats.Hits > 0 || status.Stats.Misses > 0 {
					dbInfo["pool"].(map[string]interface{})["hits"] = status.Stats.Hits
					dbInfo["pool"].(map[string]interface{})["misses"] = status.Stats.Misses
					dbInfo["pool"].(map[string]interface{})["timeouts"] = status.Stats.Timeouts
					dbInfo["pool"].(map[string]interface{})["total_conns"] = status.Stats.TotalConns
					dbInfo["pool"].(map[string]interface{})["idle_conns"] = status.Stats.IdleConns
					dbInfo["pool"].(map[string]interface{})["stale_conns"] = status.Stats.StaleConns
				}
			}

			databases[name] = dbInfo

			if status.Status != "healthy" {
				overallStatus = "unhealthy"
			}
		}

		response["status"] = overallStatus
		response["databases"] = databases
		response["timestamp"] = time.Now().UTC().Format(time.RFC3339)

		// Get connection list
		response["connections"] = manager.List()

		code := http.StatusOK
		if overallStatus != "healthy" {
			code = http.StatusServiceUnavailable
		}

		c.JSON(code, response)
	}
}
