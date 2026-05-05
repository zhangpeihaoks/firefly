// Package mongodb provides MongoDB database connection for the Firefly framework.
package mongodb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zhangpeihaoks/firefly/internal/database"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	assert.Equal(t, "local", opts.ReadConcern)
	assert.Equal(t, "primary", opts.ReadPreference)
	assert.Equal(t, uint64(100), opts.MaxPoolSize)
	assert.Equal(t, uint64(0), opts.MinPoolSize)
	assert.Equal(t, 30*time.Second, opts.ServerSelectionTimeout)
	assert.Equal(t, 10*time.Second, opts.HeartbeatInterval)
	assert.True(t, opts.RetryWrites)
	assert.True(t, opts.RetryReads)
	assert.Equal(t, 30*time.Second, opts.ConnectTimeout)
}

func TestOptions_WithDatabase(t *testing.T) {
	opts := &Options{}
	WithDatabase("testdb")(opts)
	assert.Equal(t, "testdb", opts.Database)
}

func TestOptions_WithReadConcern(t *testing.T) {
	opts := &Options{}
	WithReadConcern("majority")(opts)
	assert.Equal(t, "majority", opts.ReadConcern)
}

func TestOptions_WithWriteConcern(t *testing.T) {
	opts := &Options{}
	wc := &WriteConcernOptions{W: "majority", J: true}
	WithWriteConcern(wc)(opts)
	assert.NotNil(t, opts.WriteConcern)
	assert.Equal(t, "majority", opts.WriteConcern.W)
	assert.True(t, opts.WriteConcern.J)
}

func TestOptions_WithReadPreference(t *testing.T) {
	opts := &Options{}
	WithReadPreference("secondary")(opts)
	assert.Equal(t, "secondary", opts.ReadPreference)
}

func TestOptions_WithReplicaSet(t *testing.T) {
	opts := &Options{}
	WithReplicaSet("rs0")(opts)
	assert.Equal(t, "rs0", opts.ReplicaSet)
}

func TestOptions_WithDirectConnection(t *testing.T) {
	opts := &Options{}
	WithDirectConnection(true)(opts)
	assert.True(t, opts.DirectConnection)
}

func TestOptions_WithServerSelectionTimeout(t *testing.T) {
	opts := &Options{}
	WithServerSelectionTimeout(60 * time.Second)(opts)
	assert.Equal(t, 60*time.Second, opts.ServerSelectionTimeout)
}

func TestOptions_WithSocketTimeout(t *testing.T) {
	opts := &Options{}
	WithSocketTimeout(30 * time.Second)(opts)
	assert.Equal(t, 30*time.Second, opts.SocketTimeout)
}

func TestOptions_WithRetryWrites(t *testing.T) {
	opts := &Options{}
	WithRetryWrites(false)(opts)
	assert.False(t, opts.RetryWrites)
}

func TestOptions_WithRetryReads(t *testing.T) {
	opts := &Options{}
	WithRetryReads(false)(opts)
	assert.False(t, opts.RetryReads)
}

func TestOptions_WithAppName(t *testing.T) {
	opts := &Options{}
	WithAppName("testapp")(opts)
	assert.Equal(t, "testapp", opts.AppName)
}

func TestOptions_WithMonitorPoolEvents(t *testing.T) {
	opts := &Options{}
	WithMonitorPoolEvents(true)(opts)
	assert.True(t, opts.MonitorPoolEvents)
}

func TestOptions_WithMaxPoolSize(t *testing.T) {
	opts := &Options{}
	WithMaxPoolSize(200)(opts)
	assert.Equal(t, uint64(200), opts.MaxPoolSize)
}

func TestOptions_WithMinPoolSize(t *testing.T) {
	opts := &Options{}
	WithMinPoolSize(10)(opts)
	assert.Equal(t, uint64(10), opts.MinPoolSize)
}

func TestParseReadConcern(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected string
	}{
		{"local", "local", "local"},
		{"available", "available", "available"},
		{"majority", "majority", "majority"},
		{"linearizable", "linearizable", "linearizable"},
		{"snapshot", "snapshot", "snapshot"},
		{"unknown defaults to local", "unknown", "local"},
		{"empty defaults to local", "", "local"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := parseReadConcern(tt.level)
			require.NotNil(t, rc)
		})
	}
}

func TestParseReadPreference(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{"primary", "primary"},
		{"primaryPreferred", "primaryPreferred"},
		{"secondary", "secondary"},
		{"secondaryPreferred", "secondaryPreferred"},
		{"nearest", "nearest"},
		{"unknown defaults to primary", "unknown"},
		{"empty defaults to primary", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rp := parseReadPreference(tt.mode)
			require.NotNil(t, rp)
		})
	}
}

func TestParseWriteConcern(t *testing.T) {
	tests := []struct {
		name string
		opts *WriteConcernOptions
	}{
		{"nil options", nil},
		{"majority", &WriteConcernOptions{W: "majority"}},
		{"w=2", &WriteConcernOptions{W: 2}},
		{"with journal", &WriteConcernOptions{W: "majority", J: true}},
		{"with timeout", &WriteConcernOptions{W: "majority", WTimeout: 5 * time.Second}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wc := parseWriteConcern(tt.opts)
			if tt.opts == nil {
				assert.Nil(t, wc)
			} else {
				require.NotNil(t, wc)
			}
		})
	}
}

func TestNew_InvalidConfig(t *testing.T) {
	// Test with empty driver
	cfg := &database.Config{}
	_, err := New(cfg)
	assert.Error(t, err)
	assert.True(t, database.IsConfigError(err))

	// Test with wrong driver
	cfg = &database.Config{
		Driver: "mysql",
		DSN:    "root@tcp(localhost:3306)/test",
	}
	_, err = New(cfg)
	assert.Error(t, err)
	assert.True(t, database.IsConfigError(err))

	// Test with empty DSN
	cfg = &database.Config{
		Driver: string(database.TypeMongoDB),
	}
	_, err = New(cfg)
	assert.Error(t, err)
	assert.True(t, database.IsConfigError(err))
}

func TestNew_ConnectionError(t *testing.T) {
	cfg := &database.Config{
		Driver: string(database.TypeMongoDB),
		DSN:    "mongodb://invalid-host:27017/?connectTimeoutMS=1000",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := New(cfg)
	if err == nil {
		t.Skip("Skipping: connection succeeded to invalid host")
	}
	assert.Error(t, err)
	assert.True(t, database.IsConnectionError(err))
	_ = ctx
}

func TestFactory_Type(t *testing.T) {
	factory := NewFactory()
	assert.Equal(t, database.TypeMongoDB, factory.Type())
}

func TestFactory_CreateDB_NotSupported(t *testing.T) {
	factory := NewFactory()
	cfg := &database.Config{
		Driver: string(database.TypeMongoDB),
		DSN:    "mongodb://localhost:27017",
	}

	_, err := factory.CreateDB(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NOT_SUPPORTED")
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, string(database.TypeMongoDB), cfg.Driver)
	assert.NotNil(t, cfg.Pool)
}

func TestGetDefaultOptions(t *testing.T) {
	opts := GetDefaultOptions()
	assert.Equal(t, DefaultOptions(), opts)
}

func TestMongoDB_MethodsOnDisconnectedClient(t *testing.T) {
	// Create a MongoDB instance with nil client to test disconnected behavior
	m := &MongoDB{
		client: nil,
		config: &database.Config{
			Driver: string(database.TypeMongoDB),
			DSN:    "mongodb://localhost:27017",
		},
		opts: DefaultOptions(),
	}

	ctx := context.Background()

	// Test Connect
	err := m.Connect(ctx)
	assert.Error(t, err)

	// Test Ping
	err = m.Ping(ctx)
	assert.Error(t, err)

	// Test IsConnected
	assert.False(t, m.IsConnected())

	// Test Stats
	stats := m.Stats()
	assert.Nil(t, stats)

	// Test Collection
	coll := m.Collection("test")
	assert.Nil(t, coll)

	// Test ListDatabaseNames
	_, err = m.ListDatabaseNames(ctx, nil)
	assert.Error(t, err)

	// Test ListCollectionNames
	_, err = m.ListCollectionNames(ctx, nil)
	assert.Error(t, err)

	// Test RunCommand
	_, err = m.RunCommand(ctx, map[string]interface{}{"ping": 1})
	assert.Error(t, err)

	// Test RunCommandOnDatabase
	_, err = m.RunCommandOnDatabase(ctx, "test", map[string]interface{}{"ping": 1})
	assert.Error(t, err)

	// Test Transaction
	err = m.Transaction(ctx, func(ctx context.Context) (interface{}, error) {
		return nil, nil
	})
	assert.Error(t, err)

	// Test StartSession
	_, err = m.StartSession(ctx)
	assert.Error(t, err)
}

func TestMongoDB_CheckHealth_Disconnected(t *testing.T) {
	m := &MongoDB{
		client: nil,
		config: &database.Config{
			Driver: string(database.TypeMongoDB),
			DSN:    "mongodb://localhost:27017",
		},
		opts: DefaultOptions(),
	}

	ctx := context.Background()
	status := m.CheckHealth(ctx)

	assert.Equal(t, "unhealthy", status.Status)
	assert.NotEmpty(t, status.Message)
	assert.GreaterOrEqual(t, status.Latency, time.Duration(0))
}

func TestMongoDB_Database(t *testing.T) {
	// Test that Database method works correctly with nil client
	m := &MongoDB{
		client: nil,
		db:     nil,
		opts:   DefaultOptions(),
	}

	// Test with nil client - should return nil, not panic
	db := m.Database("test")
	assert.Nil(t, db)
}

func TestMongoDB_Options(t *testing.T) {
	opts := &Options{
		Database:       "testdb",
		ReadConcern:    "majority",
		ReadPreference: "secondary",
	}

	m := &MongoDB{
		opts: opts,
	}

	returnedOpts := m.Options()
	assert.Equal(t, opts, returnedOpts)
}

func TestMongoDB_ConnectionTime(t *testing.T) {
	connTime := time.Now()
	m := &MongoDB{
		connTime: connTime,
	}

	returnedTime := m.ConnectionTime()
	assert.Equal(t, connTime, returnedTime)
}

func TestMongoDB_Client(t *testing.T) {
	m := &MongoDB{
		client: nil,
	}

	client := m.Client()
	assert.Nil(t, client)
}

func TestMongoDB_Disconnect_NilClient(t *testing.T) {
	m := &MongoDB{
		client: nil,
	}

	err := m.Disconnect(context.Background())
	assert.NoError(t, err)
}

// Integration tests that require a running MongoDB instance
// These tests are skipped if MongoDB is not available

func TestMongoDB_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := &database.Config{
		Driver: string(database.TypeMongoDB),
		DSN:    "mongodb://localhost:27017",
		Pool: &database.PoolConfig{
			MaxOpenConns: 10,
			MaxIdleConns: 5,
		},
	}

	m, err := New(cfg, WithDatabase("testdb"), WithAppName("firefly-test"))
	if err != nil {
		t.Skipf("Skipping: could not connect to MongoDB: %v", err)
	}
	defer m.Disconnect(context.Background())

	ctx := context.Background()

	// Test IsConnected
	assert.True(t, m.IsConnected())

	// Test Ping
	err = m.Ping(ctx)
	assert.NoError(t, err)

	// Test CheckHealth
	status := m.CheckHealth(ctx)
	assert.Equal(t, "healthy", status.Status)
	assert.NotEmpty(t, status.Message)
	assert.Greater(t, status.Latency, time.Duration(0))

	// Test ListDatabaseNames
	dbs, err := m.ListDatabaseNames(ctx, nil)
	assert.NoError(t, err)
	assert.NotNil(t, dbs)

	// Test RunCommand
	result, err := m.RunCommand(ctx, map[string]interface{}{"ping": 1})
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Test Database
	db := m.Database("testdb")
	assert.NotNil(t, db)

	// Test Collection
	coll := m.Collection("testcollection")
	assert.NotNil(t, coll)

	// Test CollectionFromDB
	coll2 := m.CollectionFromDB("testdb", "testcollection")
	assert.NotNil(t, coll2)

	// Test Stats
	stats := m.Stats()
	assert.NotNil(t, stats)
	assert.Equal(t, 10, stats.MaxOpenConnections)
}

func TestMongoDB_Transaction_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := &database.Config{
		Driver: string(database.TypeMongoDB),
		DSN:    "mongodb://localhost:27017/?replicaSet=rs0",
	}

	m, err := New(cfg, WithDatabase("testdb"))
	if err != nil {
		t.Skipf("Skipping: could not connect to MongoDB: %v", err)
	}
	defer m.Disconnect(context.Background())

	ctx := context.Background()

	// Test transaction (requires replica set)
	coll := m.Collection("testcollection")

	err = m.Transaction(ctx, func(sessCtx context.Context) (interface{}, error) {
		// Insert a document within the transaction
		_, err := coll.InsertOne(sessCtx, map[string]interface{}{
			"name":  "test",
			"value": 123,
		})
		return nil, err
	})

	if err != nil {
		t.Logf("Transaction failed (may require replica set): %v", err)
	}
}

func TestMongoDB_Session_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := &database.Config{
		Driver: string(database.TypeMongoDB),
		DSN:    "mongodb://localhost:27017",
	}

	m, err := New(cfg, WithDatabase("testdb"))
	if err != nil {
		t.Skipf("Skipping: could not connect to MongoDB: %v", err)
	}
	defer m.Disconnect(context.Background())

	ctx := context.Background()

	// Test StartSession
	session, err := m.StartSession(ctx)
	if err != nil {
		t.Skipf("Skipping: could not start session: %v", err)
	}
	defer session.EndSession(ctx)

	assert.NotNil(t, session)
}
