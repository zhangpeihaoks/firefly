package consul

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zhangpeihaoks/firefly/internal/registry"
)

func TestRegistrar_Register(t *testing.T) {
	tests := []struct {
		name        string
		service     *registry.ServiceInstance
		expectError bool
	}{
		{
			name: "valid service",
			service: &registry.ServiceInstance{
				ID:        "test-id",
				Name:      "test-service",
				Version:   "v1.0.0",
				Endpoints: []string{"http://localhost:8080"},
				Metadata:  map[string]string{"env": "prod"},
			},
			expectError: false,
		},
		{
			name:        "nil service",
			service:     nil,
			expectError: true,
		},
		{
			name: "empty id",
			service: &registry.ServiceInstance{
				Name:    "test-service",
				Version: "v1.0.0",
			},
			expectError: true,
		},
		{
			name: "empty name",
			service: &registry.ServiceInstance{
				ID:      "test-id",
				Version: "v1.0.0",
			},
			expectError: true,
		},
		{
			name: "service without endpoints",
			service: &registry.ServiceInstance{
				ID:        "test-id",
				Name:      "test-service",
				Version:   "v1.0.0",
				Endpoints: []string{},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RegistrarConfig{
				Address: "localhost:8500",
				Timeout: 5 * time.Second,
			}

			r := NewRegistrar(config)
			ctx := context.Background()
			err := r.Register(ctx, tt.service)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegistrar_Deregister(t *testing.T) {
	tests := []struct {
		name        string
		register    bool
		service     *registry.ServiceInstance
		expectError bool
	}{
		{
			name:     "registered service",
			register: true,
			service: &registry.ServiceInstance{
				ID:        "test-id",
				Name:      "test-service",
				Version:   "v1.0.0",
				Endpoints: []string{"http://localhost:8080"},
			},
			expectError: false,
		},
		{
			name:     "not registered service",
			register: false,
			service: &registry.ServiceInstance{
				ID:      "unknown-id",
				Name:    "unknown-service",
				Version: "v1.0.0",
			},
			expectError: false, // Deregister returns nil for unregistered services
		},
		{
			name:        "nil service",
			register:    false,
			service:     nil,
			expectError: true,
		},
		{
			name:        "empty id",
			register:    false,
			service:     &registry.ServiceInstance{Name: "test"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RegistrarConfig{
				Address: "localhost:8500",
				Timeout: 5 * time.Second,
			}

			r := NewRegistrar(config)
			ctx := context.Background()

			if tt.register {
				require.NoError(t, r.Register(ctx, tt.service))
			}

			err := r.Deregister(ctx, tt.service)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegistrar_DoubleRegister(t *testing.T) {
	config := &RegistrarConfig{
		Address: "localhost:8500",
		Timeout: 5 * time.Second,
	}

	r := NewRegistrar(config)
	ctx := context.Background()

	service := &registry.ServiceInstance{
		ID:        "test-id",
		Name:      "test-service",
		Version:   "v1.0.0",
		Endpoints: []string{"http://localhost:8080"},
	}

	// First registration should succeed
	require.NoError(t, r.Register(ctx, service))

	// Second registration should also succeed (idempotent)
	require.NoError(t, r.Register(ctx, service))
}

func TestRegistrar_CustomClient(t *testing.T) {
	// Create a mock client that returns errors
	mockClient := &errorMockClient{err: errors.New("connection refused")}

	config := &RegistrarConfig{
		Address: "localhost:8500",
		Timeout: 5 * time.Second,
	}

	r := NewRegistrar(config, WithClient(mockClient))
	ctx := context.Background()

	service := &registry.ServiceInstance{
		ID:        "test-id",
		Name:      "test-service",
		Version:   "v1.0.0",
		Endpoints: []string{"http://localhost:8080"},
	}

	// Registration should fail due to mock client error
	err := r.Register(ctx, service)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

// errorMockClient is a mock client that always returns errors
type errorMockClient struct {
	err error
}

func (m *errorMockClient) Register(ctx context.Context, instance *consulServiceInstance) error {
	return m.err
}

func (m *errorMockClient) Deregister(ctx context.Context, serviceID string) error {
	return m.err
}

func TestToConsulInstance(t *testing.T) {
	tests := []struct {
		name            string
		service         *registry.ServiceInstance
		expectedAddress string
		expectedPort    int
	}{
		{
			name: "http endpoint with port",
			service: &registry.ServiceInstance{
				ID:        "test-id",
				Name:      "test-service",
				Version:   "v1.0.0",
				Endpoints: []string{"http://localhost:8080"},
			},
			expectedAddress: "localhost",
			expectedPort:    8080,
		},
		{
			name: "endpoint without scheme",
			service: &registry.ServiceInstance{
				ID:        "test-id",
				Name:      "test-service",
				Version:   "v1.0.0",
				Endpoints: []string{"192.168.1.1:9090"},
			},
			expectedAddress: "192.168.1.1",
			expectedPort:    9090,
		},
		{
			name: "no endpoint",
			service: &registry.ServiceInstance{
				ID:        "test-id",
				Name:      "test-service",
				Version:   "v1.0.0",
				Endpoints: []string{},
			},
			expectedAddress: "",
			expectedPort:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RegistrarConfig{
				Address: "localhost:8500",
				Timeout: 5 * time.Second,
			}

			r := NewRegistrar(config)
			instance, err := r.toConsulInstance(tt.service)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedAddress, instance.Address)
			assert.Equal(t, tt.expectedPort, instance.Port)
			assert.Equal(t, tt.service.Version, instance.Meta["version"])
		})
	}
}
