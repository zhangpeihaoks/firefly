package consul

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zhangpeihaoks/firefly/internal/registry"
)

func TestDiscovery_GetService(t *testing.T) {
	config := &DiscoveryConfig{
		Address:      "localhost:8500",
		Timeout:      5 * time.Second,
		PollInterval: 1 * time.Second,
	}

	d := NewDiscovery(config)
	ctx := context.Background()

	// Get service that doesn't exist (should return empty list)
	instances, err := d.GetService(ctx, "non-existent-service")
	require.NoError(t, err)
	assert.Empty(t, instances)
}

func TestDiscovery_GetService_WithMockClient(t *testing.T) {
	// Create mock client with pre-populated services
	mockClient := newTestServiceClientWithServices(map[string][]*consulServiceInstance{
		"user-service": {
			{
				ID:      "user-1",
				Name:    "user-service",
				Address: "192.168.1.1",
				Port:    8080,
				Meta:    map[string]string{"version": "v1.0.0", "env": "prod"},
			},
			{
				ID:      "user-2",
				Name:    "user-service",
				Address: "192.168.1.2",
				Port:    8080,
				Meta:    map[string]string{"version": "v1.0.0", "env": "prod"},
			},
		},
	})

	config := &DiscoveryConfig{
		Address:      "localhost:8500",
		Timeout:      5 * time.Second,
		PollInterval: 1 * time.Second,
	}

	d := NewDiscovery(config, WithServiceClient(mockClient))
	ctx := context.Background()

	instances, err := d.GetService(ctx, "user-service")
	require.NoError(t, err)
	assert.Len(t, instances, 2)

	// Verify conversion
	assert.Equal(t, "user-1", instances[0].ID)
	assert.Equal(t, "user-service", instances[0].Name)
	assert.Equal(t, "v1.0.0", instances[0].Version)
	assert.Contains(t, instances[0].Endpoints, "http://192.168.1.1:8080")
}

func TestDiscovery_Watch(t *testing.T) {
	mockClient := newTestServiceClientWithServices(map[string][]*consulServiceInstance{
		"test-service": {
			{
				ID:      "test-1",
				Name:    "test-service",
				Address: "localhost",
				Port:    8080,
				Meta:    map[string]string{"version": "v1.0.0"},
			},
		},
	})

	config := &DiscoveryConfig{
		Address:      "localhost:8500",
		Timeout:      5 * time.Second,
		PollInterval: 100 * time.Millisecond,
	}

	d := NewDiscovery(config, WithServiceClient(mockClient))
	ctx := context.Background()

	watcher, err := d.Watch(ctx, "test-service")
	require.NoError(t, err)
	defer watcher.Stop()

	// Get initial instances
	instances, err := watcher.Next()
	require.NoError(t, err)
	assert.NotEmpty(t, instances)
}

func TestDiscovery_Watch_Stop(t *testing.T) {
	mockClient := newTestServiceClientWithServices(map[string][]*consulServiceInstance{
		"test-service": {
			{
				ID:      "test-1",
				Name:    "test-service",
				Address: "localhost",
				Port:    8080,
				Meta:    map[string]string{"version": "v1.0.0"},
			},
		},
	})

	config := &DiscoveryConfig{
		Address:      "localhost:8500",
		Timeout:      5 * time.Second,
		PollInterval: 100 * time.Millisecond,
	}

	d := NewDiscovery(config, WithServiceClient(mockClient))
	ctx := context.Background()

	watcher, err := d.Watch(ctx, "test-service")
	require.NoError(t, err)

	// Stop should be idempotent
	require.NoError(t, watcher.Stop())
	require.NoError(t, watcher.Stop())
}

func TestToRegistryInstance(t *testing.T) {
	tests := []struct {
		name             string
		consulInstance   *consulServiceInstance
		expectedEndpoint string
		expectedVersion  string
	}{
		{
			name: "with port",
			consulInstance: &consulServiceInstance{
				ID:      "test-1",
				Name:    "test-service",
				Address: "192.168.1.1",
				Port:    8080,
				Meta:    map[string]string{"version": "v1.0.0", "env": "prod"},
			},
			expectedEndpoint: "http://192.168.1.1:8080",
			expectedVersion:  "v1.0.0",
		},
		{
			name: "without port",
			consulInstance: &consulServiceInstance{
				ID:      "test-2",
				Name:    "test-service",
				Address: "192.168.1.2",
				Port:    0,
				Meta:    map[string]string{"version": "v2.0.0"},
			},
			expectedEndpoint: "http://192.168.1.2",
			expectedVersion:  "v2.0.0",
		},
		{
			name: "no address",
			consulInstance: &consulServiceInstance{
				ID:      "test-3",
				Name:    "test-service",
				Address: "",
				Port:    8080,
				Meta:    map[string]string{"version": "v3.0.0"},
			},
			expectedEndpoint: "",
			expectedVersion:  "v3.0.0",
		},
	}

	config := &DiscoveryConfig{
		Address: "localhost:8500",
	}

	d := NewDiscovery(config)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := d.toRegistryInstance(tt.consulInstance)
			assert.Equal(t, tt.consulInstance.ID, instance.ID)
			assert.Equal(t, tt.consulInstance.Name, instance.Name)
			assert.Equal(t, tt.expectedVersion, instance.Version)

			if tt.expectedEndpoint != "" {
				assert.Contains(t, instance.Endpoints, tt.expectedEndpoint)
			} else {
				assert.Empty(t, instance.Endpoints)
			}

			// Verify metadata is copied
			assert.Equal(t, tt.consulInstance.Meta["env"], instance.Metadata["env"])
		})
	}
}

// testServiceClient is a test implementation of ServiceClient
type testServiceClient struct {
	services map[string][]*consulServiceInstance
	index    uint64
	mu       sync.RWMutex
}

func newTestServiceClientWithServices(services map[string][]*consulServiceInstance) *testServiceClient {
	return &testServiceClient{
		services: services,
	}
}

func (m *testServiceClient) GetService(ctx context.Context, serviceName string) ([]*consulServiceInstance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.services == nil {
		return []*consulServiceInstance{}, nil
	}
	instances, ok := m.services[serviceName]
	if !ok {
		return []*consulServiceInstance{}, nil
	}
	// Return a copy
	result := make([]*consulServiceInstance, len(instances))
	copy(result, instances)
	return result, nil
}

func (m *testServiceClient) Watch(ctx context.Context, serviceName string, lastIndex uint64) ([]*consulServiceInstance, uint64, error) {
	m.mu.Lock()
	m.index++
	newIndex := m.index
	m.mu.Unlock()
	instances, err := m.GetService(ctx, serviceName)
	return instances, newIndex, err
}

// Test property: service registration and deregistration
// Validates: Requirements 11.1, 11.7, 11.8
func TestRegistrar_Property_ServiceRegistration(t *testing.T) {
	// This test verifies that service instances can be registered and deregistered
	// as specified in requirements 11.1, 11.7, 11.8

	mockClient := &testRegistrationClient{}

	config := &RegistrarConfig{
		Address: "localhost:8500",
		Timeout: 5 * time.Second,
	}

	r := NewRegistrar(config, WithClient(mockClient))
	ctx := context.Background()

	// Test service registration
	service := &registry.ServiceInstance{
		ID:        "service-1",
		Name:      "test-service",
		Version:   "v1.0.0",
		Endpoints: []string{"http://localhost:8080"},
		Metadata: map[string]string{
			"env":    "production",
			"region": "us-east-1",
		},
	}

	// Register should succeed
	err := r.Register(ctx, service)
	require.NoError(t, err, "Service registration should succeed")

	// Verify the service was registered with the mock
	assert.True(t, mockClient.IsRegistered(service.ID), "Service should be registered in mock")

	// Deregister should succeed
	err = r.Deregister(ctx, service)
	require.NoError(t, err, "Service deregistration should succeed")

	// Verify the service was deregistered
	assert.False(t, mockClient.IsRegistered(service.ID), "Service should be deregistered in mock")
}

// testRegistrationClient is a test implementation for registration
type testRegistrationClient struct {
	services map[string]*consulServiceInstance
}

func (m *testRegistrationClient) Register(ctx context.Context, instance *consulServiceInstance) error {
	if m.services == nil {
		m.services = make(map[string]*consulServiceInstance)
	}
	m.services[instance.ID] = instance
	return nil
}

func (m *testRegistrationClient) Deregister(ctx context.Context, serviceID string) error {
	if m.services != nil {
		delete(m.services, serviceID)
	}
	return nil
}

func (m *testRegistrationClient) IsRegistered(serviceID string) bool {
	if m.services == nil {
		return false
	}
	_, exists := m.services[serviceID]
	return exists
}
