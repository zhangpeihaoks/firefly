// Package redis provides Redis database connection for the Firefly framework.
// It integrates go-redis with connection pool and health check support.
package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zhangpeihaoks/firefly/internal/database"
)

// DriverName is the Redis driver name.
const DriverName = "redis"

// Config represents Redis-specific configuration options.
// This matches the RedisConfig structure defined in design.md.
type Config struct {
	// Address is the Redis server address in format "host:port".
	// Can also be a comma-separated list for cluster mode.
	// Example: "localhost:6379" or "node1:6379,node2:6379,node3:6379"
	Address string `yaml:"address" json:"address" validate:"required"`

	// Password is the Redis server password.
	// Default: "" (no password)
	Password string `yaml:"password" json:"password"`

	// DB is the Redis database number to use.
	// Default: 0
	DB int `yaml:"db" json:"db"`

	// PoolSize is the maximum number of connections in the pool.
	// Default: 100
	PoolSize int `yaml:"pool_size" json:"pool_size"`

	// MinIdleConns is the minimum number of idle connections.
	// Default: 10
	MinIdleConns int `yaml:"min_idle_conns" json:"min_idle_conns"`

	// MaxRetries is the maximum number of retries for failed commands.
	// Default: 3
	MaxRetries int `yaml:"max_retries" json:"max_retries"`

	// DialTimeout is the timeout for establishing new connections.
	// Default: 5 seconds
	DialTimeout time.Duration `yaml:"dial_timeout" json:"dial_timeout"`

	// ReadTimeout is the timeout for socket reads.
	// Default: 3 seconds
	ReadTimeout time.Duration `yaml:"read_timeout" json:"read_timeout"`

	// WriteTimeout is the timeout for socket writes.
	// Default: 3 seconds
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`

	// PoolTimeout is the timeout for getting a connection from the pool.
	// Default: 4 seconds
	PoolTimeout time.Duration `yaml:"pool_timeout" json:"pool_timeout"`

	// ConnMaxLifetime is the maximum lifetime of a connection.
	// Default: 30 minutes
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`

	// ConnMaxIdleTime is the maximum idle time of a connection.
	// Default: 10 minutes
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" json:"conn_max_idle_time"`

	// TLS configuration
	// TLSEnabled enables TLS for the connection.
	TLSEnabled bool `yaml:"tls_enabled" json:"tls_enabled"`

	// TLSSkipVerify skips TLS certificate verification.
	TLSSkipVerify bool `yaml:"tls_skip_verify" json:"tls_skip_verify"`

	// Username for Redis ACL (Redis 6.0+).
	Username string `yaml:"username" json:"username"`

	// ClientName sets the client name for tracking.
	ClientName string `yaml:"client_name" json:"client_name"`

	// Protocol specifies the Redis protocol version (2 or 3).
	// Default: 3
	Protocol int `yaml:"protocol" json:"protocol"`
}

// Validate validates the Redis configuration.
func (c *Config) Validate() error {
	if c.Address == "" {
		return database.NewConfigError("address is required")
	}
	return nil
}

// ToRedisOptions converts Config to redis.Options.
func (c *Config) ToRedisOptions() *redis.Options {
	opts := &redis.Options{
		Addr:     c.Address,
		Password: c.Password,
		DB:       c.DB,
		Username: c.Username,
	}

	// Apply pool settings
	if c.PoolSize > 0 {
		opts.PoolSize = c.PoolSize
	} else {
		opts.PoolSize = 100
	}

	if c.MinIdleConns > 0 {
		opts.MinIdleConns = c.MinIdleConns
	} else {
		opts.MinIdleConns = 10
	}

	if c.MaxRetries > 0 {
		opts.MaxRetries = c.MaxRetries
	} else {
		opts.MaxRetries = 3
	}

	// Apply timeout settings
	if c.DialTimeout > 0 {
		opts.DialTimeout = c.DialTimeout
	} else {
		opts.DialTimeout = 5 * time.Second
	}

	if c.ReadTimeout > 0 {
		opts.ReadTimeout = c.ReadTimeout
	} else {
		opts.ReadTimeout = 3 * time.Second
	}

	if c.WriteTimeout > 0 {
		opts.WriteTimeout = c.WriteTimeout
	} else {
		opts.WriteTimeout = 3 * time.Second
	}

	if c.PoolTimeout > 0 {
		opts.PoolTimeout = c.PoolTimeout
	} else {
		opts.PoolTimeout = 4 * time.Second
	}

	if c.ConnMaxLifetime > 0 {
		opts.ConnMaxLifetime = c.ConnMaxLifetime
	} else {
		opts.ConnMaxLifetime = 30 * time.Minute
	}

	if c.ConnMaxIdleTime > 0 {
		opts.ConnMaxIdleTime = c.ConnMaxIdleTime
	} else {
		opts.ConnMaxIdleTime = 10 * time.Minute
	}

	// Apply other settings
	if c.ClientName != "" {
		opts.ClientName = c.ClientName
	}

	if c.Protocol > 0 {
		opts.Protocol = c.Protocol
	} else {
		opts.Protocol = 3
	}

	// Apply TLS settings
	if c.TLSEnabled {
		opts.TLSConfig = &tls.Config{
			InsecureSkipVerify: c.TLSSkipVerify,
		}
	}

	return opts
}

// DefaultConfig returns the default Redis configuration.
func DefaultConfig() *Config {
	return &Config{
		Address:         "localhost:6379",
		DB:              0,
		PoolSize:        100,
		MinIdleConns:    10,
		MaxRetries:      3,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     4 * time.Second,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
		Protocol:        3,
	}
}

// Redis represents a Redis database connection.
type Redis struct {
	client *redis.Client
	config *Config
	mu     sync.RWMutex
}

// New creates a new Redis database connection.
func New(cfg *Config) (*Redis, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	opts := cfg.ToRedisOptions()
	client := redis.NewClient(opts)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), opts.DialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, database.NewConnectionError(DriverName, "failed to ping Redis", err)
	}

	return &Redis{
		client: client,
		config: cfg,
	}, nil
}

// NewFromDSN creates a new Redis database connection from a DSN string.
// DSN format: redis://user:password@host:port/db?pool_size=100&min_idle_conns=10
func NewFromDSN(dsn string) (*Redis, error) {
	opts, err := redis.ParseURL(dsn)
	if err != nil {
		return nil, database.NewConfigError(fmt.Sprintf("invalid Redis DSN: %v", err))
	}

	client := redis.NewClient(opts)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), opts.DialTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, database.NewConnectionError(DriverName, "failed to ping Redis", err)
	}

	return &Redis{
		client: client,
		config: &Config{
			Address:      opts.Addr,
			Password:     opts.Password,
			DB:           opts.DB,
			PoolSize:     opts.PoolSize,
			MinIdleConns: opts.MinIdleConns,
			MaxRetries:   opts.MaxRetries,
		},
	}, nil
}

// Connect establishes a connection to the database.
func (r *Redis) Connect(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return database.NewConnectionError(DriverName, "client not initialized", nil)
	}

	timeout := r.config.DialTimeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return r.client.Ping(ctx).Err()
}

// Disconnect closes the database connection.
func (r *Redis) Disconnect(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.client == nil {
		return nil
	}

	err := r.client.Close()
	r.client = nil
	return err
}

// IsConnected returns true if the connection is established.
func (r *Redis) IsConnected() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.client != nil
}

// Ping checks if the database is reachable.
func (r *Redis) Ping(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return database.NewConnectionError(DriverName, "database not connected", nil)
	}

	timeout := r.config.DialTimeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return r.client.Ping(ctx).Err()
}

// Stats returns connection pool statistics.
func (r *Redis) Stats() *database.PoolStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return nil
	}

	stats := r.client.PoolStats()
	return &database.PoolStats{
		Hits:       int64(stats.Hits),
		Misses:     int64(stats.Misses),
		Timeouts:   int64(stats.Timeouts),
		TotalConns: int64(stats.TotalConns),
		IdleConns:  int64(stats.IdleConns),
		StaleConns: int64(stats.StaleConns),
	}
}

// Client returns the underlying redis.Client instance.
func (r *Redis) Client() *redis.Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client
}

// CheckHealth returns the health status of the database.
func (r *Redis) CheckHealth(ctx context.Context) *database.HealthStatus {
	start := time.Now()

	err := r.Ping(ctx)
	latency := time.Since(start)

	status := &database.HealthStatus{
		Latency: latency,
		Stats:   r.Stats(),
	}

	if err != nil {
		status.Status = "unhealthy"
		status.Message = err.Error()
	} else {
		status.Status = "healthy"
		status.Message = "Redis connection is healthy"
	}

	return status
}

// Set stores a key-value pair with optional expiration.
func (r *Redis) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value by key.
func (r *Redis) Get(ctx context.Context, key string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return "", database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.Get(ctx, key).Result()
}

// Del deletes keys.
func (r *Redis) Del(ctx context.Context, keys ...string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.Del(ctx, keys...).Err()
}

// Exists checks if keys exist.
func (r *Redis) Exists(ctx context.Context, keys ...string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return 0, database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.Exists(ctx, keys...).Result()
}

// Expire sets expiration on a key.
func (r *Redis) Expire(ctx context.Context, key string, expiration time.Duration) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.Expire(ctx, key, expiration).Err()
}

// TTL returns the time to live of a key.
func (r *Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return 0, database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.TTL(ctx, key).Result()
}

// HSet sets a hash field.
func (r *Redis) HSet(ctx context.Context, key string, field string, value any) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.HSet(ctx, key, field, value).Err()
}

// HGet gets a hash field value.
func (r *Redis) HGet(ctx context.Context, key string, field string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return "", database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.HGet(ctx, key, field).Result()
}

// HGetAll gets all hash fields and values.
func (r *Redis) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return nil, database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.HGetAll(ctx, key).Result()
}

// HDel deletes hash fields.
func (r *Redis) HDel(ctx context.Context, key string, fields ...string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.HDel(ctx, key, fields...).Err()
}

// LPush prepends values to a list.
func (r *Redis) LPush(ctx context.Context, key string, values ...any) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.LPush(ctx, key, values...).Err()
}

// RPush appends values to a list.
func (r *Redis) RPush(ctx context.Context, key string, values ...any) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.RPush(ctx, key, values...).Err()
}

// LRange gets elements from a list.
func (r *Redis) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return nil, database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.LRange(ctx, key, start, stop).Result()
}

// LPop removes and returns the first element of a list.
func (r *Redis) LPop(ctx context.Context, key string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return "", database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.LPop(ctx, key).Result()
}

// RPop removes and returns the last element of a list.
func (r *Redis) RPop(ctx context.Context, key string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return "", database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.RPop(ctx, key).Result()
}

// SAdd adds members to a set.
func (r *Redis) SAdd(ctx context.Context, key string, members ...any) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.SAdd(ctx, key, members...).Err()
}

// SMembers gets all members of a set.
func (r *Redis) SMembers(ctx context.Context, key string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return nil, database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.SMembers(ctx, key).Result()
}

// SRem removes members from a set.
func (r *Redis) SRem(ctx context.Context, key string, members ...any) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.SRem(ctx, key, members...).Err()
}

// ZAdd adds a member to a sorted set.
func (r *Redis) ZAdd(ctx context.Context, key string, score float64, member string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.ZAdd(ctx, key, redis.Z{Score: score, Member: member}).Err()
}

// ZRange gets members from a sorted set by index range.
func (r *Redis) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return nil, database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.ZRange(ctx, key, start, stop).Result()
}

// ZRangeByScore gets members from a sorted set by score range.
func (r *Redis) ZRangeByScore(ctx context.Context, key string, min, max string, offset, count int64) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return nil, database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min:    min,
		Max:    max,
		Offset: offset,
		Count:  count,
	}).Result()
}

// ZRem removes members from a sorted set.
func (r *Redis) ZRem(ctx context.Context, key string, members ...any) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.ZRem(ctx, key, members...).Err()
}

// Incr increments a key's value.
func (r *Redis) Incr(ctx context.Context, key string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return 0, database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.Incr(ctx, key).Result()
}

// IncrBy increments a key's value by a specific amount.
func (r *Redis) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return 0, database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.IncrBy(ctx, key, value).Result()
}

// Decr decrements a key's value.
func (r *Redis) Decr(ctx context.Context, key string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return 0, database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.Decr(ctx, key).Result()
}

// DecrBy decrements a key's value by a specific amount.
func (r *Redis) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.client == nil {
		return 0, database.NewConnectionError(DriverName, "database not connected", nil)
	}

	return r.client.DecrBy(ctx, key, value).Result()
}

// Pipeline creates a new pipeline.
func (r *Redis) Pipeline() redis.Pipeliner {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client.Pipeline()
}

// TxPipeline creates a new transaction pipeline.
func (r *Redis) TxPipeline() redis.Pipeliner {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.client.TxPipeline()
}

// Factory creates Redis database connectors.
type Factory struct{}

// NewFactory creates a new Redis factory.
func NewFactory() *Factory {
	return &Factory{}
}

// Create creates a new Redis connector with the given configuration.
func (f *Factory) Create(cfg *Config) (*Redis, error) {
	return New(cfg)
}

// CreateFromDSN creates a new Redis connector from a DSN string.
func (f *Factory) CreateFromDSN(dsn string) (*Redis, error) {
	return NewFromDSN(dsn)
}

// Type returns the database type this factory creates.
func (f *Factory) Type() database.DatabaseType {
	return database.TypeRedis
}
