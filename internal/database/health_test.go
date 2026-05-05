// Package database provides database health check integration tests for the Firefly framework.
package database

import (
	"context"
	"errors"
	"testing"
	"time"
)

// healthTestConnector implements Connector interface for health testing
type healthTestConnector struct {
	connected bool
	pingErr   error
	poolStats *PoolStats
}

func (m *healthTestConnector) Connect(ctx context.Context) error {
	m.connected = true
	return nil
}

func (m *healthTestConnector) Disconnect(ctx context.Context) error {
	m.connected = false
	return nil
}

func (m *healthTestConnector) IsConnected() bool {
	return m.connected
}

func (m *healthTestConnector) Ping(ctx context.Context) error {
	return m.pingErr
}

func (m *healthTestConnector) Stats() *PoolStats {
	return m.poolStats
}

// healthTestChecker implements HealthChecker interface for testing
type healthTestChecker struct {
	healthTestConnector
	status  string
	message string
	latency time.Duration
}

func (m *healthTestChecker) CheckHealth(ctx context.Context) *HealthStatus {
	return &HealthStatus{
		Status:  m.status,
		Message: m.message,
		Latency: m.latency,
		Stats:   m.poolStats,
	}
}

// mockHealthCheckRegistrar implements HealthCheckRegistrar for testing
type mockHealthCheckRegistrar struct {
	checks          map[string]func(ctx context.Context) error
	readinessChecks map[string]func(ctx context.Context) error
}

func newMockHealthCheckRegistrar() *mockHealthCheckRegistrar {
	return &mockHealthCheckRegistrar{
		checks:          make(map[string]func(ctx context.Context) error),
		readinessChecks: make(map[string]func(ctx context.Context) error),
	}
}

func (m *mockHealthCheckRegistrar) AddCheck(name string, check func(ctx context.Context) error) {
	m.checks[name] = check
}

func (m *mockHealthCheckRegistrar) AddReadinessCheck(name string, check func(ctx context.Context) error) {
	m.readinessChecks[name] = check
}

func TestHealthCheckFunc(t *testing.T) {
	tests := []struct {
		name       string
		dbName     string
		connector  Connector
		wantErr    bool
		errContain string
	}{
		{
			name:      "healthy connection",
			dbName:    "primary",
			connector: &healthTestConnector{connected: true, pingErr: nil},
			wantErr:   false,
		},
		{
			name:       "not connected",
			dbName:     "primary",
			connector:  &healthTestConnector{connected: false, pingErr: nil},
			wantErr:    true,
			errContain: "not connected",
		},
		{
			name:       "ping failed",
			dbName:     "primary",
			connector:  &healthTestConnector{connected: true, pingErr: errors.New("connection refused")},
			wantErr:    true,
			errContain: "ping failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			check := HealthCheckFunc(tt.dbName, tt.connector)
			err := check(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContain != "" && !contains(err.Error(), tt.errContain) {
					t.Errorf("error should contain %q, got %q", tt.errContain, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestDetailedHealthCheckFunc(t *testing.T) {
	tests := []struct {
		name       string
		dbName     string
		connector  Connector
		wantErr    bool
		errContain string
	}{
		{
			name:      "healthy connection with HealthChecker",
			dbName:    "primary",
			connector: &healthTestChecker{healthTestConnector: healthTestConnector{connected: true}, status: "healthy", message: "OK"},
			wantErr:   false,
		},
		{
			name:       "unhealthy connection with HealthChecker",
			dbName:     "primary",
			connector:  &healthTestChecker{healthTestConnector: healthTestConnector{connected: true}, status: "unhealthy", message: "connection lost"},
			wantErr:    true,
			errContain: "connection lost",
		},
		{
			name:      "healthy connection without HealthChecker",
			dbName:    "primary",
			connector: &healthTestConnector{connected: true, pingErr: nil},
			wantErr:   false,
		},
		{
			name:       "unhealthy connection without HealthChecker",
			dbName:     "primary",
			connector:  &healthTestConnector{connected: true, pingErr: errors.New("timeout")},
			wantErr:    true,
			errContain: "ping failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			check := DetailedHealthCheckFunc(tt.dbName, tt.connector)
			err := check(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errContain != "" && !contains(err.Error(), tt.errContain) {
					t.Errorf("error should contain %q, got %q", tt.errContain, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestRegisterHealthChecks(t *testing.T) {
	// Create a manager with mock connections
	manager := NewManager()

	// Create mock connectors
	conn1 := &healthTestChecker{
		healthTestConnector: healthTestConnector{
			connected: true,
			poolStats: &PoolStats{MaxOpenConnections: 100, OpenConnections: 10},
		},
		status:  "healthy",
		message: "OK",
	}
	conn2 := &healthTestChecker{
		healthTestConnector: healthTestConnector{connected: true},
		status:              "healthy",
		message:             "OK",
	}

	// Add connections to manager
	manager.connections["primary"] = conn1
	manager.connections["cache"] = conn2

	// Create mock registrar
	registrar := newMockHealthCheckRegistrar()

	// Register health checks
	err := RegisterHealthChecks(registrar, manager)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify checks were registered
	if len(registrar.checks) != 2 {
		t.Errorf("expected 2 health checks, got %d", len(registrar.checks))
	}
	if len(registrar.readinessChecks) != 2 {
		t.Errorf("expected 2 readiness checks, got %d", len(registrar.readinessChecks))
	}

	// Verify check names
	expectedNames := []string{"database.primary", "database.cache"}
	for _, name := range expectedNames {
		if _, ok := registrar.checks[name]; !ok {
			t.Errorf("expected health check %q to be registered", name)
		}
		if _, ok := registrar.readinessChecks[name]; !ok {
			t.Errorf("expected readiness check %q to be registered", name)
		}
	}

	// Execute a health check
	ctx := context.Background()
	for name, check := range registrar.checks {
		if err := check(ctx); err != nil {
			t.Errorf("health check %q failed: %v", name, err)
		}
	}
}

func TestManagerHealthCheck(t *testing.T) {
	tests := []struct {
		name        string
		connections map[string]Connector
		wantErr     bool
	}{
		{
			name: "all healthy",
			connections: map[string]Connector{
				"primary": &healthTestChecker{
					healthTestConnector: healthTestConnector{connected: true},
					status:              "healthy",
				},
				"cache": &healthTestChecker{
					healthTestConnector: healthTestConnector{connected: true},
					status:              "healthy",
				},
			},
			wantErr: false,
		},
		{
			name: "one unhealthy",
			connections: map[string]Connector{
				"primary": &healthTestChecker{
					healthTestConnector: healthTestConnector{connected: true},
					status:              "healthy",
				},
				"cache": &healthTestChecker{
					healthTestConnector: healthTestConnector{connected: true},
					status:              "unhealthy",
					message:             "connection lost",
				},
			},
			wantErr: true,
		},
		{
			name: "all unhealthy",
			connections: map[string]Connector{
				"primary": &healthTestChecker{
					healthTestConnector: healthTestConnector{connected: true},
					status:              "unhealthy",
					message:             "timeout",
				},
				"cache": &healthTestChecker{
					healthTestConnector: healthTestConnector{connected: true},
					status:              "unhealthy",
					message:             "refused",
				},
			},
			wantErr: true,
		},
		{
			name:        "empty manager",
			connections: map[string]Connector{},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager()
			for name, conn := range tt.connections {
				manager.connections[name] = conn
			}

			ctx := context.Background()
			check := ManagerHealthCheck(manager)
			err := check(ctx)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestRegisterManagerHealthCheck(t *testing.T) {
	manager := NewManager()
	manager.connections["primary"] = &healthTestChecker{
		healthTestConnector: healthTestConnector{connected: true},
		status:              "healthy",
	}

	registrar := newMockHealthCheckRegistrar()

	RegisterManagerHealthCheck(registrar, manager)

	// Verify single check was registered
	if len(registrar.checks) != 1 {
		t.Errorf("expected 1 health check, got %d", len(registrar.checks))
	}
	if len(registrar.readinessChecks) != 1 {
		t.Errorf("expected 1 readiness check, got %d", len(registrar.readinessChecks))
	}

	// Verify check name
	expectedName := "database.manager"
	if _, ok := registrar.checks[expectedName]; !ok {
		t.Errorf("expected health check %q to be registered", expectedName)
	}
	if _, ok := registrar.readinessChecks[expectedName]; !ok {
		t.Errorf("expected readiness check %q to be registered", expectedName)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
