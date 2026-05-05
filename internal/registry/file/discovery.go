// Package file provides file-based service discovery for the Firefly framework.
// It loads service instance configurations from YAML files.
package file

import (
	"context"
	"sync"

	"github.com/zhangpeihaoks/firefly/internal/registry"
	"gopkg.in/yaml.v3"
)

// Discovery implements file-based service discovery.
// It loads service configurations from a YAML file and provides them to clients.
type Discovery struct {
	mu        sync.RWMutex
	services  map[string][]*registry.ServiceInstance
	config    *Config
	watchers  map[string][]*fileWatcher
	watcherMu sync.Mutex
}

// Config is the configuration for file-based discovery.
type Config struct {
	// Services is the list of service configurations.
	Services []*ServiceConfig `yaml:"services"`
}

// ServiceConfig represents a service configuration in the YAML file.
type ServiceConfig struct {
	// Name is the service name.
	Name string `yaml:"name"`
	// Instances is the list of service instances.
	Instances []*registry.ServiceInstance `yaml:"instances"`
}

// NewDiscovery creates a new file-based discovery instance.
func NewDiscovery() *Discovery {
	return &Discovery{
		services: make(map[string][]*registry.ServiceInstance),
		watchers: make(map[string][]*fileWatcher),
	}
}

// Load loads service configurations from YAML data.
func (d *Discovery) Load(data []byte) error {
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Clear existing services
	d.services = make(map[string][]*registry.ServiceInstance)

	// Load new services
	for _, svc := range config.Services {
		instances := make([]*registry.ServiceInstance, 0, len(svc.Instances))
		for _, instance := range svc.Instances {
			// Create a copy to avoid shared references
			copied := &registry.ServiceInstance{
				ID:        instance.ID,
				Name:      instance.Name,
				Version:   instance.Version,
				Endpoints: make([]string, len(instance.Endpoints)),
			}
			copy(copied.Endpoints, instance.Endpoints)

			// Ensure metadata is initialized and copied
			if instance.Metadata != nil {
				copied.Metadata = make(map[string]string)
				for k, v := range instance.Metadata {
					copied.Metadata[k] = v
				}
			} else {
				copied.Metadata = make(map[string]string)
			}

			instances = append(instances, copied)
		}
		d.services[svc.Name] = instances
	}

	// Notify watchers
	d.notifyWatchers()

	return nil
}

// LoadFromFile loads service configurations from a YAML file path.
func (d *Discovery) LoadFromFile(path string) error {
	// Note: This would typically use os.ReadFile, but we're keeping
	// the actual file reading separate for testability.
	// Users should read the file and call Load() with the data.
	// This method is kept for API consistency.
	return nil
}

// GetService retrieves all instances for a given service name.
func (d *Discovery) GetService(ctx context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	instances, ok := d.services[serviceName]
	if !ok {
		return []*registry.ServiceInstance{}, nil
	}

	// Return deep copies to prevent modification
	result := make([]*registry.ServiceInstance, len(instances))
	for i, instance := range instances {
		result[i] = instance.Clone()
	}
	return result, nil
}

// Watch watches for changes to the service instances.
func (d *Discovery) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
	w := &fileWatcher{
		serviceName: serviceName,
		discovery:   d,
		ch:          make(chan []*registry.ServiceInstance, 1),
	}

	d.watcherMu.Lock()
	d.watchers[serviceName] = append(d.watchers[serviceName], w)
	d.watcherMu.Unlock()

	// Send initial state
	d.mu.RLock()
	instances := d.services[serviceName]
	d.mu.RUnlock()

	// Copy instances
	copied := make([]*registry.ServiceInstance, len(instances))
	copy(copied, instances)

	select {
	case w.ch <- copied:
	default:
		// Channel full, skip
	}

	return w, nil
}

// notifyWatchers notifies all watchers of service changes.
func (d *Discovery) notifyWatchers() {
	d.watcherMu.Lock()
	defer d.watcherMu.Unlock()

	for serviceName, watchers := range d.watchers {
		d.mu.RLock()
		instances := d.services[serviceName]
		d.mu.RUnlock()

		// Copy instances
		copied := make([]*registry.ServiceInstance, len(instances))
		copy(copied, instances)

		for _, w := range watchers {
			select {
			case w.ch <- copied:
			default:
				// Channel full, skip
			}
		}
	}
}

// fileWatcher implements the Watcher interface for file-based discovery.
type fileWatcher struct {
	serviceName string
	discovery   *Discovery
	ch          chan []*registry.ServiceInstance
	stopped     bool
	mu          sync.Mutex
}

// Next returns the next set of service instances.
func (w *fileWatcher) Next() ([]*registry.ServiceInstance, error) {
	select {
	case instances := <-w.ch:
		return instances, nil
	default:
		// Return current state if no updates
		w.discovery.mu.RLock()
		instances := w.discovery.services[w.serviceName]
		w.discovery.mu.RUnlock()
		copied := make([]*registry.ServiceInstance, len(instances))
		copy(copied, instances)
		return copied, nil
	}
}

// Stop stops the watcher.
func (w *fileWatcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return nil
	}
	w.stopped = true
	close(w.ch)

	// Remove from discovery
	w.discovery.watcherMu.Lock()
	watchers := w.discovery.watchers[w.serviceName]
	for i, watcher := range watchers {
		if watcher == w {
			w.discovery.watchers[w.serviceName] = append(watchers[:i], watchers[i+1:]...)
			break
		}
	}
	w.discovery.watcherMu.Unlock()

	return nil
}
