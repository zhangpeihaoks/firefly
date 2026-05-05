// Package registry provides service registration and discovery abstractions for the Firefly framework.
// This file defines the Discovery and Watcher interfaces for service discovery.
package registry

import (
	"context"
)

// Discovery is the service discovery interface.
// It defines the contract for discovering service instances from a service registry.
type Discovery interface {
	// GetService retrieves all instances for a given service name.
	// Returns an empty list if no instances are found.
	GetService(ctx context.Context, serviceName string) ([]*ServiceInstance, error)
	// Watch watches for changes to the service instances.
	// It returns a Watcher that yields the updated service instances.
	Watch(ctx context.Context, serviceName string) (Watcher, error)
}

// Watcher is the service watcher interface.
// It yields service instance updates as they occur in the registry.
type Watcher interface {
	// Next returns the next set of service instances.
	// It blocks until new instances are available or an error occurs.
	Next() ([]*ServiceInstance, error)
	// Stop stops the watcher and releases any resources.
	Stop() error
}

// ServiceInstanceFilter is a function that filters service instances.
// It returns true if the instance should be included, false otherwise.
type ServiceInstanceFilter func(*ServiceInstance) bool

// FilterInstances filters a list of service instances using the given filter function.
func FilterInstances(instances []*ServiceInstance, filter ServiceInstanceFilter) []*ServiceInstance {
	if filter == nil {
		return instances
	}
	filtered := make([]*ServiceInstance, 0, len(instances))
	for _, instance := range instances {
		if filter(instance) {
			filtered = append(filtered, instance)
		}
	}
	return filtered
}

// VersionFilter creates a filter that matches instances with the given version.
func VersionFilter(version string) ServiceInstanceFilter {
	return func(si *ServiceInstance) bool {
		return si.Version == version
	}
}

// MetadataFilter creates a filter that matches instances with the given metadata key-value pair.
func MetadataFilter(key, value string) ServiceInstanceFilter {
	return func(si *ServiceInstance) bool {
		if si.Metadata == nil {
			return false
		}
		return si.Metadata[key] == value
	}
}
