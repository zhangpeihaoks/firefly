// Package database provides database health check integration for the Firefly framework.
// This file implements health check functions that can be registered with the health module.
package database

import (
	"context"
	"fmt"
)

// HealthCheckNames provides standard names for database health checks.
const (
	// HealthCheckPrefix is the prefix for all database health check names.
	HealthCheckPrefix = "database"
)

// HealthCheckFunc returns a health check function for a specific database connection.
// The returned function can be registered with the health.Checker.
func HealthCheckFunc(name string, connector Connector) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if !connector.IsConnected() {
			return fmt.Errorf("database %s is not connected", name)
		}

		if err := connector.Ping(ctx); err != nil {
			return fmt.Errorf("database %s ping failed: %w", name, err)
		}

		return nil
	}
}

// DetailedHealthCheckFunc returns a health check function that provides detailed status.
// The returned function uses the HealthChecker interface if available.
func DetailedHealthCheckFunc(name string, connector Connector) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		checker, ok := connector.(HealthChecker)
		if !ok {
			// Fall back to basic health check
			return HealthCheckFunc(name, connector)(ctx)
		}

		status := checker.CheckHealth(ctx)
		if status.Status != "healthy" {
			if status.Message != "" {
				return fmt.Errorf("database %s: %s", name, status.Message)
			}
			return fmt.Errorf("database %s is %s", name, status.Status)
		}

		return nil
	}
}

// RegisterHealthChecks registers health checks for all database connections in the manager.
// This function registers both liveness and readiness checks for each connection.
func RegisterHealthChecks(checker HealthCheckRegistrar, manager *Manager) error {
	connections := manager.GetAll()
	for name, connector := range connections {
		checkName := fmt.Sprintf("%s.%s", HealthCheckPrefix, name)

		// Register liveness check (basic connectivity)
		checker.AddCheck(checkName, HealthCheckFunc(name, connector))

		// Register readiness check (detailed health with ping)
		checker.AddReadinessCheck(checkName, DetailedHealthCheckFunc(name, connector))
	}
	return nil
}

// HealthCheckRegistrar is the interface for registering health checks.
// This matches the health.Checker interface to avoid import cycles.
type HealthCheckRegistrar interface {
	// AddCheck registers a named health check function.
	AddCheck(name string, check func(ctx context.Context) error)
	// AddReadinessCheck registers a named readiness check function.
	AddReadinessCheck(name string, check func(ctx context.Context) error)
}

// ManagerHealthCheck returns a health check function for the entire database manager.
// This checks all connections and returns an error if any connection is unhealthy.
func ManagerHealthCheck(manager *Manager) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		results := manager.CheckHealth(ctx)

		var errs []error
		for name, status := range results {
			if status.Status != "healthy" {
				if status.Message != "" {
					errs = append(errs, fmt.Errorf("database %s: %s", name, status.Message))
				} else {
					errs = append(errs, fmt.Errorf("database %s is %s", name, status.Status))
				}
			}
		}

		if len(errs) > 0 {
			return fmt.Errorf("database health check failed: %v", errs)
		}

		return nil
	}
}

// RegisterManagerHealthCheck registers a single health check for the database manager.
// This is an alternative to RegisterHealthChecks when you want a single aggregated check.
func RegisterManagerHealthCheck(checker HealthCheckRegistrar, manager *Manager) {
	checkName := fmt.Sprintf("%s.manager", HealthCheckPrefix)

	// Register liveness check
	checker.AddCheck(checkName, ManagerHealthCheck(manager))

	// Register readiness check
	checker.AddReadinessCheck(checkName, ManagerHealthCheck(manager))
}
