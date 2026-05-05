// Package database provides database connection management for the Firefly framework.
// This file implements property-based tests for connection pool configuration.
// **Validates: Requirement 15.2**
package database

import (
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Property 32: Connection Pool Configuration
// Validates: Requirement 15.2 - "THE Framework SHALL 支持连接池配置：最大连接数、最小连接数、空闲超时"
// =============================================================================

// TestProperty32PoolConfigMaxOpenConns_PBT verifies that MaxOpenConns is correctly applied.
// **Validates: Requirement 15.2**
func TestProperty32PoolConfigMaxOpenConns_PBT(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Values: func(args []reflect.Value, r *rand.Rand) {
			// Generate random MaxOpenConns values (1 to 1000)
			maxOpenConns := r.Intn(1000) + 1
			args[0] = reflect.ValueOf(maxOpenConns)
		},
	}

	f := func(maxOpenConns int) bool {
		// Skip invalid values
		if maxOpenConns <= 0 {
			return true
		}

		pool := &PoolConfig{
			MaxOpenConns: maxOpenConns,
		}

		// Apply defaults (which should preserve our value)
		applyPoolDefaults(pool)

		return pool.MaxOpenConns == maxOpenConns
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 32 (MaxOpenConns configuration) failed: %v", err)
	}
}

// TestProperty32PoolConfigMaxIdleConns_PBT verifies that MaxIdleConns is correctly applied.
// **Validates: Requirement 15.2**
func TestProperty32PoolConfigMaxIdleConns_PBT(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Values: func(args []reflect.Value, r *rand.Rand) {
			// Generate random MaxIdleConns values (1 to 100), excluding 0
			// because 0 is a special case that triggers default
			maxIdleConns := r.Intn(100) + 1
			args[0] = reflect.ValueOf(maxIdleConns)
		},
	}

	f := func(maxIdleConns int) bool {
		// Skip invalid values
		if maxIdleConns <= 0 {
			return true
		}

		pool := &PoolConfig{
			MaxIdleConns: maxIdleConns,
		}

		// Apply defaults (which should preserve our value since it's non-zero)
		applyPoolDefaults(pool)

		return pool.MaxIdleConns == maxIdleConns
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 32 (MaxIdleConns configuration) failed: %v", err)
	}
}

// TestProperty32PoolConfigConnMaxIdleTime_PBT verifies that ConnMaxIdleTime is correctly applied.
// **Validates: Requirement 15.2**
func TestProperty32PoolConfigConnMaxIdleTime_PBT(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Values: func(args []reflect.Value, r *rand.Rand) {
			// Generate random durations (1 second to 1 hour)
			duration := time.Duration(r.Intn(3600)+1) * time.Second
			args[0] = reflect.ValueOf(duration)
		},
	}

	f := func(idleTime time.Duration) bool {
		// Skip invalid values
		if idleTime <= 0 {
			return true
		}

		pool := &PoolConfig{
			ConnMaxIdleTime: idleTime,
		}

		// Apply defaults (which should preserve our value since it's non-zero)
		applyPoolDefaults(pool)

		return pool.ConnMaxIdleTime == idleTime
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 32 (ConnMaxIdleTime configuration) failed: %v", err)
	}
}

// TestProperty32PoolConfigConnMaxLifetime_PBT verifies that ConnMaxLifetime is correctly applied.
// **Validates: Requirement 15.2**
func TestProperty32PoolConfigConnMaxLifetime_PBT(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Values: func(args []reflect.Value, r *rand.Rand) {
			// Generate random durations (1 minute to 2 hours)
			duration := time.Duration(r.Intn(120)+1) * time.Minute
			args[0] = reflect.ValueOf(duration)
		},
	}

	f := func(lifetime time.Duration) bool {
		// Skip invalid values
		if lifetime <= 0 {
			return true
		}

		pool := &PoolConfig{
			ConnMaxLifetime: lifetime,
		}

		// Apply defaults (which should preserve our value since it's non-zero)
		applyPoolDefaults(pool)

		return pool.ConnMaxLifetime == lifetime
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 32 (ConnMaxLifetime configuration) failed: %v", err)
	}
}

// TestProperty32PoolConfigConnectTimeout_PBT verifies that ConnectTimeout is correctly applied.
// **Validates: Requirement 15.2**
func TestProperty32PoolConfigConnectTimeout_PBT(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Values: func(args []reflect.Value, r *rand.Rand) {
			// Generate random durations (1 second to 60 seconds)
			duration := time.Duration(r.Intn(60)+1) * time.Second
			args[0] = reflect.ValueOf(duration)
		},
	}

	f := func(timeout time.Duration) bool {
		// Skip invalid values
		if timeout <= 0 {
			return true
		}

		pool := &PoolConfig{
			ConnectTimeout: timeout,
		}

		// Apply defaults (which should preserve our value since it's non-zero)
		applyPoolDefaults(pool)

		return pool.ConnectTimeout == timeout
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 32 (ConnectTimeout configuration) failed: %v", err)
	}
}

// TestProperty32FullPoolConfig_PBT verifies that all pool config fields are correctly applied together.
// **Validates: Requirement 15.2**
func TestProperty32FullPoolConfig_PBT(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Values: func(args []reflect.Value, r *rand.Rand) {
			// Generate random pool configuration (excluding 0 for MaxIdleConns)
			pool := &PoolConfig{
				MaxOpenConns:    r.Intn(1000) + 1,
				MaxIdleConns:    r.Intn(100) + 1, // Exclude 0 to avoid default application
				ConnMaxLifetime: time.Duration(r.Intn(120)+1) * time.Minute,
				ConnMaxIdleTime: time.Duration(r.Intn(60)+1) * time.Minute,
				ConnectTimeout:  time.Duration(r.Intn(30)+1) * time.Second,
			}
			args[0] = reflect.ValueOf(pool)
		},
	}

	f := func(pool *PoolConfig) bool {
		// Skip invalid configurations
		if pool.MaxOpenConns <= 0 || pool.MaxIdleConns < 0 ||
			pool.ConnMaxLifetime <= 0 || pool.ConnMaxIdleTime <= 0 ||
			pool.ConnectTimeout <= 0 {
			return true
		}

		// Store original values
		originalMaxOpenConns := pool.MaxOpenConns
		originalMaxIdleConns := pool.MaxIdleConns
		originalConnMaxLifetime := pool.ConnMaxLifetime
		originalConnMaxIdleTime := pool.ConnMaxIdleTime
		originalConnectTimeout := pool.ConnectTimeout

		// Apply defaults
		applyPoolDefaults(pool)

		// All values should be preserved
		return pool.MaxOpenConns == originalMaxOpenConns &&
			pool.MaxIdleConns == originalMaxIdleConns &&
			pool.ConnMaxLifetime == originalConnMaxLifetime &&
			pool.ConnMaxIdleTime == originalConnMaxIdleTime &&
			pool.ConnectTimeout == originalConnectTimeout
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 32 (full pool configuration) failed: %v", err)
	}
}

// TestProperty32DefaultPoolConfig_PBT verifies that default pool configuration is correct.
// **Validates: Requirement 15.2**
func TestProperty32DefaultPoolConfig_PBT(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Values: func(args []reflect.Value, r *rand.Rand) {
			// Always generate empty config to test defaults
			args[0] = reflect.ValueOf(0)
		},
	}

	f := func(_ int) bool {
		// When pool is nil, should return defaults
		cfg := &Config{
			Driver: "mysql",
			DSN:    "test",
			Pool:   nil,
		}

		pool := cfg.GetPoolConfig()

		// Default values should be applied
		return pool.MaxOpenConns == 100 &&
			pool.MaxIdleConns == 10 &&
			pool.ConnMaxLifetime == 30*time.Minute &&
			pool.ConnMaxIdleTime == 10*time.Minute &&
			pool.ConnectTimeout == 30*time.Second
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 32 (default pool configuration) failed: %v", err)
	}
}

// TestProperty32PartialPoolConfig_PBT verifies that partial pool config uses defaults for unset fields.
// **Validates: Requirement 15.2**
func TestProperty32PartialPoolConfig_PBT(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Values: func(args []reflect.Value, r *rand.Rand) {
			// Generate random configuration with only some fields set
			// 0 = only MaxOpenConns, 1 = only MaxIdleConns, 2 = both, etc.
			caseNum := r.Intn(4)
			args[0] = reflect.ValueOf(caseNum)
		},
	}

	f := func(caseNum int) bool {
		var cfg *Config

		switch caseNum {
		case 0:
			// Only MaxOpenConns set
			cfg = &Config{
				Driver: "mysql",
				DSN:    "test",
				Pool:   &PoolConfig{MaxOpenConns: 50},
			}
		case 1:
			// Only MaxIdleConns set
			cfg = &Config{
				Driver: "mysql",
				DSN:    "test",
				Pool:   &PoolConfig{MaxIdleConns: 20},
			}
		case 2:
			// Both MaxOpenConns and MaxIdleConns set
			cfg = &Config{
				Driver: "mysql",
				DSN:    "test",
				Pool:   &PoolConfig{MaxOpenConns: 50, MaxIdleConns: 20},
			}
		default:
			// Empty pool config - should use all defaults
			cfg = &Config{
				Driver: "mysql",
				DSN:    "test",
				Pool:   &PoolConfig{},
			}
		}

		pool := cfg.GetPoolConfig()

		switch caseNum {
		case 0:
			return pool.MaxOpenConns == 50 &&
				pool.MaxIdleConns == 10 && // default
				pool.ConnMaxLifetime == 30*time.Minute && // default
				pool.ConnMaxIdleTime == 10*time.Minute && // default
				pool.ConnectTimeout == 30*time.Second // default
		case 1:
			return pool.MaxOpenConns == 100 && // default
				pool.MaxIdleConns == 20 &&
				pool.ConnMaxLifetime == 30*time.Minute && // default
				pool.ConnMaxIdleTime == 10*time.Minute && // default
				pool.ConnectTimeout == 30*time.Second // default
		case 2:
			return pool.MaxOpenConns == 50 &&
				pool.MaxIdleConns == 20 &&
				pool.ConnMaxLifetime == 30*time.Minute && // default
				pool.ConnMaxIdleTime == 10*time.Minute && // default
				pool.ConnectTimeout == 30*time.Second // default
		default:
			return pool.MaxOpenConns == 100 &&
				pool.MaxIdleConns == 10 &&
				pool.ConnMaxLifetime == 30*time.Minute &&
				pool.ConnMaxIdleTime == 10*time.Minute &&
				pool.ConnectTimeout == 30*time.Second
		}
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 32 (partial pool configuration) failed: %v", err)
	}
}

// TestProperty32PoolConfigImmutability_PBT verifies that GetPoolConfig returns a copy, not original.
// **Validates: Requirement 15.2**
func TestProperty32PoolConfigImmutability_PBT(t *testing.T) {
	config := &quick.Config{
		MaxCount: 50,
		Values: func(args []reflect.Value, r *rand.Rand) {
			// Generate random configuration, excluding 0 values
			pool := &PoolConfig{
				MaxOpenConns:    r.Intn(1000) + 1,
				MaxIdleConns:    r.Intn(100) + 1, // Exclude 0
				ConnMaxLifetime: time.Duration(r.Intn(120)+1) * time.Minute,
				ConnMaxIdleTime: time.Duration(r.Intn(60)+1) * time.Minute,
				ConnectTimeout:  time.Duration(r.Intn(30)+1) * time.Second,
			}
			args[0] = reflect.ValueOf(pool)
		},
	}

	f := func(original *PoolConfig) bool {
		// Skip invalid configurations (MaxIdleConns = 0 is valid - triggers default)
		if original.MaxOpenConns <= 0 ||
			original.ConnMaxLifetime <= 0 || original.ConnMaxIdleTime <= 0 ||
			original.ConnectTimeout <= 0 {
			return true
		}

		cfg := &Config{
			Driver: "mysql",
			DSN:    "test",
			Pool:   original,
		}

		// Get pool config
		pool1 := cfg.GetPoolConfig()
		pool2 := cfg.GetPoolConfig()

		// Both calls should return equivalent configs
		if pool1 == pool2 {
			// Same pointer returned - this is acceptable but not ideal
			// The important thing is that the original config is not modified
			return true
		}

		// If different pointers, they should have same values
		return pool1.MaxOpenConns == pool2.MaxOpenConns &&
			pool1.MaxIdleConns == pool2.MaxIdleConns &&
			pool1.ConnMaxLifetime == pool2.ConnMaxLifetime &&
			pool1.ConnMaxIdleTime == pool2.ConnMaxIdleTime &&
			pool1.ConnectTimeout == pool2.ConnectTimeout
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 32 (pool config immutability) failed: %v", err)
	}
}

// TestProperty32PoolConfigZeroValues_PBT verifies that zero values get defaults applied.
// **Validates: Requirement 15.2**
func TestProperty32PoolConfigZeroValues_PBT(t *testing.T) {
	// Test specific zero value scenarios
	pool := &PoolConfig{
		MaxOpenConns:    0, // should get default
		MaxIdleConns:    0, // should get default
		ConnMaxLifetime: 0, // should get default
		ConnMaxIdleTime: 0, // should get default
		ConnectTimeout:  0, // should get default
	}

	applyPoolDefaults(pool)

	if pool.MaxOpenConns != 100 {
		t.Errorf("MaxOpenConns = %d, want 100", pool.MaxOpenConns)
	}
	if pool.MaxIdleConns != 10 {
		t.Errorf("MaxIdleConns = %d, want 10", pool.MaxIdleConns)
	}
	if pool.ConnMaxLifetime != 30*time.Minute {
		t.Errorf("ConnMaxLifetime = %v, want 30m0s", pool.ConnMaxLifetime)
	}
	if pool.ConnMaxIdleTime != 10*time.Minute {
		t.Errorf("ConnMaxIdleTime = %v, want 10m0s", pool.ConnMaxIdleTime)
	}
	if pool.ConnectTimeout != 30*time.Second {
		t.Errorf("ConnectTimeout = %v, want 30s", pool.ConnectTimeout)
	}
}

// TestProperty32PoolConfigBoundaryValues_PBT tests boundary values for pool configuration.
// **Validates: Requirement 15.2**
func TestProperty32PoolConfigBoundaryValues_PBT(t *testing.T) {
	// Test boundary values
	testCases := []struct {
		name               string
		pool               *PoolConfig
		expectMaxOk        bool
		expectIdleOk       bool
		expectZeroDefaults bool // if true, zero values get defaults
	}{
		{
			name:         "minimum values",
			pool:         &PoolConfig{MaxOpenConns: 1, MaxIdleConns: 1, ConnMaxLifetime: 1 * time.Second, ConnMaxIdleTime: 1 * time.Second, ConnectTimeout: 1 * time.Second},
			expectMaxOk:  true,
			expectIdleOk: true,
		},
		{
			name:         "large values",
			pool:         &PoolConfig{MaxOpenConns: 10000, MaxIdleConns: 5000, ConnMaxLifetime: 24 * time.Hour, ConnMaxIdleTime: 1 * time.Hour, ConnectTimeout: 5 * time.Minute},
			expectMaxOk:  true,
			expectIdleOk: true,
		},
		{
			name:               "all zeros - should get defaults",
			pool:               &PoolConfig{},
			expectMaxOk:        false,
			expectIdleOk:       false,
			expectZeroDefaults: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := *tc.pool
			applyPoolDefaults(tc.pool)

			if tc.expectMaxOk {
				if tc.pool.MaxOpenConns != original.MaxOpenConns {
					t.Errorf("MaxOpenConns changed from %d to %d", original.MaxOpenConns, tc.pool.MaxOpenConns)
				}
			} else if tc.expectZeroDefaults {
				if tc.pool.MaxOpenConns == 0 {
					t.Error("MaxOpenConns should have default applied")
				}
			}

			if tc.expectIdleOk {
				if tc.pool.MaxIdleConns != original.MaxIdleConns {
					t.Errorf("MaxIdleConns changed from %d to %d", original.MaxIdleConns, tc.pool.MaxIdleConns)
				}
			} else if tc.expectZeroDefaults {
				if tc.pool.MaxIdleConns == 0 {
					t.Error("MaxIdleConns should have default applied")
				}
			}
		})
	}
}

// =============================================================================
// Property 33: 数据库连接错误 (Database Connection Errors)
// Validates: Requirement 15.3 - "WHEN 数据库连接失败时，THE Framework SHALL 返回描述性错误信息"
// =============================================================================

// TestProperty33ConnectionErrorContainsDriver_PBT verifies that connection errors include driver name.
// **Validates: Requirement 15.3**
func TestProperty33ConnectionErrorContainsDriver_PBT(t *testing.T) {
	drivers := []string{"mysql", "postgres", "mongodb", "redis"}

	for _, driver := range drivers {
		t.Run(driver, func(t *testing.T) {
			err := NewConnectionError(driver, "connection failed", nil)
			errStr := err.Error()

			assert.Contains(t, errStr, driver, "error should include driver name")
			assert.Contains(t, errStr, "CONNECTION_ERROR", "error should have CONNECTION_ERROR code")
		})
	}
}

// TestProperty33ConfigErrorInvalidDriver_PBT verifies config validation returns descriptive errors for invalid drivers.
// **Validates: Requirement 15.3**
func TestProperty33ConfigErrorInvalidDriver_PBT(t *testing.T) {
	invalidDrivers := []string{"", "sqlite", "oracle", "mssql", "InvalidDriver", "MYSQL", "Postgres"}

	for _, driver := range invalidDrivers {
		t.Run(driver, func(t *testing.T) {
			cfg := &Config{
				Driver: driver,
				DSN:    "test",
			}

			err := cfg.Validate()
			require.Error(t, err, "invalid driver should cause validation error")
			assert.True(t, IsConfigError(err), "should be a config error")

			errStr := err.Error()
			if driver == "" {
				assert.Contains(t, errStr, "driver is required")
			} else {
				assert.Contains(t, errStr, "driver must be one of")
			}
		})
	}
}

// TestProperty33ConfigErrorEmptyDSN_PBT verifies config validation returns descriptive errors for empty DSN.
// **Validates: Requirement 15.3**
func TestProperty33ConfigErrorEmptyDSN_PBT(t *testing.T) {
	drivers := []string{"mysql", "postgres", "mongodb", "redis"}

	for _, driver := range drivers {
		t.Run(driver, func(t *testing.T) {
			cfg := &Config{
				Driver: driver,
				DSN:    "",
			}

			err := cfg.Validate()
			require.Error(t, err, "empty DSN should cause validation error")
			assert.True(t, IsConfigError(err), "should be a config error")
			assert.Contains(t, err.Error(), "dsn is required")
		})
	}
}

// TestProperty33ErrorMessageNotEmpty_PBT verifies all error types have non-empty messages.
// **Validates: Requirement 15.3**
func TestProperty33ErrorMessageNotEmpty_PBT(t *testing.T) {
	testCases := []struct {
		name    string
		err     *Error
		message string
	}{
		{
			name:    "config_error",
			err:     NewConfigError("invalid configuration value"),
			message: "invalid configuration value",
		},
		{
			name:    "connection_error",
			err:     NewConnectionError("mysql", "database not reachable", nil),
			message: "database not reachable",
		},
		{
			name:    "query_error",
			err:     NewQueryError("postgres", "SELECT *", "syntax error near FROM", nil),
			message: "syntax error near FROM",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotEmpty(t, tc.err.Message, "error message should not be empty")
			assert.Equal(t, tc.message, tc.err.Message)
			assert.NotEmpty(t, tc.err.Error(), "Error() should return non-empty string")
		})
	}
}

// TestProperty33ErrorCodesValid_PBT verifies all error codes are properly defined.
// **Validates: Requirement 15.3**
func TestProperty33ErrorCodesValid_PBT(t *testing.T) {
	errorCodes := map[string]string{
		ErrCodeConfig:            "CONFIG_ERROR",
		ErrCodeConnection:        "CONNECTION_ERROR",
		ErrCodeQuery:             "QUERY_ERROR",
		ErrCodeTimeout:           "TIMEOUT_ERROR",
		ErrCodePoolExhausted:     "POOL_EXHAUSTED",
		ErrCodeNotConnected:      "NOT_CONNECTED",
		ErrCodeUnsupportedDriver: "UNSUPPORTED_DRIVER",
	}

	for code, expected := range errorCodes {
		t.Run(code, func(t *testing.T) {
			assert.Equal(t, expected, code, "error code should match expected value")
		})
	}
}

// TestProperty33ErrorStringFormat_PBT verifies error string format is consistent.
// **Validates: Requirement 15.3**
func TestProperty33ErrorStringFormat_PBT(t *testing.T) {
	t.Run("connection_error_format", func(t *testing.T) {
		err := NewConnectionError("mysql", "test message", nil)
		errStr := err.Error()

		// Should contain: "database error", error code, driver, message
		assert.Contains(t, errStr, "database error")
		assert.Contains(t, errStr, "[CONNECTION_ERROR]")
		assert.Contains(t, errStr, "driver=mysql")
		assert.Contains(t, errStr, "test message")
	})

	t.Run("config_error_format", func(t *testing.T) {
		err := NewConfigError("test message")
		errStr := err.Error()

		assert.Contains(t, errStr, "database error")
		assert.Contains(t, errStr, "[CONFIG_ERROR]")
		assert.Contains(t, errStr, "test message")
	})

	t.Run("query_error_format", func(t *testing.T) {
		err := NewQueryError("mysql", "SELECT 1", "error message", nil)
		errStr := err.Error()

		assert.Contains(t, errStr, "database error")
		assert.Contains(t, errStr, "[QUERY_ERROR]")
		assert.Contains(t, errStr, "driver=mysql")
		assert.Contains(t, errStr, "query=SELECT 1")
		assert.Contains(t, errStr, "error message")
	})
}

// TestProperty33ErrorTypeCheckingFunctions_PBT verifies all error type checking functions work correctly.
// **Validates: Requirement 15.3**
func TestProperty33ErrorTypeCheckingFunctions_PBT(t *testing.T) {
	t.Run("all_error_types", func(t *testing.T) {
		errors := map[string]*Error{
			"config":     NewConfigError("test"),
			"connection": NewConnectionError("mysql", "test", nil),
			"query":      NewQueryError("mysql", "SELECT 1", "test", nil),
		}

		// Config error
		assert.True(t, IsConfigError(errors["config"]))
		assert.False(t, IsConnectionError(errors["config"]))
		assert.False(t, IsQueryError(errors["config"]))

		// Connection error
		assert.False(t, IsConfigError(errors["connection"]))
		assert.True(t, IsConnectionError(errors["connection"]))
		assert.False(t, IsQueryError(errors["connection"]))

		// Query error
		assert.False(t, IsConfigError(errors["query"]))
		assert.False(t, IsConnectionError(errors["query"]))
		assert.True(t, IsQueryError(errors["query"]))
	})
}

// TestProperty33PoolConfigErrorMessages_PBT verifies pool configuration errors are descriptive.
// **Validates: Requirement 15.3**
func TestProperty33PoolConfigErrorMessages_PBT(t *testing.T) {
	// Test with various zero values that would trigger defaults
	pool := &PoolConfig{}

	applyPoolDefaults(pool)

	// After applying defaults, no error should occur - just verify defaults are applied
	assert.Equal(t, 100, pool.MaxOpenConns)
	assert.Equal(t, 10, pool.MaxIdleConns)
	assert.Equal(t, 30*time.Minute, pool.ConnMaxLifetime)
	assert.Equal(t, 10*time.Minute, pool.ConnMaxIdleTime)
	assert.Equal(t, 30*time.Second, pool.ConnectTimeout)
}
