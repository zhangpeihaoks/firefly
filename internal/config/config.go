// Package config provides configuration management for the Firefly framework.
// It uses Viper for configuration loading and supports YAML files.
//
// Deprecated: Use pkg/config instead. This package will be removed in a future version.
// Migration: change "github.com/zhangpeihaoks/firefly/internal/config" to
// "github.com/zhangpeihaoks/firefly/pkg/config" in your imports.
package config

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config is the configuration manager.
// It wraps Viper and provides thread-safe configuration operations.
type Config struct {
	viper *viper.Viper
	mu    sync.RWMutex
}

// ConfigOption is a configuration option function.
type ConfigOption func(*Config)

// New creates a new configuration manager with the given options.
//
// Example:
//
//	cfg := config.New()
//	var bootstrap Bootstrap
//	if err := cfg.Load("config.yaml", &bootstrap); err != nil {
//	    log.Fatal(err)
//	}
func New(opts ...ConfigOption) *Config {
	c := &Config{
		viper: viper.New(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Load loads configuration from a file into the target struct.
// The file format is determined by the file extension (yaml, yml, json, toml, etc.).
//
// Supported formats:
//   - YAML (.yaml, .yml)
//   - JSON (.json)
//   - TOML (.toml)
//   - INI (.ini)
//
// Example:
//
//	var bootstrap Bootstrap
//	if err := cfg.Load("config.yaml", &bootstrap); err != nil {
//	    log.Fatal(err)
//	}
func (c *Config) Load(path string, target any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.viper.SetConfigFile(path)

	// Read the config file
	if err := c.viper.ReadInConfig(); err != nil {
		return fmt.Errorf("config: failed to read config file %q: %w", path, err)
	}

	// Unmarshal into target
	if err := c.viper.Unmarshal(target); err != nil {
		return fmt.Errorf("config: failed to unmarshal config: %w", err)
	}

	return nil
}

// LoadWithEnv loads configuration from a file and applies environment variable overrides.
// Environment variables take precedence over file configuration.
//
// The envPrefix is used to prefix environment variables (e.g., "APP" -> APP_DATABASE_DSN).
// Set envPrefix to empty string to disable prefixing.
//
// Example:
//
//	var bootstrap Bootstrap
//	// Environment variables like APP_HTTP_ADDRESS, APP_GRPC_ADDRESS will override config
//	if err := cfg.LoadWithEnv("config.yaml", "APP", &bootstrap); err != nil {
//	    log.Fatal(err)
//	}
func (c *Config) LoadWithEnv(path string, envPrefix string, target any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.viper.SetConfigFile(path)

	// Read the config file
	if err := c.viper.ReadInConfig(); err != nil {
		return fmt.Errorf("config: failed to read config file %q: %w", path, err)
	}

	// Set environment variable prefix
	if envPrefix != "" {
		c.viper.SetEnvPrefix(envPrefix)
	}

	// Enable automatic environment variable binding
	c.viper.AutomaticEnv()

	// Unmarshal into target (environment variables will override file values)
	if err := c.viper.Unmarshal(target); err != nil {
		return fmt.Errorf("config: failed to unmarshal config: %w", err)
	}

	return nil
}

// LoadAndValidate loads configuration from a file, applies environment variable overrides,
// applies defaults, and validates the result.
//
// This is the recommended way to load configuration as it ensures:
// 1. File configuration is loaded
// 2. Environment variables override file values
// 3. Missing values are filled with defaults
// 4. Configuration is validated
//
// Example:
//
//	var bootstrap Bootstrap
//	if err := cfg.LoadAndValidate("config.yaml", "APP", &bootstrap); err != nil {
//	    log.Fatal(err)
//	}
func (c *Config) LoadAndValidate(path string, envPrefix string, target any) error {
	// Load with environment variable override
	if err := c.LoadWithEnv(path, envPrefix, target); err != nil {
		return err
	}

	// Apply defaults if target is a Bootstrap
	if bootstrap, ok := target.(*Bootstrap); ok {
		ApplyDefaults(bootstrap)
	}

	// Validate
	if err := Validate(target); err != nil {
		return fmt.Errorf("config: validation failed: %w", err)
	}

	return nil
}

// LoadFromYAML loads configuration from YAML data into the target struct.
// This is useful for loading configuration from embedded data or strings.
//
// Example:
//
//	var bootstrap Bootstrap
//	yamlData := []byte(`
//	name: myapp
//	version: 1.0.0
//	`)
//	if err := cfg.LoadFromYAML(yamlData, &bootstrap); err != nil {
//	    log.Fatal(err)
//	}
func (c *Config) LoadFromYAML(data []byte, target any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.viper.SetConfigType("yaml")

	if err := c.viper.ReadConfig(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("config: failed to read YAML data: %w", err)
	}

	if err := c.viper.Unmarshal(target); err != nil {
		return fmt.Errorf("config: failed to unmarshal YAML data: %w", err)
	}

	return nil
}

// LoadFromJSON loads configuration from JSON data into the target struct.
func (c *Config) LoadFromJSON(data []byte, target any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.viper.SetConfigType("json")

	if err := c.viper.ReadConfig(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("config: failed to read JSON data: %w", err)
	}

	if err := c.viper.Unmarshal(target); err != nil {
		return fmt.Errorf("config: failed to unmarshal JSON data: %w", err)
	}

	return nil
}

// BindEnv binds environment variables to configuration keys.
// If no env key is provided, the config key will be used as the env key.
//
// Example:
//
//	// Bind "database.dsn" to DATABASE_DSN environment variable
//	cfg.BindEnv("database.dsn", "DATABASE_DSN")
//
//	// Bind "server.port" to SERVER_PORT (automatic env key)
//	cfg.BindEnv("server.port")
func (c *Config) BindEnv(input ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.viper.BindEnv(input...)
}

// BindEnvMultiple binds multiple environment variables at once.
// The keys map config keys to environment variable names.
//
// Example:
//
//	cfg.BindEnvMultiple(map[string]string{
//	    "database.dsn":      "DATABASE_DSN",
//	    "database.driver":   "DATABASE_DRIVER",
//	    "server.http.port":  "HTTP_PORT",
//	})
func (c *Config) BindEnvMultiple(bindings map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, env := range bindings {
		if err := c.viper.BindEnv(key, env); err != nil {
			return fmt.Errorf("config: failed to bind env %q to key %q: %w", env, key, err)
		}
	}
	return nil
}

// BindStruct binds environment variables to a struct based on struct tags.
// The struct tag format is: `env:"ENV_VAR_NAME" envDefault:"default_value"`
//
// Example:
//
//	type Config struct {
//	    DSN    string `env:"DATABASE_DSN" envDefault:"localhost:3306"`
//	    Port   int    `env:"SERVER_PORT" envDefault:"8080"`
//	    Debug  bool   `env:"DEBUG"`
//	}
func (c *Config) BindStruct(prefix string, v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return bindStructEnv(c.viper, prefix, v)
}

// AutomaticEnv enables automatic environment variable binding.
// Environment variables that match config keys will be automatically bound.
func (c *Config) AutomaticEnv() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.viper.AutomaticEnv()
}

// SetEnvPrefix sets the prefix for environment variables.
func (c *Config) SetEnvPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.viper.SetEnvPrefix(prefix)
}

// Get retrieves a configuration value by key.
// Returns nil if the key does not exist.
func (c *Config) Get(key string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.viper.Get(key)
}

// GetString retrieves a string configuration value by key.
func (c *Config) GetString(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.viper.GetString(key)
}

// GetInt retrieves an integer configuration value by key.
func (c *Config) GetInt(key string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.viper.GetInt(key)
}

// GetBool retrieves a boolean configuration value by key.
func (c *Config) GetBool(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.viper.GetBool(key)
}

// GetStringMap retrieves a string map configuration value by key.
func (c *Config) GetStringMap(key string) map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.viper.GetStringMap(key)
}

// GetStringSlice retrieves a string slice configuration value by key.
func (c *Config) GetStringSlice(key string) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.viper.GetStringSlice(key)
}

// Set sets a configuration value.
func (c *Config) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.viper.Set(key, value)
}

// IsSet checks if a configuration key has been set.
func (c *Config) IsSet(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.viper.IsSet(key)
}

// Watch watches for configuration changes and calls the onChange function.
// This enables hot-reloading of configuration files.
//
// Example:
//
//	cfg.Watch("config.yaml", func() {
//	    log.Info("Configuration changed")
//	    // Reload configuration
//	})
func (c *Config) Watch(path string, onChange func()) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.viper.SetConfigFile(path)
	c.viper.WatchConfig()
	c.viper.OnConfigChange(func(e fsnotify.Event) {
		if onChange != nil {
			onChange()
		}
	})

	return nil
}

// Viper returns the underlying Viper instance for advanced usage.
// Use with caution as it bypasses the thread-safety of this wrapper.
func (c *Config) Viper() *viper.Viper {
	return c.viper
}

// MergeConfig merges configuration from a map into the existing configuration.
func (c *Config) MergeConfig(cfg map[string]any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.viper.MergeConfigMap(cfg)
}

// Reset resets the configuration to an empty state.
func (c *Config) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.viper = viper.New()
}

// Global default config instance
var defaultConfig *Config
var once sync.Once

// Default returns the default configuration manager instance.
// It is lazily initialized on first use.
func Default() *Config {
	once.Do(func() {
		defaultConfig = New()
	})
	return defaultConfig
}

// Load loads configuration using the default config manager.
func Load(path string, target any) error {
	return Default().Load(path, target)
}

// LoadFromYAML loads YAML configuration using the default config manager.
func LoadFromYAML(data []byte, target any) error {
	return Default().LoadFromYAML(data, target)
}

// Get retrieves a configuration value using the default config manager.
func Get(key string) any {
	return Default().Get(key)
}

// Set sets a configuration value using the default config manager.
func Set(key string, value any) {
	Default().Set(key, value)
}
