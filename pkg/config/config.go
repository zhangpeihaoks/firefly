// Package config provides configuration management for the Firefly framework.
// It uses Viper for loading configuration from files and environment variables.
package config

import (
	"fmt"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config is the configuration manager.
type Config struct {
	viper *viper.Viper
	mu    sync.RWMutex
}

// Option is the configuration option function.
type Option func(*Config)

// New creates a new configuration manager.
func New(opts ...Option) *Config {
	c := &Config{
		viper: viper.New(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithName sets the configuration file name (without extension).
func WithName(name string) Option {
	return func(c *Config) {
		c.viper.SetConfigName(name)
	}
}

// WithPath sets the configuration file path.
func WithPath(path string) Option {
	return func(c *Config) {
		c.viper.AddConfigPath(path)
	}
}

// WithType sets the configuration file type.
func WithType(configType string) Option {
	return func(c *Config) {
		c.viper.SetConfigType(configType)
	}
}

// WithEnvPrefix sets the environment variable prefix.
func WithEnvPrefix(prefix string) Option {
	return func(c *Config) {
		c.viper.SetEnvPrefix(prefix)
	}
}

// Load loads configuration from a file into the target structure.
func (c *Config) Load(path string, target any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.viper.SetConfigFile(path)
	if err := c.viper.ReadInConfig(); err != nil {
		return fmt.Errorf("config: failed to read config file: %w", err)
	}
	if err := c.viper.Unmarshal(target); err != nil {
		return fmt.Errorf("config: failed to unmarshal config: %w", err)
	}
	return nil
}

// LoadFromYAML loads configuration from YAML bytes into the target structure.
func (c *Config) LoadFromYAML(data []byte, target any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.viper.SetConfigType("yaml")
	if err := c.viper.ReadConfig(strings.NewReader(string(data))); err != nil {
		return fmt.Errorf("config: failed to read yaml config: %w", err)
	}
	if err := c.viper.Unmarshal(target); err != nil {
		return fmt.Errorf("config: failed to unmarshal config: %w", err)
	}
	return nil
}

// BindEnv binds an environment variable to a config key.
func (c *Config) BindEnv(input ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.viper.BindEnv(input...)
}

// Watch watches for configuration changes.
func (c *Config) Watch(key string, onChange func(any)) {
	c.viper.OnConfigChange(func(e fsnotify.Event) {
		if e.Name == key {
			onChange(c.viper.Get(key))
		}
	})
	c.viper.WatchConfig()
}

// Get returns the value for the given key.
func (c *Config) Get(key string) any {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper.Get(key)
}

// Set sets the value for the given key.
func (c *Config) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.viper.Set(key, value)
}

// GetString returns the string value for the given key.
func (c *Config) GetString(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper.GetString(key)
}

// GetInt returns the int value for the given key.
func (c *Config) GetInt(key string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper.GetInt(key)
}

// GetBool returns the bool value for the given key.
func (c *Config) GetBool(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper.GetBool(key)
}

// IsSet returns true if the key is set.
func (c *Config) IsSet(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.viper.IsSet(key)
}
