// Package integration provides integration tests for database connections.
package integration

import (
	"testing"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/database"
)

// TestDatabaseConfigValidation tests the database configuration validation.
func TestDatabaseConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *database.Config
		wantErr bool
	}{
		{
			name: "valid MySQL config",
			cfg: &database.Config{
				Driver: "mysql",
				DSN:    "root:password@tcp(localhost:3306)/testdb",
			},
			wantErr: false,
		},
		{
			name: "valid PostgreSQL config",
			cfg: &database.Config{
				Driver: "postgres",
				DSN:    "host=localhost user=postgres password=password dbname=testdb port=5432",
			},
			wantErr: false,
		},
		{
			name: "valid Redis config",
			cfg: &database.Config{
				Driver: "redis",
				DSN:    "redis://localhost:6379",
			},
			wantErr: false,
		},
		{
			name: "valid MongoDB config",
			cfg: &database.Config{
				Driver: "mongodb",
				DSN:    "mongodb://localhost:27017/testdb",
			},
			wantErr: false,
		},
		{
			name: "missing driver",
			cfg: &database.Config{
				DSN: "some-dsn",
			},
			wantErr: true,
		},
		{
			name: "invalid driver",
			cfg: &database.Config{
				Driver: "invalid",
				DSN:    "some-dsn",
			},
			wantErr: true,
		},
		{
			name: "missing DSN",
			cfg: &database.Config{
				Driver: "mysql",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPoolConfigDefaults tests the default pool configuration.
func TestPoolConfigDefaults(t *testing.T) {
	t.Run("DefaultPoolConfig", func(t *testing.T) {
		cfg := database.DefaultPoolConfig()

		if cfg.MaxOpenConns != 100 {
			t.Errorf("Expected MaxOpenConns=100, got %d", cfg.MaxOpenConns)
		}
		if cfg.MaxIdleConns != 10 {
			t.Errorf("Expected MaxIdleConns=10, got %d", cfg.MaxIdleConns)
		}
		if cfg.ConnMaxLifetime != 30*time.Minute {
			t.Errorf("Expected ConnMaxLifetime=30m, got %v", cfg.ConnMaxLifetime)
		}
		if cfg.ConnMaxIdleTime != 10*time.Minute {
			t.Errorf("Expected ConnMaxIdleTime=10m, got %v", cfg.ConnMaxIdleTime)
		}
	})

	t.Run("GetPoolConfig with nil", func(t *testing.T) {
		cfg := &database.Config{}
		pool := cfg.GetPoolConfig()

		if pool.MaxOpenConns != 100 {
			t.Errorf("Expected default MaxOpenConns=100, got %d", pool.MaxOpenConns)
		}
	})

	t.Run("GetPoolConfig with partial config", func(t *testing.T) {
		cfg := &database.Config{
			Pool: &database.PoolConfig{
				MaxOpenConns: 50,
			},
		}
		pool := cfg.GetPoolConfig()

		// Custom value should be preserved
		if pool.MaxOpenConns != 50 {
			t.Errorf("Expected MaxOpenConns=50, got %d", pool.MaxOpenConns)
		}
		// Default should be applied
		if pool.MaxIdleConns != 10 {
			t.Errorf("Expected default MaxIdleConns=10, got %d", pool.MaxIdleConns)
		}
	})
}

// TestDatabasePoolStats tests the PoolStats structure.
func TestDatabasePoolStats(t *testing.T) {
	stats := &database.PoolStats{
		MaxOpenConnections: 100,
		OpenConnections:    10,
		InUse:              5,
		Idle:               5,
		WaitCount:          100,
		WaitDuration:       50 * time.Millisecond,
	}

	if stats.MaxOpenConnections != 100 {
		t.Errorf("Expected 100, got %d", stats.MaxOpenConnections)
	}
	if stats.OpenConnections != 10 {
		t.Errorf("Expected 10, got %d", stats.OpenConnections)
	}
	if stats.InUse != 5 {
		t.Errorf("Expected 5, got %d", stats.InUse)
	}
	if stats.Idle != 5 {
		t.Errorf("Expected 5, got %d", stats.Idle)
	}
}

// TestRedisPoolStats tests Redis-specific pool stats.
func TestRedisPoolStats(t *testing.T) {
	stats := &database.PoolStats{
		Hits:       1000,
		Misses:     50,
		Timeouts:   5,
		TotalConns: 10,
		IdleConns:  8,
		StaleConns: 2,
	}

	if stats.Hits != 1000 {
		t.Errorf("Expected 1000, got %d", stats.Hits)
	}
	if stats.Misses != 50 {
		t.Errorf("Expected 50, got %d", stats.Misses)
	}
	if stats.TotalConns != 10 {
		t.Errorf("Expected 10, got %d", stats.TotalConns)
	}
	if stats.IdleConns != 8 {
		t.Errorf("Expected 8, got %d", stats.IdleConns)
	}
}

// TestDatabaseConfigPoolFields tests pool configuration fields.
func TestDatabaseConfigPoolFields(t *testing.T) {
	t.Run("PoolConfig fields", func(t *testing.T) {
		pool := &database.PoolConfig{
			MaxOpenConns:    50,
			MaxIdleConns:    5,
			ConnMaxLifetime: 15 * time.Minute,
			ConnMaxIdleTime: 5 * time.Minute,
			ConnectTimeout:  10 * time.Second,
		}

		if pool.MaxOpenConns != 50 {
			t.Errorf("Expected MaxOpenConns=50, got %d", pool.MaxOpenConns)
		}
		if pool.MaxIdleConns != 5 {
			t.Errorf("Expected MaxIdleConns=5, got %d", pool.MaxIdleConns)
		}
		if pool.ConnMaxLifetime != 15*time.Minute {
			t.Errorf("Expected ConnMaxLifetime=15m, got %v", pool.ConnMaxLifetime)
		}
		if pool.ConnMaxIdleTime != 5*time.Minute {
			t.Errorf("Expected ConnMaxIdleTime=5m, got %v", pool.ConnMaxIdleTime)
		}
		if pool.ConnectTimeout != 10*time.Second {
			t.Errorf("Expected ConnectTimeout=10s, got %v", pool.ConnectTimeout)
		}
	})
}

// TestDatabaseConfigDriverTypes tests different database driver types.
func TestDatabaseConfigDriverTypes(t *testing.T) {
	t.Run("Driver types", func(t *testing.T) {
		tests := []struct {
			driver    string
			wantValid bool
		}{
			{"mysql", true},
			{"postgres", true},
			{"mongodb", true},
			{"redis", true},
			{"sqlite", false},
			{"oracle", false},
		}

		for _, tt := range tests {
			cfg := &database.Config{
				Driver: tt.driver,
				DSN:    "test-dsn",
			}
			err := cfg.Validate()
			if tt.wantValid && err != nil {
				t.Errorf("Expected %s to be valid, got error: %v", tt.driver, err)
			}
			if !tt.wantValid && err == nil {
				t.Errorf("Expected %s to be invalid, but it was valid", tt.driver)
			}
		}
	})
}

// TestDatabaseConfigWithPool tests configuration with pool settings.
func TestDatabaseConfigWithPool(t *testing.T) {
	t.Run("Config with pool settings", func(t *testing.T) {
		cfg := &database.Config{
			Driver: "mysql",
			DSN:    "user:pass@tcp(localhost:3306)/db",
			Pool: &database.PoolConfig{
				MaxOpenConns:    25,
				MaxIdleConns:    3,
				ConnMaxLifetime: 20 * time.Minute,
				ConnMaxIdleTime: 3 * time.Minute,
				ConnectTimeout:  5 * time.Second,
			},
		}

		pool := cfg.GetPoolConfig()
		if pool.MaxOpenConns != 25 {
			t.Errorf("Expected 25, got %d", pool.MaxOpenConns)
		}
		if pool.MaxIdleConns != 3 {
			t.Errorf("Expected 3, got %d", pool.MaxIdleConns)
		}
	})
}

// TestDatabaseHealthStatus tests the HealthStatus structure.
func TestDatabaseHealthStatus(t *testing.T) {
	t.Run("HealthStatus fields", func(t *testing.T) {
		status := &database.HealthStatus{
			Status:  "healthy",
			Message: "Connection OK",
			Latency: 10 * time.Millisecond,
			Stats: &database.PoolStats{
				OpenConnections: 5,
				InUse:           2,
				Idle:            3,
			},
		}

		if status.Status != "healthy" {
			t.Errorf("Expected status=healthy, got %s", status.Status)
		}
		if status.Message != "Connection OK" {
			t.Errorf("Expected message=Connection OK, got %s", status.Message)
		}
		if status.Latency != 10*time.Millisecond {
			t.Errorf("Expected latency=10ms, got %v", status.Latency)
		}
		if status.Stats.OpenConnections != 5 {
			t.Errorf("Expected 5 open connections, got %d", status.Stats.OpenConnections)
		}
	})
}
