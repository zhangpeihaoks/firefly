// Package consul provides Consul-based service discovery for the Firefly framework.
package consul

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/registry"
)

// Discovery implements the registry.Discovery interface for Consul.
type Discovery struct {
	client   ServiceClient
	logger   *slog.Logger
	config   *DiscoveryConfig
	watchers map[string][]*consulWatcher
	mu       sync.RWMutex
}

// DiscoveryConfig is the configuration for Consul discovery.
type DiscoveryConfig struct {
	// Address is the Consul agent address (e.g., localhost:8500).
	Address string
	// Timeout is the timeout for Consul operations.
	Timeout time.Duration
	// Token is the ACL token for authentication (optional).
	Token string
	// Namespace is the Consul namespace (optional, Enterprise only).
	Namespace string
	// Partition is the Consul partition (optional, Enterprise only).
	Partition string
	// WatchTimeout is the timeout for watch operations (blocking queries).
	WatchTimeout time.Duration
	// PollInterval is the interval for polling when watch is not available.
	PollInterval time.Duration
}

// ServiceClient is the interface for Consul service discovery operations.
type ServiceClient interface {
	// GetService retrieves all instances for a service.
	GetService(ctx context.Context, serviceName string) ([]*consulServiceInstance, error)
	// Watch watches for service changes.
	Watch(ctx context.Context, serviceName string, lastIndex uint64) ([]*consulServiceInstance, uint64, error)
}

// NewDiscovery creates a new Consul discovery instance.
func NewDiscovery(config *DiscoveryConfig, opts ...DiscoveryOption) *Discovery {
	d := &Discovery{
		config:   config,
		logger:   slog.Default(),
		watchers: make(map[string][]*consulWatcher),
	}

	// Initialize default client (can be overridden by options)
	d.client = &mockServiceClient{}

	// Apply options (may override default client)
	for _, opt := range opts {
		opt(d)
	}

	return d
}

// DiscoveryOption is a function that configures the Discovery.
type DiscoveryOption func(*Discovery)

// WithDiscoveryLogger sets the logger for discovery.
func WithDiscoveryLogger(logger *slog.Logger) DiscoveryOption {
	return func(d *Discovery) {
		d.logger = logger
	}
}

// WithServiceClient sets a custom Consul service client.
func WithServiceClient(client ServiceClient) DiscoveryOption {
	return func(d *Discovery) {
		d.client = client
	}
}

// GetService retrieves all instances for a given service name.
func (d *Discovery) GetService(ctx context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	instances, err := d.client.GetService(ctx, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get service %s: %w", serviceName, err)
	}

	// Convert to registry format
	result := make([]*registry.ServiceInstance, 0, len(instances))
	for _, instance := range instances {
		result = append(result, d.toRegistryInstance(instance))
	}

	return result, nil
}

// Watch watches for changes to the service instances.
func (d *Discovery) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
	// Get initial instances
	instances, err := d.GetService(ctx, serviceName)
	if err != nil {
		d.logger.Warn("failed to get initial instances for watch",
			"service", serviceName,
			"error", err,
		)
		instances = []*registry.ServiceInstance{}
	}

	w := &consulWatcher{
		serviceName: serviceName,
		client:      d.client,
		logger:      d.logger,
		ch:          make(chan []*registry.ServiceInstance, 10),
		initial:     instances,
		config:      d.config,
	}

	// Start watching in background
	go w.watch(ctx)

	return w, nil
}

// toRegistryInstance converts a Consul service instance to registry format.
func (d *Discovery) toRegistryInstance(instance *consulServiceInstance) *registry.ServiceInstance {
	si := &registry.ServiceInstance{
		ID:        instance.ID,
		Name:      instance.Name,
		Version:   instance.Meta["version"],
		Metadata:  make(map[string]string),
		Endpoints: []string{},
	}

	// Copy metadata
	for k, v := range instance.Meta {
		si.Metadata[k] = v
	}

	// Build endpoint
	if instance.Address != "" {
		endpoint := fmt.Sprintf("http://%s", instance.Address)
		if instance.Port > 0 {
			endpoint = fmt.Sprintf("http://%s:%d", instance.Address, instance.Port)
		}
		si.Endpoints = append(si.Endpoints, endpoint)
	}

	return si
}

// consulWatcher implements the registry.Watcher interface for Consul.
type consulWatcher struct {
	serviceName string
	client      ServiceClient
	logger      *slog.Logger
	ch          chan []*registry.ServiceInstance
	initial     []*registry.ServiceInstance
	config      *DiscoveryConfig
	stopped     bool
	mu          sync.Mutex
	lastIndex   uint64
}

// Next returns the next set of service instances.
func (w *consulWatcher) Next() ([]*registry.ServiceInstance, error) {
	// Return initial state first if available
	if w.initial != nil {
		instances := w.initial
		w.initial = nil
		return instances, nil
	}

	select {
	case instances := <-w.ch:
		return instances, nil
	}
}

// Stop stops the watcher.
func (w *consulWatcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return nil
	}
	w.stopped = true
	close(w.ch)
	return nil
}

// watch starts watching for service changes.
func (w *consulWatcher) watch(ctx context.Context) {
	pollInterval := w.config.PollInterval
	if pollInterval == 0 {
		pollInterval = 10 * time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.mu.Lock()
			stopped := w.stopped
			w.mu.Unlock()

			if stopped {
				return
			}

			// Get updated instances
			instances, lastIndex, err := w.client.Watch(ctx, w.serviceName, w.lastIndex)
			if err != nil {
				w.logger.Warn("failed to watch service",
					"service", w.serviceName,
					"error", err,
				)
				continue
			}

			// Check if changed
			if lastIndex == w.lastIndex {
				continue
			}
			w.lastIndex = lastIndex

			// Convert to registry format
			result := make([]*registry.ServiceInstance, 0, len(instances))
			for _, instance := range instances {
				result = append(result, w.toRegistryInstance(instance))
			}

			// Send update
			select {
			case w.ch <- result:
			default:
				// Channel full, skip
			}
		}
	}
}

// toRegistryInstance converts a Consul service instance to registry format.
func (w *consulWatcher) toRegistryInstance(instance *consulServiceInstance) *registry.ServiceInstance {
	si := &registry.ServiceInstance{
		ID:        instance.ID,
		Name:      instance.Name,
		Version:   instance.Meta["version"],
		Metadata:  make(map[string]string),
		Endpoints: []string{},
	}

	// Copy metadata
	for k, v := range instance.Meta {
		si.Metadata[k] = v
	}

	// Build endpoint
	if instance.Address != "" {
		endpoint := fmt.Sprintf("http://%s", instance.Address)
		if instance.Port > 0 {
			endpoint = fmt.Sprintf("http://%s:%d", instance.Address, instance.Port)
		}
		si.Endpoints = append(si.Endpoints, endpoint)
	}

	return si
}

// mockServiceClient is a mock implementation of ServiceClient for demonstration.
type mockServiceClient struct {
	services map[string][]*consulServiceInstance
	mu       sync.RWMutex
	index    uint64
}

func (m *mockServiceClient) GetService(ctx context.Context, serviceName string) ([]*consulServiceInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.services == nil {
		return []*consulServiceInstance{}, nil
	}
	instances, ok := m.services[serviceName]
	if !ok {
		return []*consulServiceInstance{}, nil
	}
	return instances, nil
}

func (m *mockServiceClient) Watch(ctx context.Context, serviceName string, lastIndex uint64) ([]*consulServiceInstance, uint64, error) {
	m.mu.Lock()
	m.index++
	newIndex := m.index
	m.mu.Unlock()

	instances, err := m.GetService(ctx, serviceName)
	return instances, newIndex, err
}
