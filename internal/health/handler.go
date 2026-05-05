package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// healthResponse is the JSON response structure for health check endpoints.
type healthResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// HealthHandler returns a Gin handler for the /health liveness endpoint.
// It returns 200 if all health checks pass, or 503 if any check fails.
func HealthHandler(checker *Checker) gin.HandlerFunc {
	return func(c *gin.Context) {
		status, checks := checker.Check(c.Request.Context())

		resp := healthResponse{
			Status: string(status),
			Checks: formatChecks(checks),
		}

		code := http.StatusOK
		if status != StatusHealthy {
			code = http.StatusServiceUnavailable
		}

		c.JSON(code, resp)
	}
}

// ReadyHandler returns a Gin handler for the /ready readiness endpoint.
// It returns 200 if all readiness checks pass, or 503 if any check fails.
func ReadyHandler(checker *Checker) gin.HandlerFunc {
	return func(c *gin.Context) {
		status, checks := checker.CheckReadiness(c.Request.Context())

		resp := healthResponse{
			Status: string(status),
			Checks: formatChecks(checks),
		}

		code := http.StatusOK
		if status != StatusHealthy {
			code = http.StatusServiceUnavailable
		}

		c.JSON(code, resp)
	}
}

// formatChecks converts check results into a string map suitable for JSON output.
func formatChecks(checks map[string]error) map[string]string {
	result := make(map[string]string, len(checks))
	for name, err := range checks {
		if err != nil {
			result[name] = "error: " + err.Error()
		} else {
			result[name] = "ok"
		}
	}
	return result
}
