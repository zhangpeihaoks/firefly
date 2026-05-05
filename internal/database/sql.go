// Package database provides database connection management for the Firefly framework.
// This file implements common SQL database functionality.
package database

import (
	"context"
	"database/sql"
	"sync"
	"time"
)

// BaseDB provides a base implementation for SQL database connections.
// It can be embedded by specific database implementations.
type BaseDB struct {
	db     *sql.DB
	config *Config
	mu     sync.RWMutex
}

// NewBaseDB creates a new BaseDB instance.
func NewBaseDB(db *sql.DB, config *Config) *BaseDB {
	return &BaseDB{
		db:     db,
		config: config,
	}
}

// Connect establishes a connection to the database.
// For SQL databases, the connection is already established when Open is called.
// This method verifies the connection is working.
func (b *BaseDB) Connect(ctx context.Context) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.db == nil {
		return NewConnectionError(b.config.Driver, "database not initialized", nil)
	}

	ctx, cancel := context.WithTimeout(ctx, b.getConnectTimeout())
	defer cancel()

	return b.db.PingContext(ctx)
}

// Disconnect closes the database connection.
func (b *BaseDB) Disconnect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.db == nil {
		return nil
	}

	err := b.db.Close()
	b.db = nil
	return err
}

// IsConnected returns true if the connection is established.
func (b *BaseDB) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.db != nil
}

// Ping checks if the database is reachable.
func (b *BaseDB) Ping(ctx context.Context) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.db == nil {
		return NewConnectionError(b.config.Driver, "database not connected", nil)
	}

	ctx, cancel := context.WithTimeout(ctx, b.getConnectTimeout())
	defer cancel()

	return b.db.PingContext(ctx)
}

// Stats returns connection pool statistics.
func (b *BaseDB) Stats() *PoolStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.db == nil {
		return nil
	}

	stats := b.db.Stats()
	return &PoolStats{
		MaxOpenConnections: stats.MaxOpenConnections,
		OpenConnections:    stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxIdleTimeClosed:  stats.MaxIdleTimeClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
	}
}

// Exec executes a query without returning any rows.
func (b *BaseDB) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.db == nil {
		return nil, NewConnectionError(b.config.Driver, "database not connected", nil)
	}

	result, err := b.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, NewQueryError(b.config.Driver, query, "exec failed", err)
	}
	return result, nil
}

// Query executes a query that returns rows.
func (b *BaseDB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.db == nil {
		return nil, NewConnectionError(b.config.Driver, "database not connected", nil)
	}

	rows, err := b.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, NewQueryError(b.config.Driver, query, "query failed", err)
	}
	return rows, nil
}

// QueryRow executes a query that returns at most one row.
func (b *BaseDB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.db.QueryRowContext(ctx, query, args...)
}

// BeginTx starts a transaction.
func (b *BaseDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.db == nil {
		return nil, NewConnectionError(b.config.Driver, "database not connected", nil)
	}

	tx, err := b.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, NewQueryError(b.config.Driver, "BEGIN", "failed to begin transaction", err)
	}
	return &baseTx{tx: tx, driver: b.config.Driver}, nil
}

// Prepare creates a prepared statement for later queries or executions.
func (b *BaseDB) Prepare(ctx context.Context, query string) (*sql.Stmt, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.db == nil {
		return nil, NewConnectionError(b.config.Driver, "database not connected", nil)
	}

	stmt, err := b.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, NewQueryError(b.config.Driver, query, "prepare failed", err)
	}
	return stmt, nil
}

// DB returns the underlying sql.DB instance.
// Use with caution as it bypasses the wrapper.
func (b *BaseDB) DB() *sql.DB {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.db
}

// getConnectTimeout returns the connect timeout from config or default.
func (b *BaseDB) getConnectTimeout() time.Duration {
	pool := b.config.GetPoolConfig()
	if pool.ConnectTimeout > 0 {
		return pool.ConnectTimeout
	}
	return 30 * time.Second
}

// baseTx implements the Tx interface.
type baseTx struct {
	tx     *sql.Tx
	driver string
}

// Exec executes a query without returning any rows.
func (t *baseTx) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	result, err := t.tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, NewQueryError(t.driver, query, "exec failed in transaction", err)
	}
	return result, nil
}

// Query executes a query that returns rows.
func (t *baseTx) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, NewQueryError(t.driver, query, "query failed in transaction", err)
	}
	return rows, nil
}

// QueryRow executes a query that returns at most one row.
func (t *baseTx) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}

// Commit commits the transaction.
func (t *baseTx) Commit() error {
	if err := t.tx.Commit(); err != nil {
		return NewQueryError(t.driver, "COMMIT", "commit failed", err)
	}
	return nil
}

// Rollback rolls back the transaction.
func (t *baseTx) Rollback() error {
	if err := t.tx.Rollback(); err != nil {
		return NewQueryError(t.driver, "ROLLBACK", "rollback failed", err)
	}
	return nil
}
