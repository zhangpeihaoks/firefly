// Package redis provides Redis database connection tests for the Firefly framework.
package redis

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zhangpeihaoks/firefly/internal/database"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			config:  &Config{Address: "localhost:6379"},
			wantErr: false,
		},
		{
			name:    "empty address",
			config:  &Config{Address: ""},
			wantErr: true,
			errMsg:  "address is required",
		},
		{
			name:    "full config",
			config:  &Config{Address: "localhost:6379", Password: "secret", DB: 1, PoolSize: 50, MinIdleConns: 5, MaxRetries: 5},
			wantErr: false,
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

func TestConfig_ToRedisOptions(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		check  func(t *testing.T, opts *redis.Options)
	}{
		{
			name:   "default values",
			config: &Config{Address: "localhost:6379"},
			check: func(t *testing.T, opts *redis.Options) {
				assert.Equal(t, "localhost:6379", opts.Addr)
				assert.Equal(t, 100, opts.PoolSize)
				assert.Equal(t, 10, opts.MinIdleConns)
				assert.Equal(t, 3, opts.MaxRetries)
				assert.Equal(t, 5*time.Second, opts.DialTimeout)
				assert.Equal(t, 3*time.Second, opts.ReadTimeout)
				assert.Equal(t, 3*time.Second, opts.WriteTimeout)
				assert.Equal(t, 4*time.Second, opts.PoolTimeout)
				assert.Equal(t, 30*time.Minute, opts.ConnMaxLifetime)
				assert.Equal(t, 10*time.Minute, opts.ConnMaxIdleTime)
				assert.Equal(t, 3, opts.Protocol)
			},
		},
		{
			name: "custom values",
			config: &Config{
				Address:         "redis.example.com:6380",
				Password:        "mypassword",
				DB:              2,
				PoolSize:        50,
				MinIdleConns:    5,
				MaxRetries:      5,
				DialTimeout:     10 * time.Second,
				ReadTimeout:     5 * time.Second,
				WriteTimeout:    5 * time.Second,
				PoolTimeout:     8 * time.Second,
				ConnMaxLifetime: 1 * time.Hour,
				ConnMaxIdleTime: 30 * time.Minute,
				ClientName:      "my-app",
				Protocol:        2,
			},
			check: func(t *testing.T, opts *redis.Options) {
				assert.Equal(t, "redis.example.com:6380", opts.Addr)
				assert.Equal(t, "mypassword", opts.Password)
				assert.Equal(t, 2, opts.DB)
				assert.Equal(t, 50, opts.PoolSize)
				assert.Equal(t, 5, opts.MinIdleConns)
				assert.Equal(t, 5, opts.MaxRetries)
				assert.Equal(t, 10*time.Second, opts.DialTimeout)
				assert.Equal(t, 5*time.Second, opts.ReadTimeout)
				assert.Equal(t, 5*time.Second, opts.WriteTimeout)
				assert.Equal(t, 8*time.Second, opts.PoolTimeout)
				assert.Equal(t, 1*time.Hour, opts.ConnMaxLifetime)
				assert.Equal(t, 30*time.Minute, opts.ConnMaxIdleTime)
				assert.Equal(t, "my-app", opts.ClientName)
				assert.Equal(t, 2, opts.Protocol)
			},
		},
		{
			name: "tls enabled",
			config: &Config{
				Address:       "redis.example.com:6379",
				TLSEnabled:    true,
				TLSSkipVerify: true,
			},
			check: func(t *testing.T, opts *redis.Options) {
				assert.NotNil(t, opts.TLSConfig)
				assert.True(t, opts.TLSConfig.InsecureSkipVerify)
			},
		},
		{
			name: "username for ACL",
			config: &Config{
				Address:  "localhost:6379",
				Username: "appuser",
				Password: "secret",
			},
			check: func(t *testing.T, opts *redis.Options) {
				assert.Equal(t, "appuser", opts.Username)
				assert.Equal(t, "secret", opts.Password)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.config.ToRedisOptions()
			tt.check(t, opts)
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "localhost:6379", cfg.Address)
	assert.Equal(t, 0, cfg.DB)
	assert.Equal(t, 100, cfg.PoolSize)
	assert.Equal(t, 10, cfg.MinIdleConns)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 5*time.Second, cfg.DialTimeout)
	assert.Equal(t, 3*time.Second, cfg.ReadTimeout)
	assert.Equal(t, 3*time.Second, cfg.WriteTimeout)
	assert.Equal(t, 4*time.Second, cfg.PoolTimeout)
	assert.Equal(t, 30*time.Minute, cfg.ConnMaxLifetime)
	assert.Equal(t, 10*time.Minute, cfg.ConnMaxIdleTime)
	assert.Equal(t, 3, cfg.Protocol)
}

func TestRedis_New_Error(t *testing.T) {
	// Test with invalid address - should fail to connect
	cfg := &Config{
		Address:     "invalid-host:6379",
		DialTimeout: 100 * time.Millisecond,
	}

	_, err := New(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to ping Redis")
}

func TestRedis_NewFromDSN_Error(t *testing.T) {
	// Test with invalid DSN
	_, err := NewFromDSN("invalid-dsn")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid Redis DSN")
}

func TestRedis_CheckHealth(t *testing.T) {
	// Create a Redis instance without actual connection for testing health check structure
	r := &Redis{
		client: nil,
		config: DefaultConfig(),
	}

	ctx := context.Background()
	status := r.CheckHealth(ctx)

	assert.Equal(t, "unhealthy", status.Status)
	assert.Contains(t, status.Message, "database not connected")
}

func TestRedis_IsConnected(t *testing.T) {
	r := &Redis{
		client: nil,
		config: DefaultConfig(),
	}

	assert.False(t, r.IsConnected())
}

func TestRedis_Ping_NotConnected(t *testing.T) {
	r := &Redis{
		client: nil,
		config: DefaultConfig(),
	}

	err := r.Ping(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")
}

func TestRedis_Connect_NotInitialized(t *testing.T) {
	r := &Redis{
		client: nil,
		config: DefaultConfig(),
	}

	err := r.Connect(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client not initialized")
}

func TestRedis_Stats_Nil(t *testing.T) {
	r := &Redis{
		client: nil,
		config: DefaultConfig(),
	}

	stats := r.Stats()
	assert.Nil(t, stats)
}

func TestRedis_Client(t *testing.T) {
	r := &Redis{
		client: nil,
		config: DefaultConfig(),
	}

	client := r.Client()
	assert.Nil(t, client)
}

func TestRedis_Operations_NotConnected(t *testing.T) {
	r := &Redis{
		client: nil,
		config: DefaultConfig(),
	}
	ctx := context.Background()

	// Test Set
	err := r.Set(ctx, "key", "value", time.Minute)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	// Test Get
	_, err = r.Get(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	// Test Del
	err = r.Del(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	// Test Exists
	_, err = r.Exists(ctx, "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	// Test Incr
	_, err = r.Incr(ctx, "counter")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	// Test HSet
	err = r.HSet(ctx, "hash", "field", "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	// Test HGet
	_, err = r.HGet(ctx, "hash", "field")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	// Test LPush
	err = r.LPush(ctx, "list", "value")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	// Test SAdd
	err = r.SAdd(ctx, "set", "member")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")

	// Test ZAdd
	err = r.ZAdd(ctx, "zset", 1.0, "member")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database not connected")
}

func TestFactory(t *testing.T) {
	f := NewFactory()

	assert.Equal(t, database.TypeRedis, f.Type())

	// Test Create with invalid config (should fail to connect)
	cfg := &Config{
		Address:     "invalid-host:6379",
		DialTimeout: 100 * time.Millisecond,
	}

	_, err := f.Create(cfg)
	assert.Error(t, err)
}

func TestFactory_CreateFromDSN(t *testing.T) {
	f := NewFactory()

	// Test with invalid DSN
	_, err := f.CreateFromDSN("invalid-dsn")
	assert.Error(t, err)
}

// Integration tests require a running Redis server.
// Run with: go test -tags=integration ./internal/database/redis/...

func TestRedis_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := DefaultConfig()
	cfg.DialTimeout = 2 * time.Second

	r, err := New(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer r.Disconnect(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("connect and ping", func(t *testing.T) {
		err := r.Connect(ctx)
		assert.NoError(t, err)
		assert.True(t, r.IsConnected())

		err = r.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("set and get", func(t *testing.T) {
		err := r.Set(ctx, "test:key", "value", time.Minute)
		require.NoError(t, err)

		val, err := r.Get(ctx, "test:key")
		require.NoError(t, err)
		assert.Equal(t, "value", val)

		err = r.Del(ctx, "test:key")
		require.NoError(t, err)
	})

	t.Run("incr and decr", func(t *testing.T) {
		r.Del(ctx, "test:counter")

		val, err := r.Incr(ctx, "test:counter")
		require.NoError(t, err)
		assert.Equal(t, int64(1), val)

		val, err = r.IncrBy(ctx, "test:counter", 10)
		require.NoError(t, err)
		assert.Equal(t, int64(11), val)

		val, err = r.Decr(ctx, "test:counter")
		require.NoError(t, err)
		assert.Equal(t, int64(10), val)

		r.Del(ctx, "test:counter")
	})

	t.Run("hash operations", func(t *testing.T) {
		err := r.HSet(ctx, "test:hash", "field1", "value1")
		require.NoError(t, err)

		val, err := r.HGet(ctx, "test:hash", "field1")
		require.NoError(t, err)
		assert.Equal(t, "value1", val)

		err = r.HDel(ctx, "test:hash", "field1")
		require.NoError(t, err)

		r.Del(ctx, "test:hash")
	})

	t.Run("list operations", func(t *testing.T) {
		err := r.RPush(ctx, "test:list", "item1", "item2", "item3")
		require.NoError(t, err)

		items, err := r.LRange(ctx, "test:list", 0, -1)
		require.NoError(t, err)
		assert.Len(t, items, 3)

		item, err := r.LPop(ctx, "test:list")
		require.NoError(t, err)
		assert.Equal(t, "item1", item)

		r.Del(ctx, "test:list")
	})

	t.Run("set operations", func(t *testing.T) {
		err := r.SAdd(ctx, "test:set", "member1", "member2")
		require.NoError(t, err)

		members, err := r.SMembers(ctx, "test:set")
		require.NoError(t, err)
		assert.Len(t, members, 2)

		err = r.SRem(ctx, "test:set", "member1")
		require.NoError(t, err)

		r.Del(ctx, "test:set")
	})

	t.Run("sorted set operations", func(t *testing.T) {
		err := r.ZAdd(ctx, "test:zset", 1.0, "one")
		require.NoError(t, err)

		err = r.ZAdd(ctx, "test:zset", 2.0, "two")
		require.NoError(t, err)

		members, err := r.ZRange(ctx, "test:zset", 0, -1)
		require.NoError(t, err)
		assert.Len(t, members, 2)

		r.Del(ctx, "test:zset")
	})

	t.Run("exists and ttl", func(t *testing.T) {
		err := r.Set(ctx, "test:ttl", "value", time.Minute)
		require.NoError(t, err)

		count, err := r.Exists(ctx, "test:ttl")
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		ttl, err := r.TTL(ctx, "test:ttl")
		require.NoError(t, err)
		assert.True(t, ttl > 0 && ttl <= time.Minute)

		r.Del(ctx, "test:ttl")
	})

	t.Run("health check", func(t *testing.T) {
		status := r.CheckHealth(ctx)
		assert.Equal(t, "healthy", status.Status)
		assert.Equal(t, "Redis connection is healthy", status.Message)
		assert.GreaterOrEqual(t, status.Latency.Microseconds(), int64(0))
		assert.NotNil(t, status.Stats)
	})

	t.Run("stats", func(t *testing.T) {
		stats := r.Stats()
		assert.NotNil(t, stats)
		assert.GreaterOrEqual(t, stats.TotalConns, int64(0))
	})

	t.Run("pipeline", func(t *testing.T) {
		pipe := r.Pipeline()
		assert.NotNil(t, pipe)
	})

	t.Run("disconnect", func(t *testing.T) {
		err := r.Disconnect(ctx)
		assert.NoError(t, err)
		assert.False(t, r.IsConnected())
	})
}
