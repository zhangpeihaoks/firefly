// Package consul provides Consul-based service registration and discovery for the Firefly framework.
package consul

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/zhangpeihaoks/firefly/internal/registry"
)

// Registrar implements the registry.Registrar interface for Consul.
type Registrar struct {
	client    Client
	logger    *slog.Logger
	config    *RegistrarConfig
	mu        sync.Mutex
	instances map[string]bool // Track registered instances
}

// RegistrarConfig is the configuration for Consul registrar.
type RegistrarConfig struct {
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
}

// Client is the interface for Consul client operations.
// This allows for mocking in tests.
type Client interface {
	// Register registers a service instance.
	Register(ctx context.Context, instance *consulServiceInstance) error
	// Deregister deregisters a service instance.
	Deregister(ctx context.Context, serviceID string) error
}

// consulServiceInstance represents a service instance in Consul format.
type consulServiceInstance struct {
	ID      string
	Name    string
	Address string
	Port    int
	Tags    []string
	Meta    map[string]string
	Check   *consulCheck
	Checks  []*consulCheck
}

// consulCheck represents a health check configuration.
type consulCheck struct {
	CheckID                        string
	Name                           string
	Interval                       time.Duration
	Timeout                        time.Duration
	HTTP                           string
	TCP                            string
	TTL                            time.Duration
	DeregisterCriticalServiceAfter time.Duration
}

// NewRegistrar creates a new Consul registrar.
func NewRegistrar(config *RegistrarConfig, opts ...RegistrarOption) *Registrar {
	r := &Registrar{
		config:    config,
		logger:    slog.Default(),
		instances: make(map[string]bool),
	}

	// Initialize default client (can be overridden by options)
	r.client = &mockClient{}

	// Apply options (may override default client)
	for _, opt := range opts {
		opt(r)
	}

	return r
}

// RegistrarOption is a function that configures the Registrar.
type RegistrarOption func(*Registrar)

// WithLogger sets the logger for the registrar.
func WithLogger(logger *slog.Logger) RegistrarOption {
	return func(r *Registrar) {
		r.logger = logger
	}
}

// WithClient sets a custom Consul client.
func WithClient(client Client) RegistrarOption {
	return func(r *Registrar) {
		r.client = client
	}
}

// Register registers a service instance with Consul.
func (r *Registrar) Register(ctx context.Context, service *registry.ServiceInstance) error {
	if service == nil {
		return fmt.Errorf("service instance cannot be nil")
	}
	if service.ID == "" {
		return fmt.Errorf("service ID cannot be empty")
	}
	if service.Name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Convert to Consul format
	consulInstance, err := r.toConsulInstance(service)
	if err != nil {
		return fmt.Errorf("failed to convert service instance: %w", err)
	}

	// Register with Consul
	if err := r.client.Register(ctx, consulInstance); err != nil {
		return fmt.Errorf("failed to register service %s: %w", service.Name, err)
	}

	r.instances[service.ID] = true
	r.logger.Info("service registered",
		"service_id", service.ID,
		"service_name", service.Name,
		"endpoints", service.Endpoints,
	)

	return nil
}

// Deregister deregisters a service instance from Consul.
func (r *Registrar) Deregister(ctx context.Context, service *registry.ServiceInstance) error {
	if service == nil {
		return fmt.Errorf("service instance cannot be nil")
	}
	if service.ID == "" {
		return fmt.Errorf("service ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if registered
	if !r.instances[service.ID] {
		return nil
	}

	// Deregister from Consul
	if err := r.client.Deregister(ctx, service.ID); err != nil {
		return fmt.Errorf("failed to deregister service %s: %w", service.Name, err)
	}

	delete(r.instances, service.ID)
	r.logger.Info("service deregistered",
		"service_id", service.ID,
		"service_name", service.Name,
	)

	return nil
}

// toConsulInstance converts a ServiceInstance to Consul format.
func (r *Registrar) toConsulInstance(service *registry.ServiceInstance) (*consulServiceInstance, error) {
	instance := &consulServiceInstance{
		ID:      service.ID,
		Name:    service.Name,
		Address: "",
		Port:    0,
		Tags:    []string{},
		Meta:    make(map[string]string),
	}

	// Copy metadata
	for k, v := range service.Metadata {
		instance.Meta[k] = v
	}
	instance.Meta["version"] = service.Version

	// Parse endpoints to get address and port
	if len(service.Endpoints) > 0 {
		endpoint := service.Endpoints[0]
		u, err := url.Parse(endpoint)
		if err == nil {
			instance.Address = u.Hostname()
			if u.Port() != "" {
				var port int
				fmt.Sscanf(u.Port(), "%d", &port)
				instance.Port = port
			}
		} else {
			// Try host:port format
			parts := strings.Split(endpoint, ":")
			if len(parts) == 2 {
				instance.Address = parts[0]
				fmt.Sscanf(parts[1], "%d", &instance.Port)
			} else {
				instance.Address = endpoint
			}
		}
	}

	// Add default health check
	instance.Check = &consulCheck{
		CheckID:  "service:" + service.ID,
		Name:     "Service Health Check",
		Interval: 10 * time.Second,
		Timeout:  5 * time.Second,
	}

	return instance, nil
}

// mockClient is a mock implementation of the Consul client for demonstration.
type mockClient struct {
	services map[string]*consulServiceInstance
	mu       sync.RWMutex
}

func (m *mockClient) Register(ctx context.Context, instance *consulServiceInstance) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.services == nil {
		m.services = make(map[string]*consulServiceInstance)
	}
	m.services[instance.ID] = instance
	return nil
}

func (m *mockClient) Deregister(ctx context.Context, serviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.services != nil {
		delete(m.services, serviceID)
	}
	return nil
}
