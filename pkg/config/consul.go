package config

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	consulapi "github.com/hashicorp/consul/api"
)

// ConsulSource is a RemoteSource backed by Consul KV.
//
// All keys under Prefix are treated as configuration. Key path segments
// become nested config keys: "config/database/host" maps to database.host.
// Value bytes are parsed as JSON when possible, otherwise kept as strings.
type ConsulSource struct {
	kv        KVClient
	prefix    string
	watchTime time.Duration

	mu        sync.Mutex
	lastIndex uint64
}

// ConsulSourceConfig configures the Consul KV configuration source.
type ConsulSourceConfig struct {
	// Address is the Consul agent address (e.g., localhost:8500).
	Address string
	// Token is the ACL token for authentication (optional).
	Token string
	// Namespace is the Consul namespace (optional, Enterprise only).
	Namespace string
	// Partition is the Consul partition (optional, Enterprise only).
	Partition string
	// Prefix is the KV prefix holding configuration (e.g., "config/my-service").
	// Defaults to "config".
	Prefix string
	// WatchTime is the blocking query timeout. Defaults to 10s.
	WatchTime time.Duration
}

// KVClient abstracts the Consul KV API for testability.
type KVClient interface {
	// List returns all KV pairs under prefix. Blocking queries are supported:
	// pass a non-zero WaitIndex to block server-side until a change occurs.
	List(ctx context.Context, prefix string, opts *KVQueryOptions) ([]*KVPair, uint64, error)
}

// KVQueryOptions are the options for KVClient.List.
type KVQueryOptions struct {
	// WaitIndex blocks until the KV index is greater than this value.
	WaitIndex uint64
	// WaitTime is the maximum time to block.
	WaitTime time.Duration
}

// KVPair is a single key-value pair from Consul KV.
type KVPair struct {
	Key   string
	Value []byte
}

// NewConsulSource creates a Consul KV configuration source.
func NewConsulSource(cfg ConsulSourceConfig) (*ConsulSource, error) {
	apiConfig := consulapi.DefaultConfig()
	apiConfig.Address = cfg.Address
	apiConfig.Token = cfg.Token
	if cfg.Namespace != "" {
		apiConfig.Namespace = cfg.Namespace
	}
	if cfg.Partition != "" {
		apiConfig.Partition = cfg.Partition
	}

	client, err := consulapi.NewClient(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("consul config: failed to create client: %w", err)
	}

	watchTime := cfg.WatchTime
	if watchTime == 0 {
		watchTime = 10 * time.Second
	}
	prefix := strings.TrimSuffix(cfg.Prefix, "/")
	if prefix == "" {
		prefix = "config"
	}

	return &ConsulSource{
		kv:        &consulKVClient{kv: client.KV()},
		prefix:    prefix,
		watchTime: watchTime,
	}, nil
}

// Name returns the source name.
func (s *ConsulSource) Name() string {
	return "consul-kv:" + s.prefix
}

// Watch performs a blocking KV list query and returns the current configuration.
// The first call returns immediately; subsequent calls block until the config
// changes or watchTime elapses.
func (s *ConsulSource) Watch(ctx context.Context) (map[string]any, error) {
	s.mu.Lock()
	waitIndex := s.lastIndex
	s.mu.Unlock()

	opts := &KVQueryOptions{WaitIndex: waitIndex, WaitTime: s.watchTime}
	pairs, index, err := s.kv.List(ctx, s.prefix, opts)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("consul config: failed to list kv %q: %w", s.prefix, err)
	}

	s.mu.Lock()
	s.lastIndex = index
	s.mu.Unlock()

	return kvsToNestedMap(pairs, s.prefix), nil
}

// consulKVClient adapts the official Consul KV API to the KVClient interface.
type consulKVClient struct {
	kv *consulapi.KV
}

func (c *consulKVClient) List(ctx context.Context, prefix string, opts *KVQueryOptions) ([]*KVPair, uint64, error) {
	q := (&consulapi.QueryOptions{
		WaitIndex: opts.WaitIndex,
		WaitTime:  opts.WaitTime,
	}).WithContext(ctx)

	pairs, meta, err := c.kv.List(prefix, q)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*KVPair, 0, len(pairs))
	for _, p := range pairs {
		result = append(result, &KVPair{Key: p.Key, Value: p.Value})
	}
	return result, meta.LastIndex, nil
}

// kvsToNestedMap converts flat KV pairs into a nested configuration map.
// KV keys use "/" as the path separator; the pair whose key equals the prefix
// itself (a value stored on the prefix node) is skipped.
func kvsToNestedMap(pairs []*KVPair, prefix string) map[string]any {
	result := make(map[string]any)
	for _, p := range pairs {
		rel := strings.TrimPrefix(p.Key, prefix)
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			continue
		}
		parts := strings.Split(rel, "/")
		insertNested(result, parts, parseKVValue(p.Value))
	}
	return result
}

// parseKVValue parses KV value bytes as JSON when possible, else as a string.
func parseKVValue(v []byte) any {
	trimmed := bytes.TrimSpace(v)
	if len(trimmed) == 0 {
		return ""
	}
	var parsed any
	if err := json.Unmarshal(trimmed, &parsed); err == nil {
		return parsed
	}
	return string(v)
}

// insertNested inserts a value into a nested map at the given key path.
func insertNested(m map[string]any, parts []string, value any) {
	if len(parts) == 1 {
		m[parts[0]] = value
		return
	}
	child, ok := m[parts[0]].(map[string]any)
	if !ok {
		child = make(map[string]any)
		m[parts[0]] = child
	}
	insertNested(child, parts[1:], value)
}
