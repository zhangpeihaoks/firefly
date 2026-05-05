// Package health provides health check functionality for the Firefly framework.
// It supports liveness (/health) and readiness (/ready) checks with custom check functions.
package health

import (
	"context"
	"sync"
)

// Status represents the health status of a service.
type Status string

const (
	// StatusHealthy indicates the service is healthy.
	StatusHealthy Status = "healthy"
	// StatusUnhealthy indicates the service is unhealthy.
	StatusUnhealthy Status = "unhealthy"
	// StatusUnknown indicates the health status is unknown.
	StatusUnknown Status = "unknown"
)

// CheckFunc is a function that performs a health check.
// It returns nil if the check passes, or an error describing the failure.
type CheckFunc func(ctx context.Context) error

// Checker manages multiple health and readiness checks.
type Checker struct {
	mu              sync.RWMutex
	healthChecks    map[string]CheckFunc
	readinessChecks map[string]CheckFunc
}

// NewChecker creates a new Checker instance.
func NewChecker() *Checker {
	return &Checker{
		healthChecks:    make(map[string]CheckFunc),
		readinessChecks: make(map[string]CheckFunc),
	}
}

// AddCheck registers a named health check function.
// Health checks determine if the service is alive and functioning.
func (c *Checker) AddCheck(name string, check CheckFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.healthChecks[name] = check
}

// AddReadinessCheck registers a named readiness check function.
// Readiness checks determine if the service is ready to accept traffic.
func (c *Checker) AddReadinessCheck(name string, check CheckFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readinessChecks[name] = check
}

// Check runs all health checks concurrently and returns the overall status
// along with individual check results.
func (c *Checker) Check(ctx context.Context) (Status, map[string]error) {
	c.mu.RLock()
	checks := make(map[string]CheckFunc, len(c.healthChecks))
	for name, fn := range c.healthChecks {
		checks[name] = fn
	}
	c.mu.RUnlock()

	return runChecks(ctx, checks)
}

// CheckReadiness runs all readiness checks concurrently and returns the overall status
// along with individual check results.
func (c *Checker) CheckReadiness(ctx context.Context) (Status, map[string]error) {
	c.mu.RLock()
	checks := make(map[string]CheckFunc, len(c.readinessChecks))
	for name, fn := range c.readinessChecks {
		checks[name] = fn
	}
	c.mu.RUnlock()

	return runChecks(ctx, checks)
}

// IsHealthy returns true if all health checks pass.
func (c *Checker) IsHealthy(ctx context.Context) bool {
	status, _ := c.Check(ctx)
	return status == StatusHealthy
}

// IsReady returns true if all readiness checks pass.
func (c *Checker) IsReady(ctx context.Context) bool {
	status, _ := c.CheckReadiness(ctx)
	return status == StatusHealthy
}

// runChecks executes a set of check functions concurrently and returns the aggregated result.
func runChecks(ctx context.Context, checks map[string]CheckFunc) (Status, map[string]error) {
	if len(checks) == 0 {
		return StatusHealthy, make(map[string]error)
	}

	type result struct {
		name string
		err  error
	}

	results := make(chan result, len(checks))
	var wg sync.WaitGroup

	for name, fn := range checks {
		wg.Add(1)
		go func(n string, f CheckFunc) {
			defer wg.Done()
			err := f(ctx)
			results <- result{name: n, err: err}
		}(name, fn)
	}

	// Close results channel after all goroutines complete.
	go func() {
		wg.Wait()
		close(results)
	}()

	checkResults := make(map[string]error, len(checks))
	status := StatusHealthy

	for r := range results {
		checkResults[r.name] = r.err
		if r.err != nil {
			status = StatusUnhealthy
		}
	}

	return status, checkResults
}
