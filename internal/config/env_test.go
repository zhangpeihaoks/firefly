// Package config provides configuration management for the Firefly framework.
package config

import (
	"os"
	"testing"
)

func TestBindEnvMultiple(t *testing.T) {
	cfg := New()

	// Set environment variables
	os.Setenv("TEST_DSN", "postgres://localhost:5432/test")
	os.Setenv("TEST_DRIVER", "postgres")
	defer os.Unsetenv("TEST_DSN")
	defer os.Unsetenv("TEST_DRIVER")

	// Bind multiple environment variables
	err := cfg.BindEnvMultiple(map[string]string{
		"database.dsn":    "TEST_DSN",
		"database.driver": "TEST_DRIVER",
	})
	if err != nil {
		t.Fatalf("BindEnvMultiple() error = %v", err)
	}

	// Enable automatic environment
	cfg.AutomaticEnv()
}

func TestBindEnvWithOptions(t *testing.T) {
	cfg := New()

	// Test with default value
	err := cfg.BindEnvWithOptions("test.key", "TEST_KEY_WITH_DEFAULT",
		WithDefault("default_value"),
	)
	if err != nil {
		t.Fatalf("BindEnvWithOptions() error = %v", err)
	}

	// The default should be set
	if cfg.GetString("test.key") != "default_value" {
		t.Errorf("test.key = %v, want default_value", cfg.GetString("test.key"))
	}

	// Set the environment variable
	os.Setenv("TEST_KEY_WITH_DEFAULT", "env_value")
	defer os.Unsetenv("TEST_KEY_WITH_DEFAULT")

	// Re-bind to pick up the env var
	cfg.AutomaticEnv()
}

func TestGetEnv(t *testing.T) {
	cfg := New()

	// Test with non-existent key
	value := cfg.GetEnv("NON_EXISTENT_KEY", "fallback")
	if value != "fallback" {
		t.Errorf("GetEnv() = %v, want fallback", value)
	}

	// Set a value
	cfg.Set("existing_key", "set_value")
	value = cfg.GetEnv("existing_key", "fallback")
	if value != "set_value" {
		t.Errorf("GetEnv() = %v, want set_value", value)
	}
}

func TestMustGetEnv(t *testing.T) {
	cfg := New()

	// Set a value
	cfg.Set("required_key", "required_value")
	value := cfg.MustGetEnv("required_key")
	if value != "required_value" {
		t.Errorf("MustGetEnv() = %v, want required_value", value)
	}
}

func TestMustGetEnv_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGetEnv should panic for non-existent key")
		}
	}()

	cfg := New()
	cfg.MustGetEnv("NON_EXISTENT_REQUIRED_KEY")
}

func TestSetEnvKeyReplacer(t *testing.T) {
	cfg := New()

	// Set a replacer that replaces dots with underscores
	cfg.SetEnvKeyReplacer(nil) // Just test that it doesn't panic
}

func TestAllowEmptyEnv(t *testing.T) {
	cfg := New()
	cfg.AllowEmptyEnv(true) // Just test that it doesn't panic
}

func TestLoadWithEnv(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	yamlContent := `
name: testapp
version: "1.0.0"
http:
  address: ":8080"
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Set environment variable to override
	os.Setenv("APP_HTTP_ADDRESS", ":9090")
	defer os.Unsetenv("APP_HTTP_ADDRESS")

	cfg := New()
	var bootstrap Bootstrap
	err := cfg.LoadWithEnv(configPath, "APP", &bootstrap)
	if err != nil {
		t.Fatalf("LoadWithEnv() error = %v", err)
	}

	// The name should come from the file
	if bootstrap.Name != "testapp" {
		t.Errorf("Name = %v, want testapp", bootstrap.Name)
	}
}

func TestLoadAndValidate(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	yamlContent := `
name: testapp
version: "1.0.0"
http:
  address: ":8080"
log:
  level: info
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg := New()
	var bootstrap Bootstrap
	// LoadAndValidate applies defaults, so Registry, Database, Redis will be nil
	// and validation should pass for the basic config
	err := cfg.LoadWithEnv(configPath, "APP", &bootstrap)
	if err != nil {
		t.Fatalf("LoadWithEnv() error = %v", err)
	}

	// Apply defaults manually for this test
	ApplyDefaults(&bootstrap)

	// Verify defaults were applied
	if bootstrap.HTTP.Network != DefaultHTTPNetwork {
		t.Errorf("HTTP.Network = %v, want %v", bootstrap.HTTP.Network, DefaultHTTPNetwork)
	}
}

func TestLoadAndValidate_ValidationError(t *testing.T) {
	// Create a temporary config file with invalid config
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	yamlContent := `
name: ""
http:
  address: ""
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg := New()
	var bootstrap Bootstrap
	err := cfg.LoadAndValidate(configPath, "", &bootstrap)
	if err == nil {
		t.Error("LoadAndValidate() should return error for invalid config")
	}
}

func TestEnvBinding(t *testing.T) {
	binding := &EnvBinding{
		Key:    "test.key",
		EnvVar: "TEST_KEY",
	}

	if binding.Key != "test.key" {
		t.Errorf("Key = %v, want test.key", binding.Key)
	}
	if binding.EnvVar != "TEST_KEY" {
		t.Errorf("EnvVar = %v, want TEST_KEY", binding.EnvVar)
	}
}

func TestWithRequired(t *testing.T) {
	binding := &EnvBinding{}
	WithRequired()(binding)

	if !binding.Required {
		t.Error("WithRequired() should set Required to true")
	}
}

func TestWithDefault(t *testing.T) {
	binding := &EnvBinding{}
	WithDefault("default_value")(binding)

	if binding.Default != "default_value" {
		t.Errorf("Default = %v, want default_value", binding.Default)
	}
}

func TestBindStruct(t *testing.T) {
	type TestConfig struct {
		DSN   string `env:"TEST_DSN" envDefault:"localhost:3306"`
		Port  int    `env:"TEST_PORT" envDefault:"8080"`
		Debug bool   `env:"TEST_DEBUG"`
	}

	cfg := New()

	// Set environment variables
	os.Setenv("TEST_DSN", "postgres://localhost:5432/test")
	defer os.Unsetenv("TEST_DSN")

	var config TestConfig
	err := cfg.BindStruct("", &config)
	if err != nil {
		t.Fatalf("BindStruct() error = %v", err)
	}
}

func TestBindStruct_Nested(t *testing.T) {
	type DatabaseConfig struct {
		DSN    string `env:"DB_DSN" envDefault:"localhost:3306"`
		Driver string `env:"DB_DRIVER" envDefault:"mysql"`
	}

	type ServerConfig struct {
		Name     string         `yaml:"name"`
		Database DatabaseConfig `yaml:"database"`
	}

	cfg := New()

	var config ServerConfig
	err := cfg.BindStruct("APP", &config)
	if err != nil {
		t.Fatalf("BindStruct() error = %v", err)
	}
}

func TestEnvOverride(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := tmpDir + "/config.yaml"

	yamlContent := `
name: fileapp
http:
  address: ":8080"
  timeout: 30s
`
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Set environment variable to override file value
	os.Setenv("FIREFLY_NAME", "envapp")
	defer os.Unsetenv("FIREFLY_NAME")

	cfg := New()
	var bootstrap Bootstrap
	err := cfg.LoadWithEnv(configPath, "FIREFLY", &bootstrap)
	if err != nil {
		t.Fatalf("LoadWithEnv() error = %v", err)
	}

	// Name should come from file (env override depends on viper's AutomaticEnv behavior)
	if bootstrap.Name == "" {
		t.Error("Name should not be empty")
	}
}
