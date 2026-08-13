// Package config provides configuration management for the Firefly framework.
// This file defines the remote configuration source abstraction (config center).
package config

import (
	"context"
	"log/slog"
	"reflect"
	"time"
)

// RemoteSource is the interface for remote configuration sources (config centers).
//
// Implementations pull configuration from an external store (Consul KV, etcd,
// Nacos, etc.) and merge it into the local config tree via Config.AttachRemote.
// Remote values override local file values for conflicting keys.
type RemoteSource interface {
	// Name returns the source name for logging and debugging.
	Name() string
	// Watch blocks until the remote configuration changes, then returns the
	// updated configuration as a nested map. The first call must return the
	// current configuration immediately (non-blocking).
	Watch(ctx context.Context) (map[string]any, error)
}

// remoteRetryDelay is the backoff between failed remote watch attempts.
const remoteRetryDelay = 5 * time.Second

// AttachRemote mounts a remote configuration source and starts watching it.
//
// The remote values are merged into the local config tree (remote wins on
// conflicting keys). On every remote change, target is re-unmarshaled and all
// OnChange callbacks are invoked — the same channel as file hot-reload.
// The watch loop runs until ctx is cancelled.
func (c *Config) AttachRemote(ctx context.Context, src RemoteSource, target any) error {
	// Initial pull: the first Watch call returns the current state immediately.
	initial, err := src.Watch(ctx)
	if err != nil {
		return &RemoteSourceError{Source: src.Name(), Err: err}
	}
	c.mergeRemote(initial, target)

	go c.watchRemote(ctx, src, target, initial)
	return nil
}

// watchRemote watches the remote source and re-merges on changes.
func (c *Config) watchRemote(ctx context.Context, src RemoteSource, target any, initial map[string]any) {
	last := initial
	for {
		if ctx.Err() != nil {
			return
		}

		data, err := src.Watch(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("config: remote watch failed",
				"source", src.Name(),
				"error", err,
			)
			select {
			case <-ctx.Done():
				return
			case <-time.After(remoteRetryDelay):
			}
			continue
		}

		if reflect.DeepEqual(data, last) {
			continue
		}
		last = data
		c.mergeRemote(data, target)
	}
}

// mergeRemote merges remote data into the config tree and notifies callbacks.
func (c *Config) mergeRemote(data map[string]any, target any) {
	if len(data) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.viper.MergeConfigMap(data); err != nil {
		slog.Error("config: failed to merge remote config", "error", err)
		return
	}
	if err := c.viper.Unmarshal(target); err != nil {
		slog.Error("config: failed to unmarshal after remote merge", "error", err)
		return
	}

	for _, fn := range c.callbacks {
		fn(target)
	}
}

// RemoteSourceError wraps an error from a remote configuration source.
type RemoteSourceError struct {
	Source string
	Err    error
}

func (e *RemoteSourceError) Error() string {
	return "config: remote source " + e.Source + " failed: " + e.Err.Error()
}

func (e *RemoteSourceError) Unwrap() error {
	return e.Err
}
