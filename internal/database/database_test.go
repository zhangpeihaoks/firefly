// Package database provides database connection management for the Firefly framework.
// This file implements tests for database interfaces and configurations.
package database

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigValidate tests the Config.Validate method.
func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty driver",
			config:  &Config{DSN: "test"},
			wantErr: true,
			errMsg:  "driver is required",
		},
		{
			name:    "invalid driver",
			config:  &Config{Driver: "invalid", DSN: "test"},
			wantErr: true,
			errMsg:  "driver must be one of",
		},
		{
			name:    "empty dsn",
			config:  &Config{Driver: "mysql"},
			wantErr: true,
			errMsg:  "dsn is required",
		},
		{
			name:    "valid mysql config",
			config:  &Config{Driver: "mysql", DSN: "user:pass@tcp(localhost:3306)/db"},
			wantErr: false,
		},
		{
			name:    "valid postgres config",
			config:  &Config{Driver: "postgres", DSN: "postgres://user:pass@localhost:5432/db"},
			wantErr: false,
		},
		{
			name:    "valid mongodb config",
			config:  &Config{Driver: "mongodb", DSN: "mongodb://localhost:27017"},
			wantErr: false,
		},
		{
			name:    "valid redis config",
			config:  &Config{Driver: "redis", DSN: "redis://localhost:6379"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestGetPoolConfig tests the Config.GetPoolConfig method.
func TestGetPoolConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected *PoolConfig
	}{
		{
			name:   "nil pool uses defaults",
			config: &Config{Driver: "mysql", DSN: "test"},
			expected: &PoolConfig{
				MaxOpenConns:    100,
				MaxIdleConns:    10,
				ConnMaxLifetime: 30 * time.Minute,
				ConnMaxIdleTime: 10 * time.Minute,
				ConnectTimeout:  30 * time.Second,
			},
		},
		{
			name: "partial pool config uses defaults for unset fields",
			config: &Config{
				Driver: "mysql",
				DSN:    "test",
				Pool:   &PoolConfig{MaxOpenConns: 50},
			},
			expected: &PoolConfig{
				MaxOpenConns:    50,
				MaxIdleConns:    10,
				ConnMaxLifetime: 30 * time.Minute,
				ConnMaxIdleTime: 10 * time.Minute,
				ConnectTimeout:  30 * time.Second,
			},
		},
		{
			name: "full pool config",
			config: &Config{
				Driver: "mysql",
				DSN:    "test",
				Pool: &PoolConfig{
					MaxOpenConns:    50,
					MaxIdleConns:    5,
					ConnMaxLifetime: 1 * time.Hour,
					ConnMaxIdleTime: 30 * time.Minute,
					ConnectTimeout:  10 * time.Second,
				},
			},
			expected: &PoolConfig{
				MaxOpenConns:    50,
				MaxIdleConns:    5,
				ConnMaxLifetime: 1 * time.Hour,
				ConnMaxIdleTime: 30 * time.Minute,
				ConnectTimeout:  10 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetPoolConfig()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDefaultPoolConfig tests the DefaultPoolConfig function.
func TestDefaultPoolConfig(t *testing.T) {
	config := DefaultPoolConfig()
	assert.Equal(t, 100, config.MaxOpenConns)
	assert.Equal(t, 10, config.MaxIdleConns)
	assert.Equal(t, 30*time.Minute, config.ConnMaxLifetime)
	assert.Equal(t, 10*time.Minute, config.ConnMaxIdleTime)
	assert.Equal(t, 30*time.Second, config.ConnectTimeout)
}

// TestPoolStats tests that PoolStats has all expected fields.
func TestPoolStats(t *testing.T) {
	stats := &PoolStats{
		MaxOpenConnections: 100,
		OpenConnections:    50,
		InUse:              20,
		Idle:               30,
		WaitCount:          10,
		WaitDuration:       5 * time.Second,
		MaxIdleClosed:      5,
		MaxIdleTimeClosed:  3,
		MaxLifetimeClosed:  2,
		Hits:               1000,
		Misses:             10,
		Timeouts:           1,
		TotalConns:         50,
		IdleConns:          30,
		StaleConns:         5,
	}

	assert.Equal(t, 100, stats.MaxOpenConnections)
	assert.Equal(t, 50, stats.OpenConnections)
	assert.Equal(t, 20, stats.InUse)
	assert.Equal(t, 30, stats.Idle)
	assert.Equal(t, int64(10), stats.WaitCount)
	assert.Equal(t, 5*time.Second, stats.WaitDuration)
	assert.Equal(t, int64(5), stats.MaxIdleClosed)
	assert.Equal(t, int64(3), stats.MaxIdleTimeClosed)
	assert.Equal(t, int64(2), stats.MaxLifetimeClosed)
	assert.Equal(t, int64(1000), stats.Hits)
	assert.Equal(t, int64(10), stats.Misses)
	assert.Equal(t, int64(1), stats.Timeouts)
	assert.Equal(t, int64(50), stats.TotalConns)
	assert.Equal(t, int64(30), stats.IdleConns)
	assert.Equal(t, int64(5), stats.StaleConns)
}

// TestError tests database error creation and checking.
func TestError(t *testing.T) {
	t.Run("config error", func(t *testing.T) {
		err := NewConfigError("invalid configuration")
		assert.True(t, IsConfigError(err))
		assert.Contains(t, err.Error(), "CONFIG_ERROR")
	})

	t.Run("connection error", func(t *testing.T) {
		err := NewConnectionError("mysql", "connection refused", nil)
		assert.True(t, IsConnectionError(err))
		assert.Contains(t, err.Error(), "CONNECTION_ERROR")
		assert.Contains(t, err.Error(), "mysql")
	})

	t.Run("query error", func(t *testing.T) {
		err := NewQueryError("postgres", "SELECT * FROM users", "syntax error", nil)
		assert.True(t, IsQueryError(err))
		assert.Contains(t, err.Error(), "QUERY_ERROR")
		assert.Contains(t, err.Error(), "SELECT * FROM users")
	})
}

// TestHealthStatus tests the HealthStatus structure.
func TestHealthStatus(t *testing.T) {
	status := &HealthStatus{
		Status:  "healthy",
		Message: "Connection is healthy",
		Latency: 5 * time.Millisecond,
		Stats: &PoolStats{
			OpenConnections: 10,
			InUse:           5,
			Idle:            5,
		},
	}

	assert.Equal(t, "healthy", status.Status)
	assert.Equal(t, "Connection is healthy", status.Message)
	assert.Equal(t, 5*time.Millisecond, status.Latency)
	assert.NotNil(t, status.Stats)
}

// TestDatabaseType tests the database type constants.
func TestDatabaseType(t *testing.T) {
	assert.Equal(t, DatabaseType("mysql"), TypeMySQL)
	assert.Equal(t, DatabaseType("postgres"), TypePostgres)
	assert.Equal(t, DatabaseType("mongodb"), TypeMongoDB)
	assert.Equal(t, DatabaseType("redis"), TypeRedis)
}
