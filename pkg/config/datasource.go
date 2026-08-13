package config

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
)

// DataSourceCreatorFunc builds a RemoteSource from a parsed data source URL.
//
// The URL encodes the source location and options, e.g.:
//
//	file:///etc/app/config.yaml
//	consul://127.0.0.1:8500/config/my-service?token=xxx
//	etcd://127.0.0.1:2379/config/my-service   (register your own creator)
type DataSourceCreatorFunc func(ctx context.Context, u *url.URL) (RemoteSource, error)

// dataSourceRegistry maps URL schemes to their source creators.
// Sources register themselves via init() — scheme-driven, Jupiter style.
var dataSourceRegistry sync.Map // scheme string → DataSourceCreatorFunc

// RegisterSource registers a data source builder for the given URL scheme.
// Multiple sources can coexist (file, consul, etcd, apollo, ...).
func RegisterSource(scheme string, creator DataSourceCreatorFunc) {
	if scheme == "" || creator == nil {
		return
	}
	dataSourceRegistry.Store(scheme, creator)
}

// LoadFromDataSource loads configuration from a data source URL and starts
// watching it for changes.
//
// The scheme determines the source (see RegisterSource). Remote values are
// merged into the local config tree (remote wins on conflicting keys), and
// changes trigger the same OnChange callbacks as file hot-reload.
func (c *Config) LoadFromDataSource(ctx context.Context, dataSourceURL string, target any) error {
	u, err := url.Parse(dataSourceURL)
	if err != nil {
		return fmt.Errorf("config: invalid data source URL %q: %w", dataSourceURL, err)
	}

	v, ok := dataSourceRegistry.Load(u.Scheme)
	if !ok {
		return fmt.Errorf("config: unsupported data source scheme %q (registered: %s)",
			u.Scheme, registeredSchemes())
	}

	creator := v.(DataSourceCreatorFunc)
	src, err := creator(ctx, u)
	if err != nil {
		return fmt.Errorf("config: create data source %q: %w", u.Scheme, err)
	}

	if err := c.AttachRemote(ctx, src, target); err != nil {
		return fmt.Errorf("config: attach data source %q: %w", u.Scheme, err)
	}
	return nil
}

// registeredSchemes returns a sorted, comma-separated list of registered schemes.
func registeredSchemes() string {
	var schemes []string
	dataSourceRegistry.Range(func(k, _ any) bool {
		schemes = append(schemes, k.(string))
		return true
	})
	sort.Strings(schemes)
	return strings.Join(schemes, ", ")
}
