// Package mongodb provides MongoDB database connection for the Firefly framework.
// It integrates the official MongoDB driver with connection pool, health check, and transaction support.
package mongodb

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/database"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

// MongoDB represents a MongoDB database connection.
type MongoDB struct {
	client   *mongo.Client
	config   *database.Config
	db       *mongo.Database
	opts     *Options
	mu       sync.RWMutex
	connTime time.Time // Connection establishment time
}

// Options represents MongoDB-specific configuration options.
type Options struct {
	// Database is the default database name to use.
	Database string `yaml:"database" json:"database"`

	// ReadConcern level for read operations.
	// Valid values: "local", "available", "majority", "linearizable", "snapshot"
	// Default: "local"
	ReadConcern string `yaml:"read_concern" json:"read_concern"`

	// WriteConcern configuration for write operations.
	WriteConcern *WriteConcernOptions `yaml:"write_concern" json:"write_concern"`

	// ReadPreference for query routing.
	// Valid values: "primary", "primaryPreferred", "secondary", "secondaryPreferred", "nearest"
	// Default: "primary"
	ReadPreference string `yaml:"read_preference" json:"read_preference"`

	// ReplicaSet specifies the replica set name.
	ReplicaSet string `yaml:"replica_set" json:"replica_set"`

	// DirectConnection connects directly to a standalone server.
	DirectConnection bool `yaml:"direct_connection" json:"direct_connection"`

	// ServerSelectionTimeout is the timeout for server selection.
	// Default: 30 seconds
	ServerSelectionTimeout time.Duration `yaml:"server_selection_timeout" json:"server_selection_timeout"`

	// SocketTimeout is the timeout for socket operations.
	SocketTimeout time.Duration `yaml:"socket_timeout" json:"socket_timeout"`

	// HeartbeatInterval is the interval between server heartbeats.
	// Default: 10 seconds
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval" json:"heartbeat_interval"`

	// ServerMonitoringMode specifies the server monitoring mode.
	// Valid values: "auto", "poll", "stream"
	ServerMonitoringMode string `yaml:"server_monitoring_mode" json:"server_monitoring_mode"`

	// RetryWrites enables retryable writes.
	// Default: true
	RetryWrites bool `yaml:"retry_writes" json:"retry_writes"`

	// RetryReads enables retryable reads.
	// Default: true
	RetryReads bool `yaml:"retry_reads" json:"retry_reads"`

	// MaxPoolSize is the maximum number of connections in the pool.
	// Default: 100
	MaxPoolSize uint64 `yaml:"max_pool_size" json:"max_pool_size"`

	// MinPoolSize is the minimum number of connections in the pool.
	// Default: 0
	MinPoolSize uint64 `yaml:"min_pool_size" json:"min_pool_size"`

	// MaxConnIdleTime is the maximum idle time for a connection.
	MaxConnIdleTime time.Duration `yaml:"max_conn_idle_time" json:"max_conn_idle_time"`

	// MaxConnLifeTime is the maximum lifetime of a connection.
	MaxConnLifeTime time.Duration `yaml:"max_conn_life_time" json:"max_conn_life_time"`

	// ConnectTimeout is the timeout for connection establishment.
	// Default: 30 seconds
	ConnectTimeout time.Duration `yaml:"connect_timeout" json:"connect_timeout"`

	// AppName specifies the application name for monitoring.
	AppName string `yaml:"app_name" json:"app_name"`

	// MonitorPoolEvents enables connection pool event monitoring.
	MonitorPoolEvents bool `yaml:"monitor_pool_events" json:"monitor_pool_events"`
}

// WriteConcernOptions represents write concern configuration.
type WriteConcernOptions struct {
	// W is the write concern level.
	// Can be "majority", a number, or a custom tag set.
	W interface{} `yaml:"w" json:"w"`

	// J indicates whether to acknowledge writes after journaling.
	J bool `yaml:"j" json:"j"`

	// WTimeout is the timeout for write concern acknowledgment.
	WTimeout time.Duration `yaml:"wtimeout" json:"wtimeout"`
}

// Option is a function that configures MongoDB options.
type Option func(*Options)

// DefaultOptions returns the default MongoDB options.
func DefaultOptions() *Options {
	return &Options{
		ReadConcern:            "local",
		ReadPreference:         "primary",
		ServerSelectionTimeout: 30 * time.Second,
		HeartbeatInterval:      10 * time.Second,
		RetryWrites:            true,
		RetryReads:             true,
		MaxPoolSize:            100,
		MinPoolSize:            0,
		ConnectTimeout:         30 * time.Second,
	}
}

// WithDatabase sets the default database name.
func WithDatabase(name string) Option {
	return func(o *Options) {
		o.Database = name
	}
}

// WithReadConcern sets the read concern level.
func WithReadConcern(level string) Option {
	return func(o *Options) {
		o.ReadConcern = level
	}
}

// WithWriteConcern sets the write concern configuration.
func WithWriteConcern(wc *WriteConcernOptions) Option {
	return func(o *Options) {
		o.WriteConcern = wc
	}
}

// WithReadPreference sets the read preference.
func WithReadPreference(mode string) Option {
	return func(o *Options) {
		o.ReadPreference = mode
	}
}

// WithReplicaSet sets the replica set name.
func WithReplicaSet(name string) Option {
	return func(o *Options) {
		o.ReplicaSet = name
	}
}

// WithDirectConnection sets direct connection mode.
func WithDirectConnection(enabled bool) Option {
	return func(o *Options) {
		o.DirectConnection = enabled
	}
}

// WithServerSelectionTimeout sets the server selection timeout.
func WithServerSelectionTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.ServerSelectionTimeout = timeout
	}
}

// WithSocketTimeout sets the socket timeout.
func WithSocketTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.SocketTimeout = timeout
	}
}

// WithRetryWrites enables or disables retryable writes.
func WithRetryWrites(enabled bool) Option {
	return func(o *Options) {
		o.RetryWrites = enabled
	}
}

// WithRetryReads enables or disables retryable reads.
func WithRetryReads(enabled bool) Option {
	return func(o *Options) {
		o.RetryReads = enabled
	}
}

// WithAppName sets the application name for monitoring.
func WithAppName(name string) Option {
	return func(o *Options) {
		o.AppName = name
	}
}

// WithMonitorPoolEvents enables or disables pool event monitoring.
func WithMonitorPoolEvents(enabled bool) Option {
	return func(o *Options) {
		o.MonitorPoolEvents = enabled
	}
}

// WithMaxPoolSize sets the maximum pool size.
func WithMaxPoolSize(size uint64) Option {
	return func(o *Options) {
		o.MaxPoolSize = size
	}
}

// WithMinPoolSize sets the minimum pool size.
func WithMinPoolSize(size uint64) Option {
	return func(o *Options) {
		o.MinPoolSize = size
	}
}

// New creates a new MongoDB database connection.
func New(cfg *database.Config, opts ...Option) (*MongoDB, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if cfg.Driver != string(database.TypeMongoDB) {
		return nil, database.NewConfigError(fmt.Sprintf("expected driver %s, got %s", database.TypeMongoDB, cfg.Driver))
	}

	// Apply default options and custom options
	mongoOpts := DefaultOptions()
	for _, opt := range opts {
		opt(mongoOpts)
	}

	// Override with pool config if provided
	pool := cfg.GetPoolConfig()
	if mongoOpts.MaxPoolSize == 0 {
		mongoOpts.MaxPoolSize = uint64(pool.MaxOpenConns)
	}
	if mongoOpts.MinPoolSize == 0 {
		mongoOpts.MinPoolSize = uint64(pool.MaxIdleConns)
	}
	if mongoOpts.MaxConnIdleTime == 0 {
		mongoOpts.MaxConnIdleTime = pool.ConnMaxIdleTime
	}
	if mongoOpts.MaxConnLifeTime == 0 {
		mongoOpts.MaxConnLifeTime = pool.ConnMaxLifetime
	}
	if mongoOpts.ConnectTimeout == 0 {
		mongoOpts.ConnectTimeout = pool.ConnectTimeout
	}

	// Build client options
	clientOpts := options.Client().ApplyURI(cfg.DSN)

	// Apply additional options
	applyMongoDBOptions(clientOpts, mongoOpts, pool)

	// Connect to MongoDB
	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, database.NewConnectionError(string(database.TypeMongoDB), "failed to connect to MongoDB", err)
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), mongoOpts.ConnectTimeout)
	defer cancel()

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		return nil, database.NewConnectionError(string(database.TypeMongoDB), "failed to ping MongoDB", err)
	}

	// Get default database
	var db *mongo.Database
	if mongoOpts.Database != "" {
		db = client.Database(mongoOpts.Database)
	}

	return &MongoDB{
		client:   client,
		config:   cfg,
		db:       db,
		opts:     mongoOpts,
		connTime: time.Now(),
	}, nil
}

// applyMongoDBOptions applies MongoDB-specific options to client options.
func applyMongoDBOptions(clientOpts *options.ClientOptions, opts *Options, pool *database.PoolConfig) {
	// Pool configuration
	clientOpts.SetMaxPoolSize(opts.MaxPoolSize)
	clientOpts.SetMinPoolSize(opts.MinPoolSize)

	if opts.MaxConnIdleTime > 0 {
		clientOpts.SetMaxConnIdleTime(opts.MaxConnIdleTime)
	}
	// Note: MaxConnLifeTime is not directly available in mongo-driver v2

	// Timeouts
	if opts.ConnectTimeout > 0 {
		clientOpts.SetConnectTimeout(opts.ConnectTimeout)
	}
	if opts.ServerSelectionTimeout > 0 {
		clientOpts.SetServerSelectionTimeout(opts.ServerSelectionTimeout)
	}
	// Note: SocketTimeout and HeartbeatInterval may not be available in v2

	// Read concern
	if rc := parseReadConcern(opts.ReadConcern); rc != nil {
		clientOpts.SetReadConcern(rc)
	}

	// Write concern
	if opts.WriteConcern != nil {
		wc := parseWriteConcern(opts.WriteConcern)
		if wc != nil {
			clientOpts.SetWriteConcern(wc)
		}
	}

	// Read preference
	if rp := parseReadPreference(opts.ReadPreference); rp != nil {
		clientOpts.SetReadPreference(rp)
	}

	// Replica set
	if opts.ReplicaSet != "" {
		clientOpts.SetReplicaSet(opts.ReplicaSet)
	}

	// Direct connection
	clientOpts.SetDirect(opts.DirectConnection)

	// Retry settings
	clientOpts.SetRetryWrites(opts.RetryWrites)
	clientOpts.SetRetryReads(opts.RetryReads)

	// App name
	if opts.AppName != "" {
		clientOpts.SetAppName(opts.AppName)
	}

	// Pool event monitoring
	if opts.MonitorPoolEvents {
		poolMonitor := &event.PoolMonitor{
			Event: func(evt *event.PoolEvent) {
				// Pool events can be logged or monitored here
				// This is a hook for observability integration
			},
		}
		clientOpts.SetPoolMonitor(poolMonitor)
	}
}

// parseReadConcern parses a read concern string.
func parseReadConcern(level string) *readconcern.ReadConcern {
	switch level {
	case "local":
		return readconcern.Local()
	case "available":
		return readconcern.Available()
	case "majority":
		return readconcern.Majority()
	case "linearizable":
		return readconcern.Linearizable()
	case "snapshot":
		return readconcern.Snapshot()
	default:
		return readconcern.Local()
	}
}

// parseWriteConcern parses write concern options.
func parseWriteConcern(opts *WriteConcernOptions) *writeconcern.WriteConcern {
	if opts == nil {
		return nil
	}

	wc := &writeconcern.WriteConcern{}

	if opts.W != nil {
		switch v := opts.W.(type) {
		case string:
			if v == "majority" {
				wc = writeconcern.Majority()
			} else {
				wc.W = v
			}
		case int:
			wc.W = v
		}
	}

	wc.Journal = &opts.J

	// Note: WTimeout is handled differently in v2

	return wc
}

// parseReadPreference parses a read preference string.
func parseReadPreference(mode string) *readpref.ReadPref {
	switch mode {
	case "primary":
		return readpref.Primary()
	case "primaryPreferred":
		return readpref.PrimaryPreferred()
	case "secondary":
		return readpref.Secondary()
	case "secondaryPreferred":
		return readpref.SecondaryPreferred()
	case "nearest":
		return readpref.Nearest()
	default:
		return readpref.Primary()
	}
}

// Connect establishes a connection to the database.
func (m *MongoDB) Connect(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.client == nil {
		return database.NewConnectionError(string(database.TypeMongoDB), "client not initialized", nil)
	}

	ctx, cancel := context.WithTimeout(ctx, m.opts.ConnectTimeout)
	defer cancel()

	return m.client.Ping(ctx, readpref.Primary())
}

// Disconnect closes the database connection.
func (m *MongoDB) Disconnect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client == nil {
		return nil
	}

	err := m.client.Disconnect(ctx)
	m.client = nil
	m.db = nil
	return err
}

// IsConnected returns true if the connection is established.
func (m *MongoDB) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.client != nil
}

// Ping checks if the database is reachable.
func (m *MongoDB) Ping(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.client == nil {
		return database.NewConnectionError(string(database.TypeMongoDB), "database not connected", nil)
	}

	ctx, cancel := context.WithTimeout(ctx, m.opts.ConnectTimeout)
	defer cancel()

	return m.client.Ping(ctx, readpref.Primary())
}

// Stats returns connection pool statistics.
func (m *MongoDB) Stats() *database.PoolStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.client == nil {
		return nil
	}

	// MongoDB driver v2 doesn't expose detailed pool stats directly
	// Return a minimal stats structure based on configuration
	return &database.PoolStats{
		MaxOpenConnections: int(m.opts.MaxPoolSize),
		OpenConnections:    0, // Not available in v2 driver
		InUse:              0,
		Idle:               0,
	}
}

// Database returns the MongoDB database instance.
// If name is empty, returns the default database if configured.
func (m *MongoDB) Database(name string) *mongo.Database {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client == nil {
		return nil
	}

	if m.db != nil && (name == "" || m.db.Name() == name) {
		return m.db
	}

	if name != "" {
		m.db = m.client.Database(name)
	}

	return m.db
}

// Collection returns a MongoDB collection from the default database.
// If no default database is configured, use CollectionFromDB instead.
func (m *MongoDB) Collection(name string) *mongo.Collection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.db == nil {
		return nil
	}
	return m.db.Collection(name)
}

// CollectionFromDB returns a MongoDB collection from a specific database.
func (m *MongoDB) CollectionFromDB(dbName, collName string) *mongo.Collection {
	return m.client.Database(dbName).Collection(collName)
}

// Client returns the underlying mongo.Client instance.
func (m *MongoDB) Client() *mongo.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.client
}

// Options returns the MongoDB options.
func (m *MongoDB) Options() *Options {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.opts
}

// ConnectionTime returns the time when the connection was established.
func (m *MongoDB) ConnectionTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connTime
}

// CheckHealth returns the health status of the database.
func (m *MongoDB) CheckHealth(ctx context.Context) *database.HealthStatus {
	start := time.Now()

	err := m.Ping(ctx)
	latency := time.Since(start)

	status := &database.HealthStatus{
		Latency: latency,
		Stats:   m.Stats(),
	}

	if err != nil {
		status.Status = "unhealthy"
		status.Message = err.Error()
	} else {
		status.Status = "healthy"
		status.Message = "MongoDB connection is healthy"
	}

	return status
}

// Transaction executes a function within a transaction.
// Requires MongoDB 4.0+ with replica set or sharded cluster.
func (m *MongoDB) Transaction(ctx context.Context, fn func(sessCtx context.Context) (interface{}, error)) error {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return database.NewConnectionError(string(database.TypeMongoDB), "database not connected", nil)
	}

	session, err := client.StartSession()
	if err != nil {
		return database.NewError("SESSION_ERROR", "failed to start session", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, fn)
	if err != nil {
		return database.NewError("TRANSACTION_ERROR", "transaction failed", err)
	}

	return nil
}

// StartSession starts a new session for causal consistency.
func (m *MongoDB) StartSession(ctx context.Context) (*mongo.Session, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, database.NewConnectionError(string(database.TypeMongoDB), "database not connected", nil)
	}

	return client.StartSession()
}

// ListDatabaseNames returns a list of database names.
func (m *MongoDB) ListDatabaseNames(ctx context.Context, filter interface{}) ([]string, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, database.NewConnectionError(string(database.TypeMongoDB), "database not connected", nil)
	}

	return client.ListDatabaseNames(ctx, filter)
}

// ListCollectionNames returns a list of collection names in the default database.
func (m *MongoDB) ListCollectionNames(ctx context.Context, filter interface{}) ([]string, error) {
	m.mu.RLock()
	db := m.db
	m.mu.RUnlock()

	if db == nil {
		return nil, database.NewConnectionError(string(database.TypeMongoDB), "no default database configured", nil)
	}

	return db.ListCollectionNames(ctx, filter)
}

// RunCommand runs a command on the default database.
func (m *MongoDB) RunCommand(ctx context.Context, command interface{}) (bson.M, error) {
	m.mu.RLock()
	db := m.db
	m.mu.RUnlock()

	if db == nil {
		return nil, database.NewConnectionError(string(database.TypeMongoDB), "no default database configured", nil)
	}

	var result bson.M
	err := db.RunCommand(ctx, command).Decode(&result)
	return result, err
}

// RunCommandOnDatabase runs a command on a specific database.
func (m *MongoDB) RunCommandOnDatabase(ctx context.Context, dbName string, command interface{}) (bson.M, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, database.NewConnectionError(string(database.TypeMongoDB), "database not connected", nil)
	}

	var result bson.M
	err := client.Database(dbName).RunCommand(ctx, command).Decode(&result)
	return result, err
}

// Factory creates MongoDB database connectors.
type Factory struct{}

// NewFactory creates a new MongoDB factory.
func NewFactory() *Factory {
	return &Factory{}
}

// Create creates a new MongoDB connector with the given configuration.
func (f *Factory) Create(cfg *database.Config) (database.Connector, error) {
	return New(cfg)
}

// CreateWithOptions creates a new MongoDB connector with custom options.
func (f *Factory) CreateWithOptions(cfg *database.Config, opts ...Option) (*MongoDB, error) {
	return New(cfg, opts...)
}

// CreateDB is not supported for MongoDB as it uses a different interface.
// Use Create instead.
func (f *Factory) CreateDB(cfg *database.Config) (database.DB, error) {
	return nil, database.NewError("NOT_SUPPORTED", "MongoDB does not implement the SQL DB interface. Use Create() instead.", nil)
}

// Type returns the database type this factory creates.
func (f *Factory) Type() database.DatabaseType {
	return database.TypeMongoDB
}

// DefaultConfig returns the default MongoDB configuration.
func DefaultConfig() *database.Config {
	return &database.Config{
		Driver: string(database.TypeMongoDB),
		Pool:   database.DefaultPoolConfig(),
	}
}

// DefaultOptions returns the default MongoDB options.
// This is an alias for DefaultOptions() for consistency.
func GetDefaultOptions() *Options {
	return DefaultOptions()
}
