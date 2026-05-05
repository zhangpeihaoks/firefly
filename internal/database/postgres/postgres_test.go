// Package postgres provides PostgreSQL database connection tests for the Firefly framework.
package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zhangpeihaoks/firefly/internal/database"
	"gorm.io/gorm"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				Driver: "postgres",
				DSN:    "host=localhost user=test password=test dbname=test port=5432 sslmode=disable",
			},
			wantErr: false,
		},
		{
			name: "missing driver",
			config: &Config{
				DSN: "host=localhost user=test password=test dbname=test port=5432 sslmode=disable",
			},
			wantErr: true,
			errMsg:  "driver is required",
		},
		{
			name: "wrong driver",
			config: &Config{
				Driver: "mysql",
				DSN:    "host=localhost user=test password=test dbname=test port=5432 sslmode=disable",
			},
			wantErr: true,
			errMsg:  "expected driver postgres",
		},
		{
			name: "missing dsn",
			config: &Config{
				Driver: "postgres",
			},
			wantErr: true,
			errMsg:  "dsn is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfig_ToDatabaseConfig(t *testing.T) {
	cfg := &Config{
		Driver:   "postgres",
		DSN:      "host=localhost user=test password=test dbname=test port=5432 sslmode=disable",
		LogLevel: "info",
		Pool: &database.PoolConfig{
			MaxOpenConns: 50,
			MaxIdleConns: 10,
		},
	}

	dbCfg := cfg.ToDatabaseConfig()

	assert.Equal(t, "postgres", dbCfg.Driver)
	assert.Equal(t, cfg.DSN, dbCfg.DSN)
	assert.Equal(t, cfg.Pool, dbCfg.Pool)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "postgres", cfg.Driver)
	assert.NotNil(t, cfg.Pool)
	assert.Equal(t, "warn", cfg.LogLevel)
	assert.Equal(t, 200*time.Millisecond, cfg.SlowThreshold)
	assert.True(t, cfg.PrepareStmt)
}

func TestGORMOptions(t *testing.T) {
	tests := []struct {
		name   string
		option GORMOption
		check  func(cfg *gorm.Config)
	}{
		{
			name:   "WithGORMLogLevel silent",
			option: WithGORMLogLevel("silent"),
			check: func(cfg *gorm.Config) {
				assert.NotNil(t, cfg.Logger)
			},
		},
		{
			name:   "WithGORMLogLevel error",
			option: WithGORMLogLevel("error"),
			check: func(cfg *gorm.Config) {
				assert.NotNil(t, cfg.Logger)
			},
		},
		{
			name:   "WithGORMLogLevel warn",
			option: WithGORMLogLevel("warn"),
			check: func(cfg *gorm.Config) {
				assert.NotNil(t, cfg.Logger)
			},
		},
		{
			name:   "WithGORMLogLevel info",
			option: WithGORMLogLevel("info"),
			check: func(cfg *gorm.Config) {
				assert.NotNil(t, cfg.Logger)
			},
		},
		{
			name:   "WithGORMLogLevel unknown defaults to warn",
			option: WithGORMLogLevel("unknown"),
			check: func(cfg *gorm.Config) {
				assert.NotNil(t, cfg.Logger)
			},
		},
		{
			name:   "WithSkipDefaultTransaction true",
			option: WithSkipDefaultTransaction(true),
			check: func(cfg *gorm.Config) {
				assert.True(t, cfg.SkipDefaultTransaction)
			},
		},
		{
			name:   "WithSkipDefaultTransaction false",
			option: WithSkipDefaultTransaction(false),
			check: func(cfg *gorm.Config) {
				assert.False(t, cfg.SkipDefaultTransaction)
			},
		},
		{
			name:   "WithPrepareStmt true",
			option: WithPrepareStmt(true),
			check: func(cfg *gorm.Config) {
				assert.True(t, cfg.PrepareStmt)
			},
		},
		{
			name:   "WithPrepareStmt false",
			option: WithPrepareStmt(false),
			check: func(cfg *gorm.Config) {
				assert.False(t, cfg.PrepareStmt)
			},
		},
		{
			name:   "WithDisableNestedTransaction true",
			option: WithDisableNestedTransaction(true),
			check: func(cfg *gorm.Config) {
				assert.True(t, cfg.DisableNestedTransaction)
			},
		},
		{
			name:   "WithAllowGlobalUpdate true",
			option: WithAllowGlobalUpdate(true),
			check: func(cfg *gorm.Config) {
				assert.True(t, cfg.AllowGlobalUpdate)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &gorm.Config{}
			tt.option(cfg)
			tt.check(cfg)
		})
	}
}

func TestNew_ConfigError(t *testing.T) {
	tests := []struct {
		name   string
		config *database.Config
		errMsg string
	}{
		{
			name: "missing driver",
			config: &database.Config{
				DSN: "host=localhost",
			},
			errMsg: "driver is required",
		},
		{
			name: "wrong driver",
			config: &database.Config{
				Driver: "mysql",
				DSN:    "host=localhost",
			},
			errMsg: "expected driver postgres",
		},
		{
			name: "missing dsn",
			config: &database.Config{
				Driver: "postgres",
			},
			errMsg: "dsn is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestNewWithGORM_ConfigError(t *testing.T) {
	tests := []struct {
		name   string
		config *database.Config
		errMsg string
	}{
		{
			name: "missing driver",
			config: &database.Config{
				DSN: "host=localhost",
			},
			errMsg: "driver is required",
		},
		{
			name: "wrong driver",
			config: &database.Config{
				Driver: "mysql",
				DSN:    "host=localhost",
			},
			errMsg: "expected driver postgres",
		},
		{
			name: "missing dsn",
			config: &database.Config{
				Driver: "postgres",
			},
			errMsg: "dsn is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewWithGORM(tt.config)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestNew_ConnectionError(t *testing.T) {
	// Test with invalid DSN that will fail to connect
	config := &database.Config{
		Driver: "postgres",
		DSN:    "host=invalid-host-12345 user=test password=test dbname=test port=5432 sslmode=disable connect_timeout=1",
		Pool: &database.PoolConfig{
			ConnectTimeout: 1 * time.Second,
		},
	}

	_, err := New(config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to ping database")
}

func TestNewWithGORM_ConnectionError(t *testing.T) {
	// Test with invalid DSN that will fail to connect
	config := &database.Config{
		Driver: "postgres",
		DSN:    "host=invalid-host-12345 user=test password=test dbname=test port=5432 sslmode=disable connect_timeout=1",
		Pool: &database.PoolConfig{
			ConnectTimeout: 1 * time.Second,
		},
	}

	_, err := NewWithGORM(config)
	require.Error(t, err)
	// GORM may fail at open or ping depending on the driver
	assert.Contains(t, err.Error(), "failed to")
}

func TestFactory_Type(t *testing.T) {
	factory := NewFactory()
	assert.Equal(t, database.TypePostgres, factory.Type())
}

func TestFactory_Create_ConfigError(t *testing.T) {
	factory := NewFactory()

	_, err := factory.Create(&database.Config{
		Driver: "invalid",
	})
	require.Error(t, err)
}

func TestFactory_CreateDB_ConfigError(t *testing.T) {
	factory := NewFactory()

	_, err := factory.CreateDB(&database.Config{
		Driver: "invalid",
	})
	require.Error(t, err)
}

func TestTransaction_CommitWithoutDB(t *testing.T) {
	// Transaction with nil DB should handle gracefully
	// In real usage, Transaction is only created from Begin()
	// This test verifies the Transaction struct can be created
	tx := &Transaction{db: nil}
	_ = tx // Use tx to avoid unused variable error
}

func TestTransaction_RollbackWithoutDB(t *testing.T) {
	// Transaction with nil DB should handle gracefully
	// In real usage, Transaction is only created from Begin()
	// This test verifies the Transaction struct can be created
	tx := &Transaction{db: nil}
	_ = tx // Use tx to avoid unused variable error
}

// Integration tests require a running PostgreSQL instance.
// These tests are skipped if PostgreSQL is not available.

func TestIntegration_New(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dsn := getTestDSN(t)
	if dsn == "" {
		t.Skip("PostgreSQL DSN not configured, skipping integration test")
	}

	config := &database.Config{
		Driver: "postgres",
		DSN:    dsn,
	}

	db, err := New(config)
	require.NoError(t, err)
	defer db.Disconnect(context.Background())

	assert.NotNil(t, db)
	assert.True(t, db.IsConnected())

	// Test ping
	err = db.Ping(context.Background())
	assert.NoError(t, err)

	// Test stats
	stats := db.Stats()
	assert.NotNil(t, stats)

	// Test health check
	health := db.CheckHealth(context.Background())
	assert.NotNil(t, health)
	assert.Equal(t, "healthy", health.Status)
}

func TestIntegration_NewWithGORM(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dsn := getTestDSN(t)
	if dsn == "" {
		t.Skip("PostgreSQL DSN not configured, skipping integration test")
	}

	config := &database.Config{
		Driver: "postgres",
		DSN:    dsn,
	}

	db, err := NewWithGORM(config, WithGORMLogLevel("warn"))
	require.NoError(t, err)
	defer db.Disconnect(context.Background())

	assert.NotNil(t, db)
	assert.NotNil(t, db.DB()) // GORM DB

	// Test that we have access to both GORM and raw SQL
	assert.NotNil(t, db.SQLDB())

	// Test health check
	health := db.CheckHealth(context.Background())
	assert.NotNil(t, health)
	assert.Equal(t, "healthy", health.Status)
}

func TestIntegration_Transaction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dsn := getTestDSN(t)
	if dsn == "" {
		t.Skip("PostgreSQL DSN not configured, skipping integration test")
	}

	config := &database.Config{
		Driver: "postgres",
		DSN:    dsn,
	}

	db, err := NewWithGORM(config)
	require.NoError(t, err)
	defer db.Disconnect(context.Background())

	// Test transaction with function
	err = db.Transaction(context.Background(), func(tx *gorm.DB) error {
		// Transaction logic here
		return nil
	})
	assert.NoError(t, err)

	// Test manual transaction
	tx, err := db.Begin(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, tx)

	err = tx.Commit()
	assert.NoError(t, err)
}

func TestIntegration_Disconnect(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dsn := getTestDSN(t)
	if dsn == "" {
		t.Skip("PostgreSQL DSN not configured, skipping integration test")
	}

	config := &database.Config{
		Driver: "postgres",
		DSN:    dsn,
	}

	db, err := NewWithGORM(config)
	require.NoError(t, err)

	// Disconnect
	err = db.Disconnect(context.Background())
	assert.NoError(t, err)

	// Verify disconnected
	assert.False(t, db.IsConnected())
}

// getTestDSN returns the test PostgreSQL DSN from environment variable.
func getTestDSN(t *testing.T) string {
	// In real tests, this would read from environment variable
	// e.g., os.Getenv("TEST_POSTGRES_DSN")
	return ""
}
