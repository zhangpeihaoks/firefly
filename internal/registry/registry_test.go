package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceInstance_Clone(t *testing.T) {
	tests := []struct {
		name     string
		instance *ServiceInstance
	}{
		{
			name: "empty instance",
			instance: &ServiceInstance{
				Metadata:  make(map[string]string),
				Endpoints: []string{},
			},
		},
		{
			name: "full instance",
			instance: &ServiceInstance{
				ID:        "test-id",
				Name:      "test-service",
				Version:   "v1.0.0",
				Metadata:  map[string]string{"env": "prod", "region": "us-east"},
				Endpoints: []string{"http://localhost:8080", "grpc://localhost:9090"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloned := tt.instance.Clone()

			// Verify values are equal
			assert.Equal(t, tt.instance.ID, cloned.ID)
			assert.Equal(t, tt.instance.Name, cloned.Name)
			assert.Equal(t, tt.instance.Version, cloned.Version)

			// Verify maps are copied, not referenced
			if tt.instance.Metadata != nil {
				assert.Equal(t, tt.instance.Metadata, cloned.Metadata)
				// Modify original, ensure clone is not affected
				tt.instance.Metadata["test"] = "modified"
				assert.NotEqual(t, tt.instance.Metadata, cloned.Metadata)
			}

			// Verify slices are copied, not referenced
			if len(tt.instance.Endpoints) > 0 {
				assert.Equal(t, tt.instance.Endpoints, cloned.Endpoints)
				// Modify original, ensure clone is not affected
				tt.instance.Endpoints[0] = "modified"
				assert.NotEqual(t, tt.instance.Endpoints, cloned.Endpoints)
			}
		})
	}
}

func TestNewServiceInstance(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		expected *ServiceInstance
	}{
		{
			name:     "no options",
			opts:     []Option{},
			expected: &ServiceInstance{Metadata: map[string]string{}, Endpoints: []string{}},
		},
		{
			name: "with all options",
			opts: []Option{
				WithID("test-id"),
				WithName("test-service"),
				WithVersion("v1.0.0"),
				WithMetadata(map[string]string{"env": "prod"}),
				WithEndpoints("http://localhost:8080"),
			},
			expected: &ServiceInstance{
				ID:        "test-id",
				Name:      "test-service",
				Version:   "v1.0.0",
				Metadata:  map[string]string{"env": "prod"},
				Endpoints: []string{"http://localhost:8080"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := NewServiceInstance(tt.opts...)
			assert.Equal(t, tt.expected.ID, instance.ID)
			assert.Equal(t, tt.expected.Name, instance.Name)
			assert.Equal(t, tt.expected.Version, instance.Version)
		})
	}
}

func TestFilterInstances(t *testing.T) {
	instances := []*ServiceInstance{
		{ID: "1", Name: "service-a", Version: "v1.0.0", Metadata: map[string]string{"env": "prod"}},
		{ID: "2", Name: "service-a", Version: "v2.0.0", Metadata: map[string]string{"env": "dev"}},
		{ID: "3", Name: "service-b", Version: "v1.0.0", Metadata: map[string]string{"env": "prod"}},
	}

	tests := []struct {
		name     string
		filter   ServiceInstanceFilter
		expected int
	}{
		{
			name:     "nil filter",
			filter:   nil,
			expected: 3,
		},
		{
			name:     "version filter",
			filter:   VersionFilter("v1.0.0"),
			expected: 2,
		},
		{
			name:     "metadata filter",
			filter:   MetadataFilter("env", "prod"),
			expected: 2,
		},
		{
			name:     "non-matching filter",
			filter:   VersionFilter("v3.0.0"),
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterInstances(instances, tt.filter)
			assert.Len(t, result, tt.expected)
		})
	}
}
