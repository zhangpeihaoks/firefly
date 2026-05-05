// Package config provides configuration management for the Firefly framework.
package config

import (
	"testing"
	"time"
)

func TestDefaultBootstrap(t *testing.T) {
	b := DefaultBootstrap()

	if b.Name != DefaultName {
		t.Errorf("Name = %v, want %v", b.Name, DefaultName)
	}
	if b.Version != DefaultVersion {
		t.Errorf("Version = %v, want %v", b.Version, DefaultVersion)
	}
	if b.HTTP == nil {
		t.Error("HTTP config is nil")
	}
	if b.GRPC == nil {
		t.Error("GRPC config is nil")
	}
	if b.Log == nil {
		t.Error("Log config is nil")
	}
	if b.Metrics == nil {
		t.Error("Metrics config is nil")
	}
}

func TestDefaultHTTPConfig(t *testing.T) {
	c := DefaultHTTPConfig()

	if c.Network != DefaultHTTPNetwork {
		t.Errorf("Network = %v, want %v", c.Network, DefaultHTTPNetwork)
	}
	if c.Port != DefaultHTTPPort {
		t.Errorf("Port = %v, want %v", c.Port, DefaultHTTPPort)
	}
	if c.Timeout != DefaultHTTPTimeout {
		t.Errorf("Timeout = %v, want %v", c.Timeout, DefaultHTTPTimeout)
	}
	if c.ReadTimeout != DefaultHTTPReadTimeout {
		t.Errorf("ReadTimeout = %v, want %v", c.ReadTimeout, DefaultHTTPReadTimeout)
	}
	if c.WriteTimeout != DefaultHTTPWriteTimeout {
		t.Errorf("WriteTimeout = %v, want %v", c.WriteTimeout, DefaultHTTPWriteTimeout)
	}
	if c.IdleTimeout != DefaultHTTPIdleTimeout {
		t.Errorf("IdleTimeout = %v, want %v", c.IdleTimeout, DefaultHTTPIdleTimeout)
	}
	// Test GetAddress method
	if c.GetAddress() != ":8080" {
		t.Errorf("GetAddress() = %v, want :8080", c.GetAddress())
	}
}

func TestDefaultGRPCConfig(t *testing.T) {
	c := DefaultGRPCConfig()

	if c.Network != DefaultGRPCNetwork {
		t.Errorf("Network = %v, want %v", c.Network, DefaultGRPCNetwork)
	}
	if c.Port != DefaultGRPCPort {
		t.Errorf("Port = %v, want %v", c.Port, DefaultGRPCPort)
	}
	if c.Timeout != DefaultGRPCTimeout {
		t.Errorf("Timeout = %v, want %v", c.Timeout, DefaultGRPCTimeout)
	}
	if c.MaxRecvMsgSize != DefaultGRPCMaxRecvMsgSize {
		t.Errorf("MaxRecvMsgSize = %v, want %v", c.MaxRecvMsgSize, DefaultGRPCMaxRecvMsgSize)
	}
	if c.MaxSendMsgSize != DefaultGRPCMaxSendMsgSize {
		t.Errorf("MaxSendMsgSize = %v, want %v", c.MaxSendMsgSize, DefaultGRPCMaxSendMsgSize)
	}
	// Test GetAddress method
	if c.GetAddress() != ":9090" {
		t.Errorf("GetAddress() = %v, want :9090", c.GetAddress())
	}
}

func TestDefaultLogConfig(t *testing.T) {
	c := DefaultLogConfig()

	if c.FileName != DefaultLogFileName {
		t.Errorf("FileName = %v, want %v", c.FileName, DefaultLogFileName)
	}
	if c.MaxSize != DefaultLogMaxSize {
		t.Errorf("MaxSize = %v, want %v", c.MaxSize, DefaultLogMaxSize)
	}
	if c.MaxBackups != DefaultLogMaxBackups {
		t.Errorf("MaxBackups = %v, want %v", c.MaxBackups, DefaultLogMaxBackups)
	}
	if c.MaxAge != DefaultLogMaxAge {
		t.Errorf("MaxAge = %v, want %v", c.MaxAge, DefaultLogMaxAge)
	}
	if c.Level != DefaultLogLevel {
		t.Errorf("Level = %v, want %v", c.Level, DefaultLogLevel)
	}
	if c.JSONFormat != DefaultLogJSONFormat {
		t.Errorf("JSONFormat = %v, want %v", c.JSONFormat, DefaultLogJSONFormat)
	}
	if c.Location != DefaultLogLocation {
		t.Errorf("Location = %v, want %v", c.Location, DefaultLogLocation)
	}
}

func TestDefaultDatabaseConfig(t *testing.T) {
	c := DefaultDatabaseConfig()

	if c.Driver != DefaultDatabaseDriver {
		t.Errorf("Driver = %v, want %v", c.Driver, DefaultDatabaseDriver)
	}
	if c.MaxOpenConns != DefaultDatabaseMaxOpenConns {
		t.Errorf("MaxOpenConns = %v, want %v", c.MaxOpenConns, DefaultDatabaseMaxOpenConns)
	}
	if c.MaxIdleConns != DefaultDatabaseMaxIdleConns {
		t.Errorf("MaxIdleConns = %v, want %v", c.MaxIdleConns, DefaultDatabaseMaxIdleConns)
	}
	if c.ConnMaxLifetime != DefaultDatabaseConnMaxLifetime {
		t.Errorf("ConnMaxLifetime = %v, want %v", c.ConnMaxLifetime, DefaultDatabaseConnMaxLifetime)
	}
	if c.ConnMaxIdleTime != DefaultDatabaseConnMaxIdleTime {
		t.Errorf("ConnMaxIdleTime = %v, want %v", c.ConnMaxIdleTime, DefaultDatabaseConnMaxIdleTime)
	}
}

func TestDefaultRedisConfig(t *testing.T) {
	c := DefaultRedisConfig()

	if c.DB != DefaultRedisDB {
		t.Errorf("DB = %v, want %v", c.DB, DefaultRedisDB)
	}
	if c.PoolSize != DefaultRedisPoolSize {
		t.Errorf("PoolSize = %v, want %v", c.PoolSize, DefaultRedisPoolSize)
	}
	if c.MinIdleConns != DefaultRedisMinIdleConns {
		t.Errorf("MinIdleConns = %v, want %v", c.MinIdleConns, DefaultRedisMinIdleConns)
	}
	if c.MaxRetries != DefaultRedisMaxRetries {
		t.Errorf("MaxRetries = %v, want %v", c.MaxRetries, DefaultRedisMaxRetries)
	}
}

func TestDefaultTracingConfig(t *testing.T) {
	c := DefaultTracingConfig()

	if c.Enabled != DefaultTracingEnabled {
		t.Errorf("Enabled = %v, want %v", c.Enabled, DefaultTracingEnabled)
	}
	if c.SamplerRatio != DefaultTracingSamplerRatio {
		t.Errorf("SamplerRatio = %v, want %v", c.SamplerRatio, DefaultTracingSamplerRatio)
	}
}

func TestDefaultMetricsConfig(t *testing.T) {
	c := DefaultMetricsConfig()

	if c.Enabled != DefaultMetricsEnabled {
		t.Errorf("Enabled = %v, want %v", c.Enabled, DefaultMetricsEnabled)
	}
	if c.Path != DefaultMetricsPath {
		t.Errorf("Path = %v, want %v", c.Path, DefaultMetricsPath)
	}
}

func TestApplyDefaults(t *testing.T) {
	t.Run("apply defaults to empty bootstrap", func(t *testing.T) {
		b := &Bootstrap{}
		ApplyDefaults(b)

		if b.Name != DefaultName {
			t.Errorf("Name = %v, want %v", b.Name, DefaultName)
		}
		if b.Version != DefaultVersion {
			t.Errorf("Version = %v, want %v", b.Version, DefaultVersion)
		}
		if b.HTTP == nil {
			t.Error("HTTP config is nil after ApplyDefaults")
		}
		if b.GRPC == nil {
			t.Error("GRPC config is nil after ApplyDefaults")
		}
		if b.Log == nil {
			t.Error("Log config is nil after ApplyDefaults")
		}
	})

	t.Run("preserve existing values", func(t *testing.T) {
		b := &Bootstrap{
			Name: "custom-app",
			HTTP: &HTTPConfig{
				Address: ":9090",
			},
		}
		ApplyDefaults(b)

		if b.Name != "custom-app" {
			t.Errorf("Name = %v, want custom-app", b.Name)
		}
		if b.HTTP.Address != ":9090" {
			t.Errorf("HTTP.Address = %v, want :9090", b.HTTP.Address)
		}
		// Defaults should be applied to missing fields
		if b.HTTP.Network != DefaultHTTPNetwork {
			t.Errorf("HTTP.Network = %v, want %v", b.HTTP.Network, DefaultHTTPNetwork)
		}
		if b.HTTP.Timeout != DefaultHTTPTimeout {
			t.Errorf("HTTP.Timeout = %v, want %v", b.HTTP.Timeout, DefaultHTTPTimeout)
		}
	})

	t.Run("apply defaults to partial config", func(t *testing.T) {
		b := &Bootstrap{
			Name: "test-app",
			HTTP: &HTTPConfig{
				Address: ":8080",
			},
			Log: &LogConfig{
				Level: "debug",
			},
		}
		ApplyDefaults(b)

		// HTTP defaults should be applied
		if b.HTTP.Network != DefaultHTTPNetwork {
			t.Errorf("HTTP.Network = %v, want %v", b.HTTP.Network, DefaultHTTPNetwork)
		}
		if b.HTTP.Timeout != DefaultHTTPTimeout {
			t.Errorf("HTTP.Timeout = %v, want %v", b.HTTP.Timeout, DefaultHTTPTimeout)
		}

		// Log defaults should be applied
		if b.Log.FileName != DefaultLogFileName {
			t.Errorf("Log.FileName = %v, want %v", b.Log.FileName, DefaultLogFileName)
		}
		if b.Log.Level != "debug" {
			t.Errorf("Log.Level = %v, want debug", b.Log.Level)
		}
	})
}

func TestApplyDefaults_Database(t *testing.T) {
	b := &Bootstrap{
		Database: &DatabaseConfig{
			Driver: "postgres",
			DSN:    "postgres://localhost:5432/db",
		},
	}
	ApplyDefaults(b)

	if b.Database.Driver != "postgres" {
		t.Errorf("Driver = %v, want postgres", b.Database.Driver)
	}
	if b.Database.MaxOpenConns != DefaultDatabaseMaxOpenConns {
		t.Errorf("MaxOpenConns = %v, want %v", b.Database.MaxOpenConns, DefaultDatabaseMaxOpenConns)
	}
}

func TestApplyDefaults_Redis(t *testing.T) {
	b := &Bootstrap{
		Redis: &RedisConfig{
			Address: "localhost:6379",
		},
	}
	ApplyDefaults(b)

	if b.Redis.Address != "localhost:6379" {
		t.Errorf("Address = %v, want localhost:6379", b.Redis.Address)
	}
	if b.Redis.PoolSize != DefaultRedisPoolSize {
		t.Errorf("PoolSize = %v, want %v", b.Redis.PoolSize, DefaultRedisPoolSize)
	}
}

func TestApplyDefaults_Tracing(t *testing.T) {
	b := &Bootstrap{
		Tracing: &TracingConfig{
			Enabled:  true,
			Endpoint: "http://localhost:14268",
		},
	}
	ApplyDefaults(b)

	if b.Tracing.SamplerRatio != DefaultTracingSamplerRatio {
		t.Errorf("SamplerRatio = %v, want %v", b.Tracing.SamplerRatio, DefaultTracingSamplerRatio)
	}
}

func TestApplyDefaults_Metrics(t *testing.T) {
	b := &Bootstrap{
		Metrics: &MetricsConfig{
			Enabled: false,
		},
	}
	ApplyDefaults(b)

	if b.Metrics.Path != DefaultMetricsPath {
		t.Errorf("Path = %v, want %v", b.Metrics.Path, DefaultMetricsPath)
	}
}

func TestApplyDefaults_Serializer(t *testing.T) {
	b := &Bootstrap{
		Serializer: &SerializerConfig{
			Mode: "protobuf",
		},
	}
	ApplyDefaults(b)

	if b.Serializer.Mode != "protobuf" {
		t.Errorf("Mode = %v, want protobuf", b.Serializer.Mode)
	}
}

func TestApplyDefaults_TLS(t *testing.T) {
	t.Run("TLS config nil", func(t *testing.T) {
		b := &Bootstrap{
			HTTP: &HTTPConfig{
				Address: ":8080",
			},
		}
		ApplyDefaults(b)

		if b.HTTP.TLS != nil {
			t.Error("TLS should remain nil")
		}
	})

	t.Run("TLS config with defaults", func(t *testing.T) {
		b := &Bootstrap{
			HTTP: &HTTPConfig{
				Address: ":8080",
				TLS:     &TLSConfig{},
			},
		}
		ApplyDefaults(b)

		if b.HTTP.TLS.Enabled != false {
			t.Error("TLS.Enabled should be false by default")
		}
	})
}

func TestDefaultValues(t *testing.T) {
	// Verify default values are reasonable
	if DefaultHTTPTimeout <= 0 {
		t.Error("DefaultHTTPTimeout should be positive")
	}
	if DefaultGRPCMaxRecvMsgSize <= 0 {
		t.Error("DefaultGRPCMaxRecvMsgSize should be positive")
	}
	if DefaultLogMaxSize <= 0 {
		t.Error("DefaultLogMaxSize should be positive")
	}
	if DefaultDatabaseMaxOpenConns <= 0 {
		t.Error("DefaultDatabaseMaxOpenConns should be positive")
	}
	if DefaultRedisPoolSize <= 0 {
		t.Error("DefaultRedisPoolSize should be positive")
	}
}

func TestDefaultDurationValues(t *testing.T) {
	// Verify duration defaults are reasonable
	if DefaultHTTPTimeout < time.Second {
		t.Error("DefaultHTTPTimeout should be at least 1 second")
	}
	if DefaultGRPCTimeout < time.Second {
		t.Error("DefaultGRPCTimeout should be at least 1 second")
	}
	if DefaultDatabaseConnMaxLifetime < time.Minute {
		t.Error("DefaultDatabaseConnMaxLifetime should be at least 1 minute")
	}
}

func TestHTTPConfigGetAddress(t *testing.T) {
	tests := []struct {
		name     string
		config   *HTTPConfig
		expected string
	}{
		{
			name:     "port only",
			config:   &HTTPConfig{Port: 8080},
			expected: ":8080",
		},
		{
			name:     "address only",
			config:   &HTTPConfig{Address: ":9090"},
			expected: ":9090",
		},
		{
			name:     "full address",
			config:   &HTTPConfig{Address: "127.0.0.1:8080"},
			expected: "127.0.0.1:8080",
		},
		{
			name:     "port takes precedence",
			config:   &HTTPConfig{Port: 3000, Address: ":9090"},
			expected: ":3000",
		},
		{
			name:     "empty",
			config:   &HTTPConfig{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.GetAddress(); got != tt.expected {
				t.Errorf("GetAddress() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGRPCConfigGetAddress(t *testing.T) {
	tests := []struct {
		name     string
		config   *GRPCConfig
		expected string
	}{
		{
			name:     "port only",
			config:   &GRPCConfig{Port: 9090},
			expected: ":9090",
		},
		{
			name:     "address only",
			config:   &GRPCConfig{Address: ":50051"},
			expected: ":50051",
		},
		{
			name:     "full address",
			config:   &GRPCConfig{Address: "127.0.0.1:9090"},
			expected: "127.0.0.1:9090",
		},
		{
			name:     "port takes precedence",
			config:   &GRPCConfig{Port: 50051, Address: ":9090"},
			expected: ":50051",
		},
		{
			name:     "empty",
			config:   &GRPCConfig{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.GetAddress(); got != tt.expected {
				t.Errorf("GetAddress() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHTTPConfigValidation(t *testing.T) {
	t.Run("no address or port", func(t *testing.T) {
		c := &HTTPConfig{}
		errs := c.Validate()
		if len(errs) == 0 {
			t.Error("expected validation error when both address and port are empty")
		}
	})

	t.Run("port provided", func(t *testing.T) {
		c := &HTTPConfig{Port: 8080}
		errs := c.Validate()
		if len(errs) != 0 {
			t.Errorf("unexpected validation errors: %v", errs)
		}
	})

	t.Run("address provided", func(t *testing.T) {
		c := &HTTPConfig{Address: ":8080"}
		errs := c.Validate()
		if len(errs) != 0 {
			t.Errorf("unexpected validation errors: %v", errs)
		}
	})
}

func TestGRPCConfigValidation(t *testing.T) {
	t.Run("no address or port", func(t *testing.T) {
		c := &GRPCConfig{}
		errs := c.Validate()
		if len(errs) == 0 {
			t.Error("expected validation error when both address and port are empty")
		}
	})

	t.Run("port provided", func(t *testing.T) {
		c := &GRPCConfig{Port: 9090}
		errs := c.Validate()
		if len(errs) != 0 {
			t.Errorf("unexpected validation errors: %v", errs)
		}
	})

	t.Run("address provided", func(t *testing.T) {
		c := &GRPCConfig{Address: ":9090"}
		errs := c.Validate()
		if len(errs) != 0 {
			t.Errorf("unexpected validation errors: %v", errs)
		}
	})
}
