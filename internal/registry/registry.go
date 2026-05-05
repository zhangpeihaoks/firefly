// Package registry provides service registration abstractions for the Firefly framework.
// It defines the Registrar interface for registering and deregistering service instances.
package registry

import (
	"context"
)

// Registrar is the service registration interface.
// It defines the contract for registering and deregistering service instances
// with a service registry (e.g., Consul, Etcd, Nacos).
type Registrar interface {
	// Register registers a service instance with the registry.
	// It should be called when the service starts.
	Register(ctx context.Context, service *ServiceInstance) error
	// Deregister deregisters a service instance from the registry.
	// It should be called when the service stops.
	Deregister(ctx context.Context, service *ServiceInstance) error
}

// ServiceInstance represents a service instance in the registry.
// It contains all the information needed to identify and connect to a service.
type ServiceInstance struct {
	// ID is the unique identifier for this service instance.
	ID string `json:"id"`
	// Name is the service name.
	Name string `json:"name"`
	// Version is the service version.
	Version string `json:"version"`
	// Metadata contains additional service metadata.
	Metadata map[string]string `json:"metadata"`
	// Endpoints is the list of service endpoints (e.g., http://localhost:8080, grpc://localhost:9090).
	Endpoints []string `json:"endpoints"`
}

// Option is a function that configures a ServiceInstance.
type Option func(*ServiceInstance)

// NewServiceInstance creates a new ServiceInstance with the given options.
func NewServiceInstance(opts ...Option) *ServiceInstance {
	si := &ServiceInstance{
		Metadata:  make(map[string]string),
		Endpoints: []string{},
	}
	for _, opt := range opts {
		opt(si)
	}
	return si
}

// WithID sets the service instance ID.
func WithID(id string) Option {
	return func(si *ServiceInstance) {
		si.ID = id
	}
}

// WithName sets the service name.
func WithName(name string) Option {
	return func(si *ServiceInstance) {
		si.Name = name
	}
}

// WithVersion sets the service version.
func WithVersion(version string) Option {
	return func(si *ServiceInstance) {
		si.Version = version
	}
}

// WithMetadata sets the service metadata.
func WithMetadata(metadata map[string]string) Option {
	return func(si *ServiceInstance) {
		if si.Metadata == nil {
			si.Metadata = make(map[string]string)
		}
		for k, v := range metadata {
			si.Metadata[k] = v
		}
	}
}

// WithEndpoints sets the service endpoints.
func WithEndpoints(endpoints ...string) Option {
	return func(si *ServiceInstance) {
		si.Endpoints = append(si.Endpoints, endpoints...)
	}
}

// Clone creates a copy of the ServiceInstance.
func (si *ServiceInstance) Clone() *ServiceInstance {
	cloned := &ServiceInstance{
		ID:        si.ID,
		Name:      si.Name,
		Version:   si.Version,
		Metadata:  make(map[string]string),
		Endpoints: make([]string, len(si.Endpoints)),
	}
	copy(cloned.Endpoints, si.Endpoints)
	for k, v := range si.Metadata {
		cloned.Metadata[k] = v
	}
	return cloned
}
