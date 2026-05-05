// Package config provides configuration management for the Firefly framework.
// This file implements environment variable binding utilities.
package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/spf13/viper"
)

// bindStructEnv binds environment variables to a struct based on struct tags.
// Supported tags:
//   - env: Environment variable name
//   - envDefault: Default value if environment variable is not set
//   - envSeparator: Separator for slice values (default: ",")
//
// Example:
//
//	type Config struct {
//	    DSN      string   `env:"DATABASE_DSN" envDefault:"localhost:3306"`
//	    Port     int      `env:"SERVER_PORT" envDefault:"8080"`
//	    Debug    bool     `env:"DEBUG"`
//	    Hosts    []string `env:"HOSTS" envSeparator:","`
//	}
func bindStructEnv(v *viper.Viper, prefix string, val any) error {
	return bindStructEnvRecursive(v, prefix, reflect.ValueOf(val))
}

// bindStructEnvRecursive recursively binds environment variables to struct fields.
func bindStructEnvRecursive(v *viper.Viper, prefix string, val reflect.Value) error {
	// Dereference pointers
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	// Only process structs
	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get environment variable name from tag
		envName := field.Tag.Get("env")
		if envName == "" {
			// If no env tag, try to bind nested struct
			if fieldValue.Kind() == reflect.Ptr || fieldValue.Kind() == reflect.Struct {
				fieldPrefix := prefix
				if prefix != "" {
					fieldPrefix = prefix + "_"
				}
				// Use yaml/json tag name if available
				if yamlTag := field.Tag.Get("yaml"); yamlTag != "" {
					fieldPrefix += strings.Split(yamlTag, ",")[0]
				} else if jsonTag := field.Tag.Get("json"); jsonTag != "" {
					fieldPrefix += strings.Split(jsonTag, ",")[0]
				} else {
					fieldPrefix += field.Name
				}
				if err := bindStructEnvRecursive(v, fieldPrefix, fieldValue); err != nil {
					return err
				}
			}
			continue
		}

		// Build full environment variable name
		fullEnvName := envName
		if prefix != "" {
			fullEnvName = prefix + "_" + envName
		}

		// Get default value from tag
		envDefault := field.Tag.Get("envDefault")

		// Bind the environment variable
		if envDefault != "" {
			v.SetDefault(fullEnvName, envDefault)
		}
		if err := v.BindEnv(fullEnvName); err != nil {
			return fmt.Errorf("failed to bind env %q: %w", fullEnvName, err)
		}
	}

	return nil
}

// EnvBinding represents an environment variable binding.
type EnvBinding struct {
	Key      string // Config key
	EnvVar   string // Environment variable name
	Required bool   // Whether the variable is required
	Default  string // Default value
}

// EnvBindingOption is an option for environment variable binding.
type EnvBindingOption func(*EnvBinding)

// WithRequired marks the environment variable as required.
func WithRequired() EnvBindingOption {
	return func(b *EnvBinding) {
		b.Required = true
	}
}

// WithDefault sets a default value for the environment variable.
func WithDefault(defaultValue string) EnvBindingOption {
	return func(b *EnvBinding) {
		b.Default = defaultValue
	}
}

// BindEnvWithOptions binds an environment variable with options.
//
// Example:
//
//	cfg.BindEnvWithOptions("database.dsn", "DATABASE_DSN",
//	    config.WithRequired(),
//	    config.WithDefault("localhost:3306"),
//	)
func (c *Config) BindEnvWithOptions(key, envVar string, opts ...EnvBindingOption) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	binding := &EnvBinding{
		Key:    key,
		EnvVar: envVar,
	}

	for _, opt := range opts {
		opt(binding)
	}

	// Set default if provided
	if binding.Default != "" {
		c.viper.SetDefault(key, binding.Default)
	}

	// Bind environment variable
	if err := c.viper.BindEnv(key, envVar); err != nil {
		return fmt.Errorf("config: failed to bind env %q to key %q: %w", envVar, key, err)
	}

	return nil
}

// GetEnv retrieves a value from environment variable with fallback.
// If the environment variable is not set, it returns the default value.
//
// Example:
//
//	dsn := cfg.GetEnv("DATABASE_DSN", "localhost:3306")
func (c *Config) GetEnv(key string, defaultValue string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.viper.IsSet(key) {
		return c.viper.GetString(key)
	}
	return defaultValue
}

// MustGetEnv retrieves a required environment variable.
// It panics if the environment variable is not set.
//
// Example:
//
//	dsn := cfg.MustGetEnv("DATABASE_DSN") // panics if DATABASE_DSN is not set
func (c *Config) MustGetEnv(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.viper.IsSet(key) {
		panic(fmt.Sprintf("config: required environment variable %q is not set", key))
	}
	return c.viper.GetString(key)
}

// SetEnvKeyReplacer sets the replacer for environment variable keys.
// This allows custom transformation of environment variable names to config keys.
//
// Example:
//
//	// Replace dots with underscores in environment variable names
//	cfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
func (c *Config) SetEnvKeyReplacer(replacer *strings.Replacer) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.viper.SetEnvKeyReplacer(replacer)
}

// AllowEmptyEnv sets whether empty environment variables are considered set.
func (c *Config) AllowEmptyEnv(allow bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.viper.AllowEmptyEnv(allow)
}
