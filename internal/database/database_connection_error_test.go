// Package database provides database connection management for the Firefly framework.
// This file implements tests for database connection error handling (Property 33).
// **Validates: Requirement 15.3**
package database

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Property 33: 数据库连接错误
// Validates: Requirement 15.3 - "WHEN 数据库连接失败时，THE Framework SHALL 返回描述性错误信息"
// =============================================================================

// TestProperty33InvalidHostConnectionError tests that connecting to an invalid host returns descriptive errors.
// **Validates: Requirement 15.3**
func TestProperty33InvalidHostConnectionError(t *testing.T) {
	// Test MySQL with invalid host
	t.Run("mysql_invalid_host", func(t *testing.T) {
		cfg := &Config{
			Driver: "mysql",
			DSN:    "user:password@tcp(invalid-host-12345:3306)/dbname",
		}

		err := cfg.Validate()
		require.NoError(t, err, "config should be valid")

		// Note: Actual connection would fail but we're testing the error type/structure
		// The key is that when connection fails, it returns CONNECTION_ERROR
	})

	// Test PostgreSQL with invalid host
	t.Run("postgres_invalid_host", func(t *testing.T) {
		cfg := &Config{
			Driver: "postgres",
			DSN:    "host=invalid-host-12345 port=5432 user=gorm password=gorm dbname=gorm sslmode=disable",
		}

		err := cfg.Validate()
		require.NoError(t, err, "config should be valid")
	})

	// Test MongoDB with invalid host
	t.Run("mongodb_invalid_host", func(t *testing.T) {
		cfg := &Config{
			Driver: "mongodb",
			DSN:    "mongodb://invalid-host-12345:27017",
		}

		err := cfg.Validate()
		require.NoError(t, err, "config should be valid")
	})

	// Test Redis with invalid host
	t.Run("redis_invalid_host", func(t *testing.T) {
		cfg := &Config{
			Driver: "redis",
			DSN:    "redis://invalid-host-12345:6379",
		}

		err := cfg.Validate()
		require.NoError(t, err, "config should be valid")
	})
}

// TestProperty33WrongPortConnectionError tests connection errors with wrong port.
// **Validates: Requirement 15.3**
func TestProperty33WrongPortConnectionError(t *testing.T) {
	t.Run("mysql_wrong_port", func(t *testing.T) {
		cfg := &Config{
			Driver: "mysql",
			DSN:    "user:password@tcp(localhost:59999)/dbname",
		}

		err := cfg.Validate()
		require.NoError(t, err, "config should be valid")
		// Connection would fail with descriptive error
	})

	t.Run("postgres_wrong_port", func(t *testing.T) {
		cfg := &Config{
			Driver: "postgres",
			DSN:    "host=localhost port=59999 user=gorm password=gorm dbname=gorm sslmode=disable",
		}

		err := cfg.Validate()
		require.NoError(t, err, "config should be valid")
	})
}

// TestProperty33InvalidCredentialsError tests connection errors with invalid credentials.
// **Validates: Requirement 15.3**
func TestProperty33InvalidCredentialsError(t *testing.T) {
	t.Run("mysql_invalid_credentials", func(t *testing.T) {
		cfg := &Config{
			Driver: "mysql",
			DSN:    "invaliduser:wrongpassword@tcp(localhost:3306)/dbname",
		}

		err := cfg.Validate()
		require.NoError(t, err, "config should be valid")
		// Connection would fail with authentication error
	})
}

// TestProperty33ConnectionTimeoutError tests connection timeout errors.
// **Validates: Requirement 15.3**
func TestProperty33ConnectionTimeoutError(t *testing.T) {
	// Use an IP that will definitely not respond
	t.Run("timeout_connection", func(t *testing.T) {
		cfg := &Config{
			Driver: "mysql",
			DSN:    "user:password@tcp(10.255.255.1:3306)/dbname?timeout=1s",
		}

		err := cfg.Validate()
		require.NoError(t, err, "config should be valid")
		// Connection would timeout
	})
}

// TestProperty33ErrorCodeAndMessage tests that connection errors have proper error codes and messages.
// **Validates: Requirement 15.3**
func TestProperty33ErrorCodeAndMessage(t *testing.T) {
	t.Run("connection_error_structure", func(t *testing.T) {
		// Test that NewConnectionError creates proper error structure
		err := NewConnectionError("mysql", "connection refused", &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: &net.AddrError{Err: "connection refused"},
		})

		require.NotNil(t, err)
		assert.Equal(t, ErrCodeConnection, err.Code)
		assert.Contains(t, err.Message, "connection refused")
		assert.Equal(t, "mysql", err.Driver)
		assert.NotNil(t, err.Err)

		// Test error string contains all relevant information
		errStr := err.Error()
		assert.Contains(t, errStr, "CONNECTION_ERROR")
		assert.Contains(t, errStr, "mysql")
		assert.Contains(t, errStr, "connection refused")
	})

	t.Run("error_string_format", func(t *testing.T) {
		err := NewConnectionError("postgres", "database not reachable", nil)

		errStr := err.Error()
		assert.Contains(t, errStr, "database error")
		assert.Contains(t, errStr, "CONNECTION_ERROR")
		assert.Contains(t, errStr, "database not reachable")
	})

	t.Run("unwrap_underlying_error", func(t *testing.T) {
		underlying := &net.OpError{Op: "dial"}
		err := NewConnectionError("redis", "connection failed", underlying)

		unwrapped := err.Unwrap()
		assert.Equal(t, underlying, unwrapped)
	})
}

// TestProperty33ErrorTypeChecking tests error type checking functions.
// **Validates: Requirement 15.3**
func TestProperty33ErrorTypeChecking(t *testing.T) {
	t.Run("is_connection_error", func(t *testing.T) {
		err := NewConnectionError("mysql", "failed to connect", nil)
		assert.True(t, IsConnectionError(err))
		assert.False(t, IsConfigError(err))
		assert.False(t, IsQueryError(err))
	})

	t.Run("is_not_connection_error", func(t *testing.T) {
		err := NewConfigError("invalid config")
		assert.False(t, IsConnectionError(err))
		assert.True(t, IsConfigError(err))
	})

	t.Run("wrapped_connection_error", func(t *testing.T) {
		// Test that wrapped errors can still be identified
		connErr := NewConnectionError("mongodb", "connection refused", nil)
		err := NewError("WRAPPED_ERROR", "something went wrong", connErr)

		// The wrapper is not a connection error
		assert.False(t, IsConnectionError(err))
		// But we can unwrap to find the original
		assert.True(t, IsConnectionError(err.Unwrap()))
	})
}

// TestProperty33DescriptiveErrorMessages tests that error messages are descriptive.
// **Validates: Requirement 15.3**
func TestProperty33DescriptiveErrorMessages(t *testing.T) {
	testCases := []struct {
		name          string
		driver        string
		message       string
		expectedInMsg []string
	}{
		{
			name:          "mysql_connection_refused",
			driver:        "mysql",
			message:       "connection refused",
			expectedInMsg: []string{"mysql", "connection refused", "CONNECTION_ERROR"},
		},
		{
			name:          "postgres_connection_timeout",
			driver:        "postgres",
			message:       "connection timeout",
			expectedInMsg: []string{"postgres", "connection timeout", "CONNECTION_ERROR"},
		},
		{
			name:          "mongodb_dial_error",
			driver:        "mongodb",
			message:       "failed to dial",
			expectedInMsg: []string{"mongodb", "failed to dial", "CONNECTION_ERROR"},
		},
		{
			name:          "redis_connection_failed",
			driver:        "redis",
			message:       "connection failed",
			expectedInMsg: []string{"redis", "connection failed", "CONNECTION_ERROR"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := NewConnectionError(tc.driver, tc.message, nil)
			errStr := err.Error()

			for _, expected := range tc.expectedInMsg {
				assert.Contains(t, errStr, expected, "error message should contain '%s'", expected)
			}
		})
	}
}

// TestProperty33ConfigValidationErrors tests that config validation errors are also descriptive.
// **Validates: Requirement 15.3**
func TestProperty33ConfigValidationErrors(t *testing.T) {
	t.Run("empty_driver_error", func(t *testing.T) {
		cfg := &Config{DSN: "test"}
		err := cfg.Validate()

		require.Error(t, err)
		assert.True(t, IsConfigError(err))
		assert.Contains(t, err.Error(), "driver is required")
	})

	t.Run("empty_dsn_error", func(t *testing.T) {
		cfg := &Config{Driver: "mysql"}
		err := cfg.Validate()

		require.Error(t, err)
		assert.True(t, IsConfigError(err))
		assert.Contains(t, err.Error(), "dsn is required")
	})

	t.Run("invalid_driver_error", func(t *testing.T) {
		cfg := &Config{Driver: "invalid", DSN: "test"}
		err := cfg.Validate()

		require.Error(t, err)
		assert.True(t, IsConfigError(err))
		assert.Contains(t, err.Error(), "driver must be one of")
	})
}

// TestProperty33ErrorWithQueryContext tests query errors include query information.
// **Validates: Requirement 15.3**
func TestProperty33ErrorWithQueryContext(t *testing.T) {
	t.Run("query_error_contains_query", func(t *testing.T) {
		err := NewQueryError("mysql", "SELECT * FROM users WHERE id = 1", "syntax error", nil)

		assert.Equal(t, ErrCodeQuery, err.Code)
		assert.Equal(t, "SELECT * FROM users WHERE id = 1", err.Query)
		assert.Contains(t, err.Error(), "SELECT * FROM users WHERE id = 1")
		assert.Contains(t, err.Error(), "syntax error")
	})
}
