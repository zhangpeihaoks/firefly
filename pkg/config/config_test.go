package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestLoadWithEnvOverride tests that environment variables override config file values.
func TestLoadWithEnvOverride(t *testing.T) {
	// Create a temporary config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	content := []byte("name: test-service\nversion: v1.0.0\nhttp:\n  address: ':8080'\n  timeout: 30s\n")
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	type HTTPConfig struct {
		Address string `yaml:"address"`
		Timeout string `yaml:"timeout"`
	}
	type Bootstrap struct {
		Name    string     `yaml:"name"`
		Version string     `yaml:"version"`
		HTTP    *HTTPConfig `yaml:"http"`
	}

	t.Run("env var overrides config file", func(t *testing.T) {
		os.Setenv("FIREFLY_HTTP_ADDRESS", ":9090")
		defer os.Unsetenv("FIREFLY_HTTP_ADDRESS")

		c := New(WithEnvPrefix("FIREFLY"))
		cfg := &Bootstrap{}
		if err := c.Load(configPath, cfg); err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		if cfg.Name != "test-service" {
			t.Errorf("expected name 'test-service', got %q", cfg.Name)
		}
		if cfg.HTTP.Address != ":9090" {
			t.Errorf("expected address ':9090' (env override), got %q", cfg.HTTP.Address)
		}
	})

	t.Run("no env var set uses config file value", func(t *testing.T) {
		c := New(WithEnvPrefix("FIREFLY"))
		cfg := &Bootstrap{}
		if err := c.Load(configPath, cfg); err != nil {
			t.Fatalf("Load failed: %v", err)
		}

		if cfg.HTTP.Address != ":8080" {
			t.Errorf("expected address ':8080' from config, got %q", cfg.HTTP.Address)
		}
	})
}

// TestWatchHotReload tests that Watch detects file changes and invokes callbacks.
func TestWatchHotReload(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	type Bootstrap struct {
		Name string `yaml:"name"`
		Port int    `yaml:"port"`
	}

	// Write initial config
	initialContent := []byte("name: initial\nport: 8080\n")
	if err := os.WriteFile(configPath, initialContent, 0644); err != nil {
		t.Fatal(err)
	}

	c := New()
	cfg := &Bootstrap{}
	if err := c.Load(configPath, cfg); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Name != "initial" {
		t.Fatalf("expected initial name 'initial', got %q", cfg.Name)
	}

	// Setup Watch with callback
	var mu sync.Mutex
	var callbackCfg *Bootstrap
	c.OnChange(func(v any) {
		mu.Lock()
		defer mu.Unlock()
		callbackCfg = v.(*Bootstrap)
	})

	c.Watch(cfg)

	// Modify the config file (viper needs a moment to start watching)
	time.Sleep(50 * time.Millisecond)

	updatedContent := []byte("name: updated\nport: 9090\n")
	if err := os.WriteFile(configPath, updatedContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for viper to detect the change (fsnotify + viper polling)
	// This can take up to a few seconds depending on the OS
	deadline := time.Now().Add(10 * time.Second)
	for {
		mu.Lock()
		got := callbackCfg
		mu.Unlock()
		if got != nil && got.Name == "updated" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for config reload callback")
		}
		time.Sleep(100 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if callbackCfg.Name != "updated" {
		t.Errorf("expected name 'updated', got %q", callbackCfg.Name)
	}
	if callbackCfg.Port != 9090 {
		t.Errorf("expected port 9090, got %d", callbackCfg.Port)
	}
}
