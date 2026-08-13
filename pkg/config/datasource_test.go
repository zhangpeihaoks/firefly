package config

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// mockRemoteSource returns a fixed map on every Watch call.
type mockRemoteSource struct {
	name string
	data map[string]any
}

func (m *mockRemoteSource) Name() string { return m.name }

func (m *mockRemoteSource) Watch(context.Context) (map[string]any, error) {
	return m.data, nil
}

// TestLoadFromDataSource_RegisteredScheme verifies URL-driven source dispatch
// and remote-overrides-local merging.
func TestLoadFromDataSource_RegisteredScheme(t *testing.T) {
	// Register a mock scheme.
	RegisterSource("mock", func(_ context.Context, u *url.URL) (RemoteSource, error) {
		if u.Host == "" {
			return nil, os.ErrInvalid
		}
		return &mockRemoteSource{
			name: "mock:" + u.Host,
			data: map[string]any{
				"name":    "remote-service",
				"version": "v2.0.0",
			},
		}, nil
	})

	type Bootstrap struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	}

	c := New()
	cfg := &Bootstrap{}
	if err := c.LoadFromYAML([]byte("name: local-service\nversion: v1.0.0\n"), cfg); err != nil {
		t.Fatalf("LoadFromYAML failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := c.LoadFromDataSource(ctx, "mock://config-server", cfg); err != nil {
		t.Fatalf("LoadFromDataSource failed: %v", err)
	}

	// Remote values override local file values.
	if cfg.Name != "remote-service" {
		t.Errorf("expected remote override name, got %q", cfg.Name)
	}
	if cfg.Version != "v2.0.0" {
		t.Errorf("expected remote version, got %q", cfg.Version)
	}
}

func TestLoadFromDataSource_UnsupportedScheme(t *testing.T) {
	c := New()
	cfg := &struct{}{}
	err := c.LoadFromDataSource(context.Background(), "unknown://x", cfg)
	if err == nil {
		t.Fatal("expected error for unsupported scheme")
	}
}

func TestLoadFromDataSource_InvalidURL(t *testing.T) {
	c := New()
	cfg := &struct{}{}
	err := c.LoadFromDataSource(context.Background(), "://bad-url", cfg)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

// TestFileSource_Watch verifies first-call-returns-immediately and
// change-driven updates.
func TestFileSource_Watch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("name: v1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	src := NewFileSource(path)
	ctx := context.Background()

	// First call returns immediately.
	data, err := src.Watch(ctx)
	if err != nil {
		t.Fatalf("first watch failed: %v", err)
	}
	if data["name"] != "v1" {
		t.Errorf("expected name v1, got %v", data["name"])
	}

	// Second call blocks until the file changes.
	changed := make(chan struct{})
	go func() {
		time.Sleep(100 * time.Millisecond)
		os.WriteFile(path, []byte("name: v2\n"), 0644)
		close(changed)
	}()

	data, err = src.Watch(ctx)
	if err != nil {
		t.Fatalf("change watch failed: %v", err)
	}
	if data["name"] != "v2" {
		t.Errorf("expected name v2 after change, got %v", data["name"])
	}
	<-changed
}

// TestFileSource_Cancel verifies Watch returns on context cancellation.
func TestFileSource_Cancel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("name: v1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	src := NewFileSource(path)
	ctx, cancel := context.WithCancel(context.Background())

	// First call (initiate).
	if _, err := src.Watch(ctx); err != nil {
		t.Fatalf("first watch failed: %v", err)
	}

	// Second call should unblock on cancel.
	done := make(chan error, 1)
	go func() {
		_, err := src.Watch(ctx)
		done <- err
	}()
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected context error after cancel, got nil")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Watch did not return after context cancellation")
	}
}

// TestConsulSource_FromURL verifies consul:// URL parsing.
func TestConsulSource_FromURL(t *testing.T) {
	src, err := newConsulSourceFromURL(context.Background(), &url.URL{
		Scheme:   "consul",
		Host:     "127.0.0.1:8500",
		Path:     "/config/my-service",
		RawQuery: "token=abc&watch_time=5s",
	})
	if err != nil {
		t.Fatalf("failed to build consul source: %v", err)
	}

	cs, ok := src.(*ConsulSource)
	if !ok {
		t.Fatalf("expected *ConsulSource, got %T", src)
	}
	if cs.prefix != "config/my-service" {
		t.Errorf("expected prefix 'config/my-service', got %q", cs.prefix)
	}
	if cs.watchTime != 5*time.Second {
		t.Errorf("expected watch time 5s, got %v", cs.watchTime)
	}
	if src.Name() != "consul-kv:config/my-service" {
		t.Errorf("unexpected source name %q", src.Name())
	}
}

func TestParseDurationQuery(t *testing.T) {
	if d, err := parseDurationQuery("", 10*time.Second); err != nil || d != 10*time.Second {
		t.Errorf("empty should use default, got %v err=%v", d, err)
	}
	if d, err := parseDurationQuery("5", 10*time.Second); err != nil || d != 5*time.Second {
		t.Errorf("numeric seconds, got %v err=%v", d, err)
	}
	if d, err := parseDurationQuery("500ms", 10*time.Second); err != nil || d != 500*time.Millisecond {
		t.Errorf("duration string, got %v err=%v", d, err)
	}
	if _, err := parseDurationQuery("bad", 10*time.Second); err == nil {
		t.Error("expected error for invalid duration")
	}
}
