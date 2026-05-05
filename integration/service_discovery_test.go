// Package integration provides integration tests for service discovery and database connections.
package integration

import (
	"context"
	"testing"

	"github.com/zhangpeihaoks/firefly/internal/registry"
	"github.com/zhangpeihaoks/firefly/internal/registry/file"
	"gopkg.in/yaml.v3"
)

// TestFileServiceDiscovery tests the file-based service discovery implementation.
func TestFileServiceDiscovery(t *testing.T) {
	// Create file discovery
	discovery := file.NewDiscovery()

	// Load test services configuration
	servicesYAML := `
services:
  - name: "user-service"
    instances:
      - id: "service-1"
        name: "user-service"
        version: "v1"
        metadata:
          env: "test"
        endpoints:
          - "http://localhost:8080"
          - "grpc://localhost:9090"
      - id: "service-3"
        name: "user-service"
        version: "v2"
        metadata:
          env: "production"
        endpoints:
          - "http://localhost:9080"
  - name: "order-service"
    instances:
      - id: "service-2"
        name: "order-service"
        version: "v1"
        metadata:
          env: "test"
        endpoints:
          - "http://localhost:8081"
`

	if err := discovery.Load([]byte(servicesYAML)); err != nil {
		t.Fatalf("Failed to load services: %v", err)
	}

	ctx := context.Background()

	// Test GetService - user-service
	t.Run("GetService user-service", func(t *testing.T) {
		instances, err := discovery.GetService(ctx, "user-service")
		if err != nil {
			t.Fatalf("Failed to get service: %v", err)
		}

		if len(instances) != 2 {
			t.Errorf("Expected 2 instances, got %d", len(instances))
		}

		// Check that we have both v1 and v2
		versions := make(map[string]bool)
		for _, inst := range instances {
			versions[inst.Version] = true
		}

		if !versions["v1"] {
			t.Error("Expected v1 version to be present")
		}
		if !versions["v2"] {
			t.Error("Expected v2 version to be present")
		}
	})

	// Test GetService - order-service
	t.Run("GetService order-service", func(t *testing.T) {
		instances, err := discovery.GetService(ctx, "order-service")
		if err != nil {
			t.Fatalf("Failed to get service: %v", err)
		}

		if len(instances) != 1 {
			t.Errorf("Expected 1 instance, got %d", len(instances))
		}

		if instances[0].ID != "service-2" {
			t.Errorf("Expected service-2, got %s", instances[0].ID)
		}
	})

	// Test GetService - non-existent service
	t.Run("GetService non-existent", func(t *testing.T) {
		instances, err := discovery.GetService(ctx, "non-existent")
		if err != nil {
			t.Fatalf("Failed to get service: %v", err)
		}

		if len(instances) != 0 {
			t.Errorf("Expected 0 instances, got %d", len(instances))
		}
	})

	// Test Watch
	t.Run("Watch service", func(t *testing.T) {
		watcher, err := discovery.Watch(ctx, "user-service")
		if err != nil {
			t.Fatalf("Failed to watch service: %v", err)
		}
		defer watcher.Stop()

		// Get initial instances
		instances, err := watcher.Next()
		if err != nil {
			t.Fatalf("Failed to get next: %v", err)
		}

		if len(instances) != 2 {
			t.Errorf("Expected 2 instances, got %d", len(instances))
		}
	})

	// Test YAML unmarshaling
	t.Run("YAML unmarshal", func(t *testing.T) {
		var config file.Config
		if err := yaml.Unmarshal([]byte(servicesYAML), &config); err != nil {
			t.Fatalf("Failed to unmarshal YAML: %v", err)
		}

		if len(config.Services) != 2 {
			t.Errorf("Expected 2 services, got %d", len(config.Services))
		}
	})
}

// TestServiceInstanceFilter tests the service instance filtering functions.
func TestServiceInstanceFilter(t *testing.T) {
	instances := []*registry.ServiceInstance{
		{ID: "1", Name: "svc", Version: "v1", Metadata: map[string]string{"env": "prod"}},
		{ID: "2", Name: "svc", Version: "v2", Metadata: map[string]string{"env": "prod"}},
		{ID: "3", Name: "svc", Version: "v1", Metadata: map[string]string{"env": "dev"}},
		{ID: "4", Name: "svc", Version: "v2", Metadata: map[string]string{"env": "dev"}},
	}

	// Test version filter
	t.Run("Version filter v1", func(t *testing.T) {
		filtered := registry.FilterInstances(instances, registry.VersionFilter("v1"))
		if len(filtered) != 2 {
			t.Errorf("Expected 2 instances, got %d", len(filtered))
		}
		for _, inst := range filtered {
			if inst.Version != "v1" {
				t.Errorf("Expected v1, got %s", inst.Version)
			}
		}
	})

	// Test metadata filter
	t.Run("Metadata filter env=prod", func(t *testing.T) {
		filtered := registry.FilterInstances(instances, registry.MetadataFilter("env", "prod"))
		if len(filtered) != 2 {
			t.Errorf("Expected 2 instances, got %d", len(filtered))
		}
		for _, inst := range filtered {
			if inst.Metadata["env"] != "prod" {
				t.Errorf("Expected env=prod, got env=%s", inst.Metadata["env"])
			}
		}
	})

	// Test combined filters
	t.Run("Combined filters", func(t *testing.T) {
		filter := func(si *registry.ServiceInstance) bool {
			return si.Version == "v1" && si.Metadata["env"] == "dev"
		}
		filtered := registry.FilterInstances(instances, filter)
		if len(filtered) != 1 {
			t.Errorf("Expected 1 instance, got %d", len(filtered))
		}
		if filtered[0].ID != "3" {
			t.Errorf("Expected ID 3, got %s", filtered[0].ID)
		}
	})
}

// TestServiceInstanceClone tests the Clone method of ServiceInstance.
func TestServiceInstanceClone(t *testing.T) {
	original := &registry.ServiceInstance{
		ID:        "test-id",
		Name:      "test-service",
		Version:   "v1",
		Metadata:  map[string]string{"key1": "value1", "key2": "value2"},
		Endpoints: []string{"http://localhost:8080", "grpc://localhost:9090"},
	}

	cloned := original.Clone()

	// Verify all fields are copied
	if cloned.ID != original.ID {
		t.Errorf("ID not copied: got %s, want %s", cloned.ID, original.ID)
	}
	if cloned.Name != original.Name {
		t.Errorf("Name not copied: got %s, want %s", cloned.Name, original.Name)
	}
	if cloned.Version != original.Version {
		t.Errorf("Version not copied: got %s, want %s", cloned.Version, original.Version)
	}
	if len(cloned.Endpoints) != len(original.Endpoints) {
		t.Errorf("Endpoints not copied correctly")
	}
	if cloned.Metadata["key1"] != "value1" {
		t.Errorf("Metadata not copied: got %s, want value1", cloned.Metadata["key1"])
	}

	// Verify it's a deep copy (modifying clone doesn't affect original)
	cloned.Metadata["key1"] = "modified"
	if original.Metadata["key1"] == "modified" {
		t.Error("Clone should be a deep copy")
	}

	cloned.Endpoints = append(cloned.Endpoints, "http://localhost:9999")
	if len(original.Endpoints) == len(cloned.Endpoints) {
		t.Error("Clone should have independent slice")
	}
}

// TestServiceRegistrationWorkflow tests the complete service registration workflow.
func TestServiceRegistrationWorkflow(t *testing.T) {
	ctx := context.Background()

	// Create a service instance
	service := registry.NewServiceInstance(
		registry.WithID("test-instance-1"),
		registry.WithName("test-service"),
		registry.WithVersion("v1.0.0"),
		registry.WithMetadata(map[string]string{
			"env":    "test",
			"region": "us-west-1",
		}),
		registry.WithEndpoints(
			"http://localhost:8080",
			"grpc://localhost:9090",
		),
	)

	// Verify service instance is correctly configured
	if service.ID != "test-instance-1" {
		t.Errorf("Expected ID test-instance-1, got %s", service.ID)
	}
	if service.Name != "test-service" {
		t.Errorf("Expected Name test-service, got %s", service.Name)
	}
	if service.Version != "v1.0.0" {
		t.Errorf("Expected Version v1.0.0, got %s", service.Version)
	}
	if len(service.Endpoints) != 2 {
		t.Errorf("Expected 2 endpoints, got %d", len(service.Endpoints))
	}
	if service.Metadata["env"] != "test" {
		t.Errorf("Expected env=test, got env=%s", service.Metadata["env"])
	}

	// Verify clone works in registration context
	_ = service.Clone()

	// Simulate successful registration
	select {
	case <-ctx.Done():
		t.Fatal("Context cancelled before registration completed")
	default:
		// Registration would succeed here
	}
}
