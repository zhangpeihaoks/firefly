// Package database provides database connection management for the Firefly framework.
// It supports multiple database types including MySQL, PostgreSQL, MongoDB, and Redis.
package database

import (
	"context"
	"database/sql"
	"time"
)

// DatabaseType represents the type of database.
type DatabaseType string

const (
	// TypeMySQL represents MySQL database.
	TypeMySQL DatabaseType = "mysql"
	// TypePostgres represents PostgreSQL database.
	TypePostgres DatabaseType = "postgres"
	// TypeMongoDB represents MongoDB database.
	TypeMongoDB DatabaseType = "mongodb"
	// TypeRedis represents Redis database.
	TypeRedis DatabaseType = "redis"
)

// Connector is the interface for database connections.
// It provides methods for connecting to and managing database connections.
type Connector interface {
	// Connect establishes a connection to the database.
	Connect(ctx context.Context) error
	// Disconnect closes the database connection.
	Disconnect(ctx context.Context) error
	// IsConnected returns true if the connection is established.
	IsConnected() bool
	// Ping checks if the database is reachable.
	Ping(ctx context.Context) error
	// Stats returns connection pool statistics.
	Stats() *PoolStats
}

// PoolConfig represents the connection pool configuration.
type PoolConfig struct {
	// MaxOpenConns is the maximum number of open connections to the database.
	// Default: 100
	MaxOpenConns int `yaml:"max_open_conns" json:"max_open_conns"`

	// MaxIdleConns is the maximum number of connections in the idle connection pool.
	// Default: 10
	MaxIdleConns int `yaml:"max_idle_conns" json:"max_idle_conns"`

	// ConnMaxLifetime is the maximum amount of time a connection may be reused.
	// Default: 30 minutes
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`

	// ConnMaxIdleTime is the maximum amount of time a connection may be idle.
	// Default: 10 minutes
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" json:"conn_max_idle_time"`

	// ConnectTimeout is the timeout for establishing new connections.
	// Default: 30 seconds
	ConnectTimeout time.Duration `yaml:"connect_timeout" json:"connect_timeout"`
}

// PoolStats represents connection pool statistics.
type PoolStats struct {
	// MaxOpenConnections is the maximum number of open connections.
	MaxOpenConnections int `json:"max_open_connections"`

	// OpenConnections is the number of currently open connections.
	OpenConnections int `json:"open_connections"`

	// InUse is the number of connections currently in use.
	InUse int `json:"in_use"`

	// Idle is the number of idle connections.
	Idle int `json:"idle"`

	// WaitCount is the total number of connections waited for.
	WaitCount int64 `json:"wait_count"`

	// WaitDuration is the total time waited for connections.
	WaitDuration time.Duration `json:"wait_duration"`

	// MaxIdleClosed is the number of connections closed due to max idle.
	MaxIdleClosed int64 `json:"max_idle_closed"`

	// MaxIdleTimeClosed is the number of connections closed due to max idle time.
	MaxIdleTimeClosed int64 `json:"max_idle_time_closed"`

	// MaxLifetimeClosed is the number of connections closed due to max lifetime.
	MaxLifetimeClosed int64 `json:"max_lifetime_closed"`

	// Redis-specific stats

	// Hits is the number of times a connection was found in the pool (Redis).
	Hits int64 `json:"hits,omitempty"`

	// Misses is the number of times a connection was not found in the pool (Redis).
	Misses int64 `json:"misses,omitempty"`

	// Timeouts is the number of times a wait timeout occurred (Redis).
	Timeouts int64 `json:"timeouts,omitempty"`

	// TotalConns is the total number of connections in the pool (Redis).
	TotalConns int64 `json:"total_conns,omitempty"`

	// IdleConns is the number of idle connections in the pool (Redis).
	IdleConns int64 `json:"idle_conns,omitempty"`

	// StaleConns is the number of stale connections removed from the pool (Redis).
	StaleConns int64 `json:"stale_conns,omitempty"`
}

// Config represents the database configuration.
type Config struct {
	// Driver is the database driver name (mysql, postgres, mongodb, redis).
	Driver string `yaml:"driver" json:"driver" validate:"required,oneof=mysql,postgres,mongodb,redis"`

	// DSN is the data source name (connection string).
	DSN string `yaml:"dsn" json:"dsn" validate:"required"`

	// Pool is the connection pool configuration.
	Pool *PoolConfig `yaml:"pool" json:"pool"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Driver == "" {
		return NewConfigError("driver is required")
	}

	validDrivers := map[string]bool{
		string(TypeMySQL):    true,
		string(TypePostgres): true,
		string(TypeMongoDB):  true,
		string(TypeRedis):    true,
	}
	if !validDrivers[c.Driver] {
		return NewConfigError("driver must be one of: mysql, postgres, mongodb, redis")
	}

	if c.DSN == "" {
		return NewConfigError("dsn is required")
	}

	return nil
}

// GetPoolConfig returns the pool configuration with defaults applied.
func (c *Config) GetPoolConfig() *PoolConfig {
	if c.Pool == nil {
		return DefaultPoolConfig()
	}

	pool := *c.Pool
	applyPoolDefaults(&pool)
	return &pool
}

// DefaultPoolConfig returns the default pool configuration.
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    10,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
		ConnectTimeout:  30 * time.Second,
	}
}

// applyPoolDefaults applies default values to unset pool configuration fields.
func applyPoolDefaults(c *PoolConfig) {
	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = 100
	}
	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = 10
	}
	if c.ConnMaxLifetime == 0 {
		c.ConnMaxLifetime = 30 * time.Minute
	}
	if c.ConnMaxIdleTime == 0 {
		c.ConnMaxIdleTime = 10 * time.Minute
	}
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = 30 * time.Second
	}
}

// DB is the interface for SQL database operations.
// It wraps the standard sql.DB interface.
type DB interface {
	Connector
	// Exec executes a query without returning any rows.
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
	// Query executes a query that returns rows.
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	// QueryRow executes a query that returns at most one row.
	QueryRow(ctx context.Context, query string, args ...any) *sql.Row
	// BeginTx starts a transaction.
	BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)
	// Prepare creates a prepared statement for later queries or executions.
	Prepare(ctx context.Context, query string) (*sql.Stmt, error)
}

// Tx is the interface for transaction operations.
type Tx interface {
	// Exec executes a query without returning any rows.
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
	// Query executes a query that returns rows.
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	// QueryRow executes a query that returns at most one row.
	QueryRow(ctx context.Context, query string, args ...any) *sql.Row
	// Commit commits the transaction.
	Commit() error
	// Rollback rolls back the transaction.
	Rollback() error
}

// HealthChecker is the interface for database health checking.
type HealthChecker interface {
	// CheckHealth returns the health status of the database.
	CheckHealth(ctx context.Context) *HealthStatus
}

// HealthStatus represents the health status of a database connection.
type HealthStatus struct {
	// Status is the health status: "healthy" or "unhealthy".
	Status string `json:"status"`

	// Message provides additional information about the health status.
	Message string `json:"message,omitempty"`

	// Latency is the time it took to check the health.
	Latency time.Duration `json:"latency"`

	// Stats contains connection pool statistics.
	Stats *PoolStats `json:"stats,omitempty"`
}

// Factory is the interface for creating database connectors.
type Factory interface {
	// Create creates a new database connector with the given configuration.
	Create(cfg *Config) (Connector, error)
	// CreateDB creates a new SQL database with the given configuration.
	CreateDB(cfg *Config) (DB, error)
	// Type returns the database type this factory creates.
	Type() DatabaseType
}
