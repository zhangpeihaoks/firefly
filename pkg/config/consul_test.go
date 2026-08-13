package config

import (
	"context"
	"sync"
	"testing"
	"time"
)

// mockKVClient simulates Consul KV with an incrementing index per call,
// so every List acts like a blocking query that returns a change.
type mockKVClient struct {
	mu    sync.Mutex
	pairs []*KVPair
	index uint64
}

func (m *mockKVClient) List(_ context.Context, _ string, _ *KVQueryOptions) ([]*KVPair, uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.index++
	return m.pairs, m.index, nil
}

func (m *mockKVClient) setPairs(pairs []*KVPair) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pairs = pairs
}

func TestKVsToNestedMap(t *testing.T) {
	pairs := []*KVPair{
		{Key: "config", Value: []byte("ignored-node-value")}, // prefix node itself, skipped
		{Key: "config/name", Value: []byte("my-service")},
		{Key: "config/database/host", Value: []byte("db.example.com")},
		{Key: "config/database/port", Value: []byte("5432")},
		{Key: "config/features", Value: []byte(`{"tracing":true,"metrics":false}`)},
	}

	got := kvsToNestedMap(pairs, "config")

	if got["name"] != "my-service" {
		t.Errorf("expected name 'my-service', got %v", got["name"])
	}

	db, ok := got["database"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested database map, got %#v", got["database"])
	}
	if db["host"] != "db.example.com" {
		t.Errorf("expected db host, got %v", db["host"])
	}
	// "5432" is valid JSON, so it parses as a number (Viper converts on demand).
	if db["port"] != float64(5432) {
		t.Errorf("expected port 5432 (float64), got %v (%T)", db["port"], db["port"])
	}

	features, ok := got["features"].(map[string]any)
	if !ok {
		t.Fatalf("expected JSON parsed features map, got %#v", got["features"])
	}
	if features["tracing"] != true {
		t.Errorf("expected features.tracing=true, got %v", features["tracing"])
	}
}

func TestConsulSource_Watch(t *testing.T) {
	kv := &mockKVClient{}
	src := &ConsulSource{kv: kv, prefix: "config", watchTime: 100 * time.Millisecond}

	kv.setPairs([]*KVPair{
		{Key: "config/name", Value: []byte("svc-a")},
	})

	ctx := context.Background()

	// First Watch returns immediately with current state.
	data, err := src.Watch(ctx)
	if err != nil {
		t.Fatalf("first watch failed: %v", err)
	}
	if data["name"] != "svc-a" {
		t.Errorf("expected name 'svc-a', got %v", data["name"])
	}

	// Second Watch sees a change (mock increments index every call).
	kv.setPairs([]*KVPair{
		{Key: "config/name", Value: []byte("svc-b")},
	})
	data, err = src.Watch(ctx)
	if err != nil {
		t.Fatalf("second watch failed: %v", err)
	}
	if data["name"] != "svc-b" {
		t.Errorf("expected name 'svc-b' after change, got %v", data["name"])
	}
}

func TestConfig_AttachRemote(t *testing.T) {
	type Bootstrap struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	}

	c := New()
	cfg := &Bootstrap{}
	if err := c.LoadFromYAML([]byte("name: local-service\nversion: v1.0.0\n"), cfg); err != nil {
		t.Fatalf("LoadFromYAML failed: %v", err)
	}
	if cfg.Name != "local-service" {
		t.Fatalf("expected local name before attach, got %q", cfg.Name)
	}

	kv := &mockKVClient{}
	src := &ConsulSource{kv: kv, prefix: "config", watchTime: 50 * time.Millisecond}
	kv.setPairs([]*KVPair{
		{Key: "config/name", Value: []byte("remote-service")},
	})

	// Track callbacks.
	cbCh := make(chan struct{}, 10)
	c.OnChange(func(any) {
		select {
		case cbCh <- struct{}{}:
		default:
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := c.AttachRemote(ctx, src, cfg); err != nil {
		t.Fatalf("AttachRemote failed: %v", err)
	}

	// Initial merge should have applied and fired a callback.
	select {
	case <-cbCh:
	case <-time.After(2 * time.Second):
		t.Fatal("expected callback after initial remote load")
	}
	if cfg.Name != "remote-service" {
		t.Errorf("expected remote override name='remote-service', got %q", cfg.Name)
	}

	// A remote change should re-merge and fire another callback.
	kv.setPairs([]*KVPair{
		{Key: "config/name", Value: []byte("remote-v2")},
		{Key: "config/version", Value: []byte("v2.0.0")},
	})
	select {
	case <-cbCh:
	case <-time.After(2 * time.Second):
		t.Fatal("expected callback after remote change")
	}
	if cfg.Name != "remote-v2" {
		t.Errorf("expected name 'remote-v2' after change, got %q", cfg.Name)
	}
	if cfg.Version != "v2.0.0" {
		t.Errorf("expected version 'v2.0.0', got %q", cfg.Version)
	}
}
