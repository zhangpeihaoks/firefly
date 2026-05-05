// Package mysql provides MySQL database connection for the Firefly framework.
package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
				Driver: "mysql",
				DSN:    "user:password@tcp(localhost:3306)/dbname",
			},
			wantErr: false,
		},
		{
			name: "missing driver",
			config: &Config{
				DSN: "user:password@tcp(localhost:3306)/dbname",
			},
			wantErr: true,
			errMsg:  "driver is required",
		},
		{
			name: "invalid driver",
			config: &Config{
				Driver: "postgres",
				DSN:    "user:password@tcp(localhost:3306)/dbname",
			},
			wantErr: true,
			errMsg:  "expected driver mysql",
		},
		{
			name: "missing dsn",
			config: &Config{
				Driver: "mysql",
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
	pool := &database.PoolConfig{
		MaxOpenConns:    50,
		MaxIdleConns:    10,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	cfg := &Config{
		Driver: "mysql",
		DSN:    "user:password@tcp(localhost:3306)/dbname",
		Pool:   pool,
	}

	dbCfg := cfg.ToDatabaseConfig()

	assert.Equal(t, "mysql", dbCfg.Driver)
	assert.Equal(t, "user:password@tcp(localhost:3306)/dbname", dbCfg.DSN)
	assert.Equal(t, pool, dbCfg.Pool)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "mysql", cfg.Driver)
	assert.NotNil(t, cfg.Pool)
	assert.Equal(t, "warn", cfg.LogLevel)
	assert.Equal(t, 200*time.Millisecond, cfg.SlowThreshold)
	assert.True(t, cfg.PrepareStmt)
}

func TestNew_ConfigError(t *testing.T) {
	tests := []struct {
		name   string
		config *database.Config
		errMsg string
	}{
		{
			name: "empty driver",
			config: &database.Config{
				DSN: "user:password@tcp(localhost:3306)/dbname",
			},
			errMsg: "driver is required",
		},
		{
			name: "empty dsn",
			config: &database.Config{
				Driver: "mysql",
			},
			errMsg: "dsn is required",
		},
		{
			name: "wrong driver",
			config: &database.Config{
				Driver: "postgres",
				DSN:    "user:password@tcp(localhost:3306)/dbname",
			},
			errMsg: "expected driver mysql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			assert.Error(t, err)
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
			name: "empty driver",
			config: &database.Config{
				DSN: "user:password@tcp(localhost:3306)/dbname",
			},
			errMsg: "driver is required",
		},
		{
			name: "empty dsn",
			config: &database.Config{
				Driver: "mysql",
			},
			errMsg: "dsn is required",
		},
		{
			name: "wrong driver",
			config: &database.Config{
				Driver: "postgres",
				DSN:    "user:password@tcp(localhost:3306)/dbname",
			},
			errMsg: "expected driver mysql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewWithGORM(tt.config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
		})
	}
}

func TestFactory_Type(t *testing.T) {
	f := NewFactory()
	assert.Equal(t, database.TypeMySQL, f.Type())
}

func TestFactory_Create_ConfigError(t *testing.T) {
	f := NewFactory()

	_, err := f.Create(&database.Config{
		Driver: "invalid",
	})
	assert.Error(t, err)
}

func TestFactory_CreateDB_ConfigError(t *testing.T) {
	f := NewFactory()

	_, err := f.CreateDB(&database.Config{
		Driver: "invalid",
	})
	assert.Error(t, err)
}

func TestMySQL_CheckHealth_NotConnected(t *testing.T) {
	// Create a MySQL instance with nil BaseDB
	m := &MySQL{
		BaseDB: database.NewBaseDB(nil, &database.Config{
			Driver: "mysql",
			DSN:    "test",
		}),
		config: &database.Config{
			Driver: "mysql",
			DSN:    "test",
		},
	}

	ctx := context.Background()
	status := m.CheckHealth(ctx)

	assert.Equal(t, "unhealthy", status.Status)
	assert.NotEmpty(t, status.Message)
}

func TestGORMOptions(t *testing.T) {
	t.Run("WithGORMLogLevel", func(t *testing.T) {
		tests := []struct {
			level     string
			wantLevel string
		}{
			{"silent", "silent"},
			{"error", "error"},
			{"warn", "warn"},
			{"info", "info"},
			{"unknown", "warn"}, // default
		}

		for _, tt := range tests {
			t.Run(tt.level, func(t *testing.T) {
				cfg := &gorm.Config{}
				opt := WithGORMLogLevel(tt.level)
				opt(cfg)
				assert.NotNil(t, cfg.Logger)
			})
		}
	})

	t.Run("WithSkipDefaultTransaction", func(t *testing.T) {
		cfg := &gorm.Config{}
		opt := WithSkipDefaultTransaction(true)
		opt(cfg)
		assert.True(t, cfg.SkipDefaultTransaction)
	})

	t.Run("WithPrepareStmt", func(t *testing.T) {
		cfg := &gorm.Config{}
		opt := WithPrepareStmt(true)
		opt(cfg)
		assert.True(t, cfg.PrepareStmt)
	})

	t.Run("WithDisableNestedTransaction", func(t *testing.T) {
		cfg := &gorm.Config{}
		opt := WithDisableNestedTransaction(true)
		opt(cfg)
		assert.True(t, cfg.DisableNestedTransaction)
	})

	t.Run("WithAllowGlobalUpdate", func(t *testing.T) {
		cfg := &gorm.Config{}
		opt := WithAllowGlobalUpdate(true)
		opt(cfg)
		assert.True(t, cfg.AllowGlobalUpdate)
	})
}

func TestMySQL_Transaction_NotConnected(t *testing.T) {
	m := &MySQL{
		BaseDB: database.NewBaseDB(nil, &database.Config{
			Driver: "mysql",
			DSN:    "test",
		}),
		config: &database.Config{
			Driver: "mysql",
			DSN:    "test",
		},
	}

	ctx := context.Background()
	err := m.Transaction(ctx, func(tx *gorm.DB) error {
		return nil
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gorm database not initialized")
}

func TestMySQL_Begin_NotConnected(t *testing.T) {
	m := &MySQL{
		BaseDB: database.NewBaseDB(nil, &database.Config{
			Driver: "mysql",
			DSN:    "test",
		}),
		config: &database.Config{
			Driver: "mysql",
			DSN:    "test",
		},
	}

	ctx := context.Background()
	_, err := m.Begin(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gorm database not initialized")
}
