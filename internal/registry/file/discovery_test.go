package file

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscovery_Load(t *testing.T) {
	tests := []struct {
		name        string
		yamlData    string
		expectError bool
	}{
		{
			name: "valid yaml",
			yamlData: `
services:
  - name: user-service
    instances:
      - id: user-1
        name: user-service
        version: v1.0.0
        endpoints:
          - http://localhost:8080
        metadata:
          env: prod
  - name: order-service
    instances:
      - id: order-1
        name: order-service
        version: v1.0.0
        endpoints:
          - http://localhost:8081
`,
			expectError: false,
		},
		{
			name:        "empty yaml",
			yamlData:    ``,
			expectError: false,
		},
		{
			name: "invalid yaml",
			yamlData: `
services:
  - name: [invalid
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDiscovery()
			err := d.Load([]byte(tt.yamlData))

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDiscovery_GetService(t *testing.T) {
	tests := []struct {
		name          string
		yamlData      string
		serviceName   string
		expectedCount int
		expectedIDs   []string
	}{
		{
			name: "existing service",
			yamlData: `
services:
  - name: user-service
    instances:
      - id: user-1
        name: user-service
        version: v1.0.0
        endpoints:
          - http://localhost:8080
        metadata:
          env: prod
      - id: user-2
        name: user-service
        version: v1.0.0
        endpoints:
          - http://localhost:8081
        metadata:
          env: dev
`,
			serviceName:   "user-service",
			expectedCount: 2,
			expectedIDs:   []string{"user-1", "user-2"},
		},
		{
			name: "another existing service",
			yamlData: `
services:
  - name: order-service
    instances:
      - id: order-1
        name: order-service
        version: v2.0.0
`,
			serviceName:   "order-service",
			expectedCount: 1,
			expectedIDs:   []string{"order-1"},
		},
		{
			name: "non-existing service",
			yamlData: `
services:
  - name: test-service
    instances:
      - id: test-1
        name: test-service
`,
			serviceName:   "unknown-service",
			expectedCount: 0,
			expectedIDs:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDiscovery()
			require.NoError(t, d.Load([]byte(tt.yamlData)))

			ctx := context.Background()
			instances, err := d.GetService(ctx, tt.serviceName)

			require.NoError(t, err)
			assert.Len(t, instances, tt.expectedCount)

			// Verify instance IDs
			if len(tt.expectedIDs) > 0 {
				for i, expectedID := range tt.expectedIDs {
					assert.Equal(t, expectedID, instances[i].ID)
				}
			}

			// Verify instances are copies, not references
			if len(instances) > 0 {
				originalID := instances[0].ID
				instances[0].ID = "modified"
				newInstances, _ := d.GetService(ctx, tt.serviceName)
				assert.Equal(t, originalID, newInstances[0].ID, "Modifying returned instance should not affect internal state")
			}
		})
	}
}

func TestDiscovery_Watch(t *testing.T) {
	d := NewDiscovery()

	// Load initial data
	yamlData := `
services:
  - name: user-service
    instances:
      - id: user-1
        name: user-service
        version: v1.0.0
`
	require.NoError(t, d.Load([]byte(yamlData)))

	ctx := context.Background()
	watcher, err := d.Watch(ctx, "user-service")
	require.NoError(t, err)
	defer watcher.Stop()

	// Get initial instances
	instances, err := watcher.Next()
	require.NoError(t, err)
	assert.Len(t, instances, 1)
	assert.Equal(t, "user-1", instances[0].ID)
}

func TestFileWatcher_Stop(t *testing.T) {
	d := NewDiscovery()
	require.NoError(t, d.Load([]byte(`
services:
  - name: test-service
    instances:
      - id: test-1
        name: test-service
`)))

	ctx := context.Background()
	watcher, err := d.Watch(ctx, "test-service")
	require.NoError(t, err)

	// Stop should be idempotent
	assert.NoError(t, watcher.Stop())
	assert.NoError(t, watcher.Stop())
}

// Test property: service discovery configuration loading
// Validates: Requirements 11.3
func TestDiscovery_Property_ServiceDiscoveryConfigLoading(t *testing.T) {
	// This test verifies that service configurations can be correctly loaded from YAML files
	// as specified in requirement 11.3

	tests := []struct {
		name             string
		yamlData         string
		expectedServices map[string]int    // serviceName -> instance count
		expectedVersions map[string]string // serviceName -> version
	}{
		{
			name: "multiple services with multiple instances",
			yamlData: `
services:
  - name: api-gateway
    instances:
      - id: gateway-1
        name: api-gateway
        version: v1.0.0
        endpoints:
          - http://gateway-1:8080
      - id: gateway-2
        name: api-gateway
        version: v1.0.0
        endpoints:
          - http://gateway-2:8080
  - name: user-service
    instances:
      - id: user-1
        name: user-service
        version: v2.1.0
        endpoints:
          - http://user-1:8081
`,
			expectedServices: map[string]int{
				"api-gateway":  2,
				"user-service": 1,
			},
			expectedVersions: map[string]string{
				"api-gateway":  "v1.0.0",
				"user-service": "v2.1.0",
			},
		},
		{
			name: "service with metadata",
			yamlData: `
services:
  - name: order-service
    instances:
      - id: order-1
        name: order-service
        version: v1.0.0
        metadata:
          env: production
          region: us-east-1
          team: backend
`,
			expectedServices: map[string]int{
				"order-service": 1,
			},
			expectedVersions: map[string]string{
				"order-service": "v1.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDiscovery()
			require.NoError(t, d.Load([]byte(tt.yamlData)))

			ctx := context.Background()
			for serviceName, expectedCount := range tt.expectedServices {
				instances, err := d.GetService(ctx, serviceName)
				require.NoError(t, err, "Failed to get service: %s", serviceName)
				assert.Len(t, instances, expectedCount, "Unexpected instance count for service: %s", serviceName)

				if expectedVersion, ok := tt.expectedVersions[serviceName]; ok && len(instances) > 0 {
					assert.Equal(t, expectedVersion, instances[0].Version, "Unexpected version for service: %s", serviceName)
				}
			}
		})
	}
}
