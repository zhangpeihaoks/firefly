// Package conf provides configuration structures for the Firefly framework.
package conf

import (
	"time"
)

// Bootstrap is the application bootstrap configuration.
type Bootstrap struct {
	// Name is the service name
	Name string `yaml:"name" json:"name"`
	// Version is the service version
	Version string `yaml:"version" json:"version"`
	// Metadata contains service metadata
	Metadata map[string]string `yaml:"metadata" json:"metadata"`
	// HTTP is the HTTP server configuration
	HTTP *HTTPConfig `yaml:"http" json:"http"`
	// GRPC is the gRPC server configuration
	GRPC *GRPCConfig `yaml:"grpc" json:"grpc"`
	// Log is the logging configuration
	Log *LogConfig `yaml:"log" json:"log"`
	// Registry is the service registry configuration
	Registry *RegistryConfig `yaml:"registry" json:"registry"`
	// Database is the database configuration
	Database *DatabaseConfig `yaml:"database" json:"database"`
	// Redis is the Redis configuration
	Redis *RedisConfig `yaml:"redis" json:"redis"`
	// Tracing is the tracing configuration
	Tracing *TracingConfig `yaml:"tracing" json:"tracing"`
	// Metrics is the metrics configuration
	Metrics *MetricsConfig `yaml:"metrics" json:"metrics"`
	// Serializer is the serializer configuration
	Serializer *SerializerConfig `yaml:"serializer" json:"serializer"`
}

// SerializerConfig is the serializer configuration.
type SerializerConfig struct {
	// Mode is the serialization mode: json, protobuf
	Mode string `yaml:"mode" json:"mode"`
	// DefaultMode is the default serialization mode
	DefaultMode string `yaml:"default_mode" json:"default_mode"`
}

// HTTPConfig is the HTTP server configuration.
type HTTPConfig struct {
	// Network is the network type (tcp, unix)
	Network string `yaml:"network" json:"network"`
	// Address is the listen address
	Address string `yaml:"address" json:"address"`
	// Timeout is the request timeout
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
	// ReadTimeout is the read timeout
	ReadTimeout time.Duration `yaml:"read_timeout" json:"read_timeout"`
	// WriteTimeout is the write timeout
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
	// IdleTimeout is the idle timeout
	IdleTimeout time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
	// TLS is the TLS configuration
	TLS *TLSConfig `yaml:"tls" json:"tls"`
}

// GRPCConfig is the gRPC server configuration.
type GRPCConfig struct {
	// Network is the network type (tcp, unix)
	Network string `yaml:"network" json:"network"`
	// Address is the listen address
	Address string `yaml:"address" json:"address"`
	// Timeout is the request timeout
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
	// MaxRecvMsgSize is the maximum receive message size
	MaxRecvMsgSize int `yaml:"max_recv_msg_size" json:"max_recv_msg_size"`
	// MaxSendMsgSize is the maximum send message size
	MaxSendMsgSize int `yaml:"max_send_msg_size" json:"max_send_msg_size"`
	// TLS is the TLS configuration
	TLS *TLSConfig `yaml:"tls" json:"tls"`
}

// LogConfig is the logging configuration.
type LogConfig struct {
	// FileName is the log file name
	FileName string `yaml:"filename" json:"filename"`
	// MaxSize is the maximum file size in MB
	MaxSize int `yaml:"max_size" json:"max_size"`
	// MaxBackups is the maximum number of backups
	MaxBackups int `yaml:"max_backups" json:"max_backups"`
	// MaxAge is the maximum retention days
	MaxAge int `yaml:"max_age" json:"max_age"`
	// Level is the log level
	Level string `yaml:"level" json:"level"`
	// JSONFormat enables JSON format
	JSONFormat bool `yaml:"json_format" json:"json_format"`
	// Location enables source code location
	Location bool `yaml:"location" json:"location"`
}

// RegistryConfig is the service registry configuration.
type RegistryConfig struct {
	// Type is the registry type (consul, etcd, nacos, file)
	Type string `yaml:"type" json:"type"`
	// Address is the registry address
	Address string `yaml:"address" json:"address"`
	// Timeout is the registry timeout
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
	// Metadata contains registry metadata
	Metadata map[string]string `yaml:"metadata" json:"metadata"`
}

// DatabaseConfig is the database configuration.
type DatabaseConfig struct {
	// Driver is the database driver (mysql, postgres, mongodb)
	Driver string `yaml:"driver" json:"driver"`
	// DSN is the data source name
	DSN string `yaml:"dsn" json:"dsn"`
	// MaxOpenConns is the maximum open connections
	MaxOpenConns int `yaml:"max_open_conns" json:"max_open_conns"`
	// MaxIdleConns is the maximum idle connections
	MaxIdleConns int `yaml:"max_idle_conns" json:"max_idle_conns"`
	// ConnMaxLifetime is the maximum connection lifetime
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
	// ConnMaxIdleTime is the maximum idle time
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" json:"conn_max_idle_time"`
}

// RedisConfig is the Redis configuration.
type RedisConfig struct {
	// Address is the Redis address
	Address string `yaml:"address" json:"address"`
	// Password is the Redis password
	Password string `yaml:"password" json:"password"`
	// DB is the Redis database number
	DB int `yaml:"db" json:"db"`
	// PoolSize is the connection pool size
	PoolSize int `yaml:"pool_size" json:"pool_size"`
	// MinIdleConns is the minimum idle connections
	MinIdleConns int `yaml:"min_idle_conns" json:"min_idle_conns"`
	// MaxRetries is the maximum retries
	MaxRetries int `yaml:"max_retries" json:"max_retries"`
}

// TracingConfig is the tracing configuration.
type TracingConfig struct {
	// Enabled enables tracing
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Endpoint is the tracing endpoint
	Endpoint string `yaml:"endpoint" json:"endpoint"`
	// SamplerRatio is the sampling ratio
	SamplerRatio float64 `yaml:"sampler_ratio" json:"sampler_ratio"`
}

// MetricsConfig is the metrics configuration.
type MetricsConfig struct {
	// Enabled enables metrics
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Path is the metrics endpoint path
	Path string `yaml:"path" json:"path"`
}

// TLSConfig is the TLS configuration.
type TLSConfig struct {
	// Enabled enables TLS
	Enabled bool `yaml:"enabled" json:"enabled"`
	// CertFile is the certificate file path
	CertFile string `yaml:"cert_file" json:"cert_file"`
	// KeyFile is the private key file path
	KeyFile string `yaml:"key_file" json:"key_file"`
}

// DefaultBootstrap returns the default bootstrap configuration.
func DefaultBootstrap() *Bootstrap {
	return &Bootstrap{
		Name:    "firefly",
		Version: "1.0.0",
		HTTP: &HTTPConfig{
			Network:      "tcp",
			Address:      ":8080",
			Timeout:      30 * time.Second,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		GRPC: &GRPCConfig{
			Network:        "tcp",
			Address:        ":9090",
			Timeout:        30 * time.Second,
			MaxRecvMsgSize: 4 * 1024 * 1024,
			MaxSendMsgSize: 4 * 1024 * 1024,
		},
		Log: &LogConfig{
			FileName:   "",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
			Level:      "info",
			JSONFormat: true,
			Location:   true,
		},
		Serializer: &SerializerConfig{
			Mode:        "json",
			DefaultMode: "json",
		},
		Tracing: &TracingConfig{
			Enabled:      false,
			SamplerRatio: 1.0,
		},
		Metrics: &MetricsConfig{
			Enabled: true,
			Path:    "/metrics",
		},
	}
}
