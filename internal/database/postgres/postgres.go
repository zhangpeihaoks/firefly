// Package postgres provides PostgreSQL database connection for the Firefly framework.
// It integrates GORM for ORM capabilities with connection pool and health check support.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // Import PostgreSQL driver for database/sql
	"github.com/zhangpeihaoks/firefly/internal/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DriverName is the PostgreSQL driver name.
const DriverName = "postgres"

// Postgres represents a PostgreSQL database connection with GORM support.
type Postgres struct {
	*database.BaseDB
	config *database.Config
	gormDB *gorm.DB
}

// Config represents PostgreSQL-specific configuration options.
type Config struct {
	// Driver is the database driver name.
	Driver string `yaml:"driver" json:"driver" validate:"required"`

	// DSN is the data source name (connection string).
	// Format: host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable TimeZone=Asia/Shanghai
	DSN string `yaml:"dsn" json:"dsn" validate:"required"`

	// Pool is the connection pool configuration.
	Pool *database.PoolConfig `yaml:"pool" json:"pool"`

	// GORM specific options

	// LogLevel is the GORM log level: "silent", "error", "warn", "info".
	// Default: "warn"
	LogLevel string `yaml:"log_level" json:"log_level"`

	// SlowThreshold is the threshold for slow query logging.
	// Default: 200ms
	SlowThreshold time.Duration `yaml:"slow_threshold" json:"slow_threshold"`

	// SkipDefaultTransaction skips default transaction.
	SkipDefaultTransaction bool `yaml:"skip_default_transaction" json:"skip_default_transaction"`

	// PrepareStmt enables prepared statement cache.
	// Default: true
	PrepareStmt bool `yaml:"prepare_stmt" json:"prepare_stmt"`

	// DisableNestedTransaction disables nested transaction.
	DisableNestedTransaction bool `yaml:"disable_nested_transaction" json:"disable_nested_transaction"`

	// AllowGlobalUpdate allows global update without WHERE clause.
	AllowGlobalUpdate bool `yaml:"allow_global_update" json:"allow_global_update"`
}

// Validate validates the PostgreSQL configuration.
func (c *Config) Validate() error {
	if c.Driver == "" {
		return database.NewConfigError("driver is required")
	}
	if c.Driver != string(database.TypePostgres) {
		return database.NewConfigError(fmt.Sprintf("expected driver %s, got %s", database.TypePostgres, c.Driver))
	}
	if c.DSN == "" {
		return database.NewConfigError("dsn is required")
	}
	return nil
}

// ToDatabaseConfig converts PostgreSQL Config to database.Config.
func (c *Config) ToDatabaseConfig() *database.Config {
	return &database.Config{
		Driver: c.Driver,
		DSN:    c.DSN,
		Pool:   c.Pool,
	}
}

// New creates a new PostgreSQL database connection.
// This creates a basic connection without GORM support.
func New(cfg *database.Config) (*Postgres, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if cfg.Driver != string(database.TypePostgres) {
		return nil, database.NewConfigError(fmt.Sprintf("expected driver %s, got %s", database.TypePostgres, cfg.Driver))
	}

	// Open database connection
	db, err := sql.Open(DriverName, cfg.DSN)
	if err != nil {
		return nil, database.NewConnectionError(DriverName, "failed to open database", err)
	}

	// Apply pool configuration
	pool := cfg.GetPoolConfig()
	db.SetMaxOpenConns(pool.MaxOpenConns)
	db.SetMaxIdleConns(pool.MaxIdleConns)
	db.SetConnMaxLifetime(pool.ConnMaxLifetime)
	db.SetConnMaxIdleTime(pool.ConnMaxIdleTime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), pool.ConnectTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, database.NewConnectionError(DriverName, "failed to ping database", err)
	}

	return &Postgres{
		BaseDB: database.NewBaseDB(db, cfg),
		config: cfg,
	}, nil
}

// NewWithGORM creates a new PostgreSQL database connection with GORM support.
func NewWithGORM(cfg *database.Config, opts ...GORMOption) (*Postgres, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if cfg.Driver != string(database.TypePostgres) {
		return nil, database.NewConfigError(fmt.Sprintf("expected driver %s, got %s", database.TypePostgres, cfg.Driver))
	}

	// Create GORM config with defaults
	gormConfig := &gorm.Config{
		SkipDefaultTransaction: false,
		PrepareStmt:            true,
		Logger:                 logger.Default.LogMode(logger.Warn),
	}

	// Apply custom options
	for _, opt := range opts {
		opt(gormConfig)
	}

	// Open GORM connection
	gormDB, err := gorm.Open(postgres.Open(cfg.DSN), gormConfig)
	if err != nil {
		return nil, database.NewConnectionError(DriverName, "failed to open gorm connection", err)
	}

	// Get underlying sql.DB
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, database.NewConnectionError(DriverName, "failed to get underlying sql.DB", err)
	}

	// Apply pool configuration
	pool := cfg.GetPoolConfig()
	sqlDB.SetMaxOpenConns(pool.MaxOpenConns)
	sqlDB.SetMaxIdleConns(pool.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(pool.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(pool.ConnMaxIdleTime)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), pool.ConnectTimeout)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, database.NewConnectionError(DriverName, "failed to ping database", err)
	}

	return &Postgres{
		BaseDB: database.NewBaseDB(sqlDB, cfg),
		config: cfg,
		gormDB: gormDB,
	}, nil
}

// GORMOption is a function that configures GORM.
type GORMOption func(*gorm.Config)

// WithGORMLogLevel sets the GORM log level.
func WithGORMLogLevel(level string) GORMOption {
	return func(cfg *gorm.Config) {
		switch level {
		case "silent":
			cfg.Logger = logger.Default.LogMode(logger.Silent)
		case "error":
			cfg.Logger = logger.Default.LogMode(logger.Error)
		case "warn":
			cfg.Logger = logger.Default.LogMode(logger.Warn)
		case "info":
			cfg.Logger = logger.Default.LogMode(logger.Info)
		default:
			cfg.Logger = logger.Default.LogMode(logger.Warn)
		}
	}
}

// WithGORMLogger sets a custom GORM logger.
func WithGORMLogger(l logger.Interface) GORMOption {
	return func(cfg *gorm.Config) {
		cfg.Logger = l
	}
}

// WithSkipDefaultTransaction sets whether to skip default transaction.
func WithSkipDefaultTransaction(skip bool) GORMOption {
	return func(cfg *gorm.Config) {
		cfg.SkipDefaultTransaction = skip
	}
}

// WithPrepareStmt sets whether to enable prepared statement cache.
func WithPrepareStmt(enable bool) GORMOption {
	return func(cfg *gorm.Config) {
		cfg.PrepareStmt = enable
	}
}

// WithDisableNestedTransaction sets whether to disable nested transaction.
func WithDisableNestedTransaction(disable bool) GORMOption {
	return func(cfg *gorm.Config) {
		cfg.DisableNestedTransaction = disable
	}
}

// WithAllowGlobalUpdate sets whether to allow global update.
func WithAllowGlobalUpdate(allow bool) GORMOption {
	return func(cfg *gorm.Config) {
		cfg.AllowGlobalUpdate = allow
	}
}

// DB returns the GORM DB instance.
// This is the primary method for ORM operations.
func (p *Postgres) DB() *gorm.DB {
	return p.gormDB
}

// SQLDB returns the underlying sql.DB instance.
// Use this for raw SQL operations when needed.
func (p *Postgres) SQLDB() *sql.DB {
	if p.gormDB != nil {
		db, _ := p.gormDB.DB()
		return db
	}
	return p.BaseDB.DB()
}

// CheckHealth returns the health status of the database.
func (p *Postgres) CheckHealth(ctx context.Context) *database.HealthStatus {
	start := time.Now()

	err := p.Ping(ctx)
	latency := time.Since(start)

	status := &database.HealthStatus{
		Latency: latency,
		Stats:   p.Stats(),
	}

	if err != nil {
		status.Status = "unhealthy"
		status.Message = err.Error()
	} else {
		status.Status = "healthy"
		status.Message = "PostgreSQL connection is healthy"
	}

	return status
}

// Disconnect closes the database connection.
func (p *Postgres) Disconnect(ctx context.Context) error {
	if p.gormDB != nil {
		sqlDB, err := p.gormDB.DB()
		if err != nil {
			return database.NewConnectionError(DriverName, "failed to get sql.DB for closing", err)
		}
		if err := sqlDB.Close(); err != nil {
			return database.NewConnectionError(DriverName, "failed to close database", err)
		}
		p.gormDB = nil
		return nil
	}
	return p.BaseDB.Disconnect(ctx)
}

// Transaction executes a function within a transaction.
func (p *Postgres) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	if p.gormDB == nil {
		return database.NewConnectionError(DriverName, "gorm database not initialized", nil)
	}

	db := p.gormDB.WithContext(ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}

// Begin starts a new transaction and returns a Transaction handle.
func (p *Postgres) Begin(ctx context.Context) (*Transaction, error) {
	if p.gormDB == nil {
		return nil, database.NewConnectionError(DriverName, "gorm database not initialized", nil)
	}

	db := p.gormDB.WithContext(ctx)
	tx := db.Begin()
	if tx.Error != nil {
		return nil, database.NewQueryError(DriverName, "BEGIN", "failed to begin transaction", tx.Error)
	}

	return &Transaction{db: tx}, nil
}

// Transaction represents a GORM transaction.
type Transaction struct {
	db *gorm.DB
}

// DB returns the GORM DB instance for the transaction.
func (t *Transaction) DB() *gorm.DB {
	return t.db
}

// Commit commits the transaction.
func (t *Transaction) Commit() error {
	if err := t.db.Commit().Error; err != nil {
		return database.NewQueryError(DriverName, "COMMIT", "commit failed", err)
	}
	return nil
}

// Rollback rolls back the transaction.
func (t *Transaction) Rollback() error {
	if err := t.db.Rollback().Error; err != nil {
		return database.NewQueryError(DriverName, "ROLLBACK", "rollback failed", err)
	}
	return nil
}

// Factory creates PostgreSQL database connectors.
type Factory struct{}

// NewFactory creates a new PostgreSQL factory.
func NewFactory() *Factory {
	return &Factory{}
}

// Create creates a new PostgreSQL connector with the given configuration.
func (f *Factory) Create(cfg *database.Config) (database.Connector, error) {
	return New(cfg)
}

// CreateDB creates a new PostgreSQL database with the given configuration.
func (f *Factory) CreateDB(cfg *database.Config) (database.DB, error) {
	return New(cfg)
}

// CreateWithGORM creates a new PostgreSQL database with GORM support.
func (f *Factory) CreateWithGORM(cfg *database.Config, opts ...GORMOption) (*Postgres, error) {
	return NewWithGORM(cfg, opts...)
}

// Type returns the database type this factory creates.
func (f *Factory) Type() database.DatabaseType {
	return database.TypePostgres
}

// DefaultConfig returns the default PostgreSQL configuration.
func DefaultConfig() *Config {
	return &Config{
		Driver:        string(database.TypePostgres),
		Pool:          database.DefaultPoolConfig(),
		LogLevel:      "warn",
		SlowThreshold: 200 * time.Millisecond,
		PrepareStmt:   true,
	}
}
