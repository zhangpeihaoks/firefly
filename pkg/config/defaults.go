// Package config provides configuration management for the Firefly framework.
// This file implements default configuration functions.
package config

import (
	"fmt"
	"time"
)

// Default configuration values.
const (
	// Application defaults
	DefaultName    = "firefly-app"
	DefaultVersion = "1.0.0"

	// HTTP defaults
	DefaultHTTPNetwork      = "tcp"
	DefaultHTTPPort         = 8080
	DefaultHTTPTimeout      = 30 * time.Second
	DefaultHTTPReadTimeout  = 30 * time.Second
	DefaultHTTPWriteTimeout = 30 * time.Second
	DefaultHTTPIdleTimeout  = 120 * time.Second

	// gRPC defaults
	DefaultGRPCNetwork        = "tcp"
	DefaultGRPCPort           = 9090
	DefaultGRPCTimeout        = 30 * time.Second
	DefaultGRPCMaxRecvMsgSize = 4 * 1024 * 1024 // 4MB
	DefaultGRPCMaxSendMsgSize = 4 * 1024 * 1024 // 4MB

	// Log defaults
	DefaultLogFileName   = "logs/app.log"
	DefaultLogMaxSize    = 100 // MB
	DefaultLogMaxBackups = 3
	DefaultLogMaxAge     = 7 // days
	DefaultLogLevel      = "info"
	DefaultLogJSONFormat = true
	DefaultLogLocation   = true

	// Registry defaults
	DefaultRegistryType    = "consul"
	DefaultRegistryTimeout = 5 * time.Second

	// Database defaults
	DefaultDatabaseDriver          = "mysql"
	DefaultDatabaseMaxOpenConns    = 100
	DefaultDatabaseMaxIdleConns    = 10
	DefaultDatabaseConnMaxLifetime = 30 * time.Minute
	DefaultDatabaseConnMaxIdleTime = 10 * time.Minute

	// Redis defaults
	DefaultRedisDB           = 0
	DefaultRedisPoolSize     = 10
	DefaultRedisMinIdleConns = 5
	DefaultRedisMaxRetries   = 3

	// Tracing defaults
	DefaultTracingEnabled      = false
	DefaultTracingSamplerRatio = 1.0
	DefaultTracingExporterType = "otlp"
	DefaultTracingInsecure     = false

	// Metrics defaults
	DefaultMetricsEnabled = true
	DefaultMetricsPath    = "/metrics"

	// Serializer defaults
	DefaultSerializerMode = "json"
)

// Bootstrap is the application bootstrap configuration.
// It contains all configuration sections for the application.
type Bootstrap struct {
	Name       string            `yaml:"name" json:"name" validate:"required"`
	Version    string            `yaml:"version" json:"version"`
	Metadata   map[string]string `yaml:"metadata" json:"metadata"`
	HTTP       *HTTPConfig       `yaml:"http" json:"http"`
	GRPC       *GRPCConfig       `yaml:"grpc" json:"grpc"`
	Log        *LogConfig        `yaml:"log" json:"log"`
	Registry   *RegistryConfig   `yaml:"registry" json:"registry"`
	Database   *DatabaseConfig   `yaml:"database" json:"database"`
	Redis      *RedisConfig      `yaml:"redis" json:"redis"`
	Tracing    *TracingConfig    `yaml:"tracing" json:"tracing"`
	Metrics    *MetricsConfig    `yaml:"metrics" json:"metrics"`
	Serializer *SerializerConfig `yaml:"serializer" json:"serializer"`
}

// Validate implements the Validator interface.
func (b *Bootstrap) Validate() ValidationErrors {
	var errors ValidationErrors

	if b.Name == "" {
		errors.Add("name", "is required")
	}

	if b.HTTP != nil {
		for _, err := range b.HTTP.Validate() {
			err.Field = "http." + err.Field
			errors = append(errors, err)
		}
	}

	if b.GRPC != nil {
		for _, err := range b.GRPC.Validate() {
			err.Field = "grpc." + err.Field
			errors = append(errors, err)
		}
	}

	if b.Log != nil {
		for _, err := range b.Log.Validate() {
			err.Field = "log." + err.Field
			errors = append(errors, err)
		}
	}

	if b.Registry != nil {
		for _, err := range b.Registry.Validate() {
			err.Field = "registry." + err.Field
			errors = append(errors, err)
		}
	}

	if b.Database != nil {
		for _, err := range b.Database.Validate() {
			err.Field = "database." + err.Field
			errors = append(errors, err)
		}
	}

	if b.Redis != nil {
		for _, err := range b.Redis.Validate() {
			err.Field = "redis." + err.Field
			errors = append(errors, err)
		}
	}

	if b.Tracing != nil {
		for _, err := range b.Tracing.Validate() {
			err.Field = "tracing." + err.Field
			errors = append(errors, err)
		}
	}

	if b.Metrics != nil {
		for _, err := range b.Metrics.Validate() {
			err.Field = "metrics." + err.Field
			errors = append(errors, err)
		}
	}

	if b.Serializer != nil {
		for _, err := range b.Serializer.Validate() {
			err.Field = "serializer." + err.Field
			errors = append(errors, err)
		}
	}

	return errors
}

// HTTPConfig is the HTTP server configuration.
type HTTPConfig struct {
	Network      string        `yaml:"network" json:"network"`
	Address      string        `yaml:"address" json:"address"` // Complete address (e.g., ":8080", "127.0.0.1:8080")
	Port         int           `yaml:"port" json:"port"`       // Port number only (e.g., 8080). Takes precedence over Address.
	Timeout      time.Duration `yaml:"timeout" json:"timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
	TLS          *TLSConfig    `yaml:"tls" json:"tls"`
}

// Validate implements the Validator interface.
func (c *HTTPConfig) Validate() ValidationErrors {
	var errors ValidationErrors

	// Either Address or Port must be provided
	if c.Address == "" && c.Port == 0 {
		errors.Add("address or port", "is required")
	}

	if c.TLS != nil {
		for _, err := range c.TLS.Validate() {
			err.Field = "tls." + err.Field
			errors = append(errors, err)
		}
	}

	return errors
}

// GetAddress returns the complete address for the HTTP server.
// If Port is specified, it returns ":<port>". Otherwise, it returns Address.
func (c *HTTPConfig) GetAddress() string {
	if c.Port > 0 {
		return fmt.Sprintf(":%d", c.Port)
	}
	return c.Address
}

// GRPCConfig is the gRPC server configuration.
type GRPCConfig struct {
	Network        string        `yaml:"network" json:"network"`
	Address        string        `yaml:"address" json:"address"` // Complete address (e.g., ":9090", "127.0.0.1:9090")
	Port           int           `yaml:"port" json:"port"`       // Port number only (e.g., 9090). Takes precedence over Address.
	Timeout        time.Duration `yaml:"timeout" json:"timeout"`
	MaxRecvMsgSize int           `yaml:"max_recv_msg_size" json:"max_recv_msg_size"`
	MaxSendMsgSize int           `yaml:"max_send_msg_size" json:"max_send_msg_size"`
	TLS            *TLSConfig    `yaml:"tls" json:"tls"`
}

// Validate implements the Validator interface.
func (c *GRPCConfig) Validate() ValidationErrors {
	var errors ValidationErrors

	// Either Address or Port must be provided
	if c.Address == "" && c.Port == 0 {
		errors.Add("address or port", "is required")
	}

	if c.TLS != nil {
		for _, err := range c.TLS.Validate() {
			err.Field = "tls." + err.Field
			errors = append(errors, err)
		}
	}

	return errors
}

// GetAddress returns the complete address for the gRPC server.
// If Port is specified, it returns ":<port>". Otherwise, it returns Address.
func (c *GRPCConfig) GetAddress() string {
	if c.Port > 0 {
		return fmt.Sprintf(":%d", c.Port)
	}
	return c.Address
}

// LogConfig is the logging configuration.
type LogConfig struct {
	FileName   string `yaml:"filename" json:"filename"`
	MaxSize    int    `yaml:"max_size" json:"max_size"`
	MaxBackups int    `yaml:"max_backups" json:"max_backups"`
	MaxAge     int    `yaml:"max_age" json:"max_age"`
	Level      string `yaml:"level" json:"level" validate:"oneof=debug,info,warn,error"`
	JSONFormat bool   `yaml:"json_format" json:"json_format"`
	Location   bool   `yaml:"location" json:"location"`
}

// Validate implements the Validator interface.
func (c *LogConfig) Validate() ValidationErrors {
	var errors ValidationErrors

	if c.Level != "" {
		validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
		if !validLevels[c.Level] {
			errors.Add("level", "must be one of: debug, info, warn, error")
		}
	}

	return errors
}

// RegistryConfig is the service registry configuration.
type RegistryConfig struct {
	Type     string            `yaml:"type" json:"type" validate:"required,oneof=consul,etcd,nacos,file"`
	Address  string            `yaml:"address" json:"address" validate:"required"`
	Timeout  time.Duration     `yaml:"timeout" json:"timeout"`
	Metadata map[string]string `yaml:"metadata" json:"metadata"`
}

// Validate implements the Validator interface.
func (c *RegistryConfig) Validate() ValidationErrors {
	var errors ValidationErrors

	if c.Type == "" {
		errors.Add("type", "is required")
	} else {
		validTypes := map[string]bool{"consul": true, "etcd": true, "nacos": true, "file": true}
		if !validTypes[c.Type] {
			errors.Add("type", "must be one of: consul, etcd, nacos, file")
		}
	}

	if c.Address == "" {
		errors.Add("address", "is required")
	}

	return errors
}

// DatabaseConfig is the database configuration.
type DatabaseConfig struct {
	Driver          string        `yaml:"driver" json:"driver" validate:"required,oneof=mysql,postgres,mongodb"`
	DSN             string        `yaml:"dsn" json:"dsn" validate:"required"`
	MaxOpenConns    int           `yaml:"max_open_conns" json:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns" json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" json:"conn_max_idle_time"`
}

// Validate implements the Validator interface.
func (c *DatabaseConfig) Validate() ValidationErrors {
	var errors ValidationErrors

	if c.Driver == "" {
		errors.Add("driver", "is required")
	} else {
		validDrivers := map[string]bool{"mysql": true, "postgres": true, "mongodb": true}
		if !validDrivers[c.Driver] {
			errors.Add("driver", "must be one of: mysql, postgres, mongodb")
		}
	}

	if c.DSN == "" {
		errors.Add("dsn", "is required")
	}

	return errors
}

// RedisConfig is the Redis configuration.
type RedisConfig struct {
	Address      string `yaml:"address" json:"address" validate:"required"`
	Password     string `yaml:"password" json:"password"`
	DB           int    `yaml:"db" json:"db"`
	PoolSize     int    `yaml:"pool_size" json:"pool_size"`
	MinIdleConns int    `yaml:"min_idle_conns" json:"min_idle_conns"`
	MaxRetries   int    `yaml:"max_retries" json:"max_retries"`
}

// Validate implements the Validator interface.
func (c *RedisConfig) Validate() ValidationErrors {
	var errors ValidationErrors

	if c.Address == "" {
		errors.Add("address", "is required")
	}

	return errors
}

// TracingConfig is the distributed tracing configuration.
type TracingConfig struct {
	Enabled      bool    `yaml:"enabled" json:"enabled"`
	Endpoint     string  `yaml:"endpoint" json:"endpoint"`
	SamplerRatio float64 `yaml:"sampler_ratio" json:"sampler_ratio"`
	// ExporterType specifies the exporter type: otlp, jaeger, stdout
	ExporterType string `yaml:"exporter_type" json:"exporter_type"`
	// ServiceName is the name of the service for tracing
	ServiceName string `yaml:"service_name" json:"service_name"`
	// Insecure determines whether to use insecure connection for OTLP
	Insecure bool `yaml:"insecure" json:"insecure"`
}

// Validate implements the Validator interface.
func (c *TracingConfig) Validate() ValidationErrors {
	var errors ValidationErrors

	if c.Enabled && c.Endpoint == "" {
		errors.Add("endpoint", "is required when tracing is enabled")
	}

	if c.SamplerRatio < 0 || c.SamplerRatio > 1 {
		errors.Add("sampler_ratio", "must be between 0 and 1")
	}

	if c.ExporterType != "" {
		validTypes := map[string]bool{"otlp": true, "jaeger": true, "stdout": true}
		if !validTypes[c.ExporterType] {
			errors.Add("exporter_type", "must be one of: otlp, jaeger, stdout")
		}
	}

	return errors
}

// MetricsConfig is the metrics configuration.
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Path    string `yaml:"path" json:"path"`
}

// Validate implements the Validator interface.
func (c *MetricsConfig) Validate() ValidationErrors {
	var errors ValidationErrors

	if c.Path == "" {
		errors.Add("path", "is required")
	}

	return errors
}

// TLSConfig is the TLS configuration.
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	CertFile string `yaml:"cert_file" json:"cert_file"`
	KeyFile  string `yaml:"key_file" json:"key_file"`
}

// Validate implements the Validator interface.
func (c *TLSConfig) Validate() ValidationErrors {
	var errors ValidationErrors

	if c.Enabled {
		if c.CertFile == "" {
			errors.Add("cert_file", "is required when TLS is enabled")
		}
		if c.KeyFile == "" {
			errors.Add("key_file", "is required when TLS is enabled")
		}
	}

	return errors
}

// SerializerConfig is the serializer configuration.
type SerializerConfig struct {
	Mode string `yaml:"mode" json:"mode" validate:"oneof=json,protobuf"`
}

// Validate implements the Validator interface.
func (c *SerializerConfig) Validate() ValidationErrors {
	var errors ValidationErrors

	if c.Mode != "" {
		validModes := map[string]bool{"json": true, "protobuf": true}
		if !validModes[c.Mode] {
			errors.Add("mode", "must be one of: json, protobuf")
		}
	}

	return errors
}

// DefaultBootstrap returns a Bootstrap with default values.
func DefaultBootstrap() *Bootstrap {
	return &Bootstrap{
		Name:    DefaultName,
		Version: DefaultVersion,
		HTTP:    DefaultHTTPConfig(),
		GRPC:    DefaultGRPCConfig(),
		Log:     DefaultLogConfig(),
		Metrics: DefaultMetricsConfig(),
		Serializer: &SerializerConfig{
			Mode: DefaultSerializerMode,
		},
	}
}

// DefaultHTTPConfig returns an HTTPConfig with default values.
func DefaultHTTPConfig() *HTTPConfig {
	return &HTTPConfig{
		Network:      DefaultHTTPNetwork,
		Port:         DefaultHTTPPort,
		Timeout:      DefaultHTTPTimeout,
		ReadTimeout:  DefaultHTTPReadTimeout,
		WriteTimeout: DefaultHTTPWriteTimeout,
		IdleTimeout:  DefaultHTTPIdleTimeout,
	}
}

// DefaultGRPCConfig returns a GRPCConfig with default values.
func DefaultGRPCConfig() *GRPCConfig {
	return &GRPCConfig{
		Network:        DefaultGRPCNetwork,
		Port:           DefaultGRPCPort,
		Timeout:        DefaultGRPCTimeout,
		MaxRecvMsgSize: DefaultGRPCMaxRecvMsgSize,
		MaxSendMsgSize: DefaultGRPCMaxSendMsgSize,
	}
}

// DefaultLogConfig returns a LogConfig with default values.
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		FileName:   DefaultLogFileName,
		MaxSize:    DefaultLogMaxSize,
		MaxBackups: DefaultLogMaxBackups,
		MaxAge:     DefaultLogMaxAge,
		Level:      DefaultLogLevel,
		JSONFormat: DefaultLogJSONFormat,
		Location:   DefaultLogLocation,
	}
}

// DefaultRegistryConfig returns a RegistryConfig with default values.
func DefaultRegistryConfig() *RegistryConfig {
	return &RegistryConfig{
		Type:    DefaultRegistryType,
		Timeout: DefaultRegistryTimeout,
	}
}

// DefaultDatabaseConfig returns a DatabaseConfig with default values.
func DefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Driver:          DefaultDatabaseDriver,
		MaxOpenConns:    DefaultDatabaseMaxOpenConns,
		MaxIdleConns:    DefaultDatabaseMaxIdleConns,
		ConnMaxLifetime: DefaultDatabaseConnMaxLifetime,
		ConnMaxIdleTime: DefaultDatabaseConnMaxIdleTime,
	}
}

// DefaultRedisConfig returns a RedisConfig with default values.
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		DB:           DefaultRedisDB,
		PoolSize:     DefaultRedisPoolSize,
		MinIdleConns: DefaultRedisMinIdleConns,
		MaxRetries:   DefaultRedisMaxRetries,
	}
}

// DefaultTracingConfig returns a TracingConfig with default values.
func DefaultTracingConfig() *TracingConfig {
	return &TracingConfig{
		Enabled:      DefaultTracingEnabled,
		SamplerRatio: DefaultTracingSamplerRatio,
		ExporterType: DefaultTracingExporterType,
		Insecure:     DefaultTracingInsecure,
	}
}

// DefaultMetricsConfig returns a MetricsConfig with default values.
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Enabled: DefaultMetricsEnabled,
		Path:    DefaultMetricsPath,
	}
}

// ApplyDefaults applies default values to a Bootstrap configuration.
// This function fills in missing values with defaults.
func ApplyDefaults(b *Bootstrap) {
	if b.Name == "" {
		b.Name = DefaultName
	}
	if b.Version == "" {
		b.Version = DefaultVersion
	}

	if b.HTTP == nil {
		b.HTTP = DefaultHTTPConfig()
	} else {
		applyHTTPDefaults(b.HTTP)
	}

	if b.GRPC == nil {
		b.GRPC = DefaultGRPCConfig()
	} else {
		applyGRPCDefaults(b.GRPC)
	}

	if b.Log == nil {
		b.Log = DefaultLogConfig()
	} else {
		applyLogDefaults(b.Log)
	}

	if b.Registry == nil {
		b.Registry = DefaultRegistryConfig()
	} else {
		applyRegistryDefaults(b.Registry)
	}

	if b.Database == nil {
		b.Database = DefaultDatabaseConfig()
	} else {
		applyDatabaseDefaults(b.Database)
	}

	if b.Redis == nil {
		b.Redis = DefaultRedisConfig()
	} else {
		applyRedisDefaults(b.Redis)
	}

	if b.Tracing == nil {
		b.Tracing = DefaultTracingConfig()
	} else {
		applyTracingDefaults(b.Tracing)
	}

	if b.Metrics == nil {
		b.Metrics = DefaultMetricsConfig()
	} else {
		applyMetricsDefaults(b.Metrics)
	}

	if b.Serializer == nil {
		b.Serializer = &SerializerConfig{Mode: DefaultSerializerMode}
	} else {
		applySerializerDefaults(b.Serializer)
	}
}

func applyHTTPDefaults(c *HTTPConfig) {
	if c.Network == "" {
		c.Network = DefaultHTTPNetwork
	}
	// Note: We don't set default Address/Port here since GetAddress() handles it
	if c.Timeout == 0 {
		c.Timeout = DefaultHTTPTimeout
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = DefaultHTTPReadTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = DefaultHTTPWriteTimeout
	}
	if c.IdleTimeout == 0 {
		c.IdleTimeout = DefaultHTTPIdleTimeout
	}
}

func applyGRPCDefaults(c *GRPCConfig) {
	if c.Network == "" {
		c.Network = DefaultGRPCNetwork
	}
	// Note: We don't set default Address/Port here since GetAddress() handles it
	if c.Timeout == 0 {
		c.Timeout = DefaultGRPCTimeout
	}
	if c.MaxRecvMsgSize == 0 {
		c.MaxRecvMsgSize = DefaultGRPCMaxRecvMsgSize
	}
	if c.MaxSendMsgSize == 0 {
		c.MaxSendMsgSize = DefaultGRPCMaxSendMsgSize
	}
}

func applyLogDefaults(c *LogConfig) {
	if c.FileName == "" {
		c.FileName = DefaultLogFileName
	}
	if c.MaxSize == 0 {
		c.MaxSize = DefaultLogMaxSize
	}
	if c.MaxBackups == 0 {
		c.MaxBackups = DefaultLogMaxBackups
	}
	if c.MaxAge == 0 {
		c.MaxAge = DefaultLogMaxAge
	}
	if c.Level == "" {
		c.Level = DefaultLogLevel
	}
}

func applyRegistryDefaults(c *RegistryConfig) {
	if c.Type == "" {
		c.Type = DefaultRegistryType
	}
	if c.Timeout == 0 {
		c.Timeout = DefaultRegistryTimeout
	}
}

func applyDatabaseDefaults(c *DatabaseConfig) {
	if c.Driver == "" {
		c.Driver = DefaultDatabaseDriver
	}
	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = DefaultDatabaseMaxOpenConns
	}
	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = DefaultDatabaseMaxIdleConns
	}
	if c.ConnMaxLifetime == 0 {
		c.ConnMaxLifetime = DefaultDatabaseConnMaxLifetime
	}
	if c.ConnMaxIdleTime == 0 {
		c.ConnMaxIdleTime = DefaultDatabaseConnMaxIdleTime
	}
}

func applyRedisDefaults(c *RedisConfig) {
	if c.PoolSize == 0 {
		c.PoolSize = DefaultRedisPoolSize
	}
	if c.MinIdleConns == 0 {
		c.MinIdleConns = DefaultRedisMinIdleConns
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = DefaultRedisMaxRetries
	}
}

func applyTracingDefaults(c *TracingConfig) {
	if c.SamplerRatio == 0 {
		c.SamplerRatio = DefaultTracingSamplerRatio
	}
	if c.ExporterType == "" {
		c.ExporterType = DefaultTracingExporterType
	}
}

func applyMetricsDefaults(c *MetricsConfig) {
	if c.Path == "" {
		c.Path = DefaultMetricsPath
	}
}

func applySerializerDefaults(c *SerializerConfig) {
	if c.Mode == "" {
		c.Mode = DefaultSerializerMode
	}
}
