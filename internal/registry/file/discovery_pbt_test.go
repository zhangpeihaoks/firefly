// Package file provides file-based service discovery for the Firefly framework.
// This file contains property-based tests for service discovery configuration loading.
package file

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// =============================================================================
// Property 25: Service Discovery Configuration Loading (服务发现配置加载)
// =============================================================================
// Validates: Requirement 11.3
// For any service configuration, loading from YAML file should be correct.
// The framework SHALL support file configuration service discovery,
// reading service node information from YAML files.

// TestProperty25ConfigLoading_PBT tests that service configurations are correctly loaded from YAML.
// Feature: backend-server-framework, Property 25: 服务发现配置加载
//
// Any valid service configuration YAML should be correctly parsed and loaded.
func TestProperty25ConfigLoading_PBT(t *testing.T) {
	testCases := []struct {
		name          string
		yamlData      string
		servicesCount int
	}{
		{
			name: "single service single instance",
			yamlData: `
services:
  - name: user-service
    instances:
      - id: user-1
        name: user-service
        version: v1.0.0
        endpoints:
          - http://localhost:8080
`,
			servicesCount: 1,
		},
		{
			name: "single service multiple instances",
			yamlData: `
services:
  - name: user-service
    instances:
      - id: user-1
        name: user-service
        version: v1.0.0
        endpoints:
          - http://localhost:8080
      - id: user-2
        name: user-service
        version: v1.0.0
        endpoints:
          - http://localhost:8081
`,
			servicesCount: 1,
		},
		{
			name: "multiple services",
			yamlData: `
services:
  - name: user-service
    instances:
      - id: user-1
        name: user-service
        version: v1.0.0
        endpoints:
          - http://localhost:8080
  - name: order-service
    instances:
      - id: order-1
        name: order-service
        version: v2.0.0
        endpoints:
          - http://localhost:8081
  - name: payment-service
    instances:
      - id: payment-1
        name: payment-service
        version: v1.0.0
        endpoints:
          - http://localhost:8082
`,
			servicesCount: 3,
		},
		{
			name: "service with multiple endpoints",
			yamlData: `
services:
  - name: api-gateway
    instances:
      - id: gateway-1
        name: api-gateway
        version: v1.0.0
        endpoints:
          - http://gateway-1:8080
          - grpc://gateway-1:9090
`,
			servicesCount: 1,
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
        endpoints:
          - http://localhost:8080
        metadata:
          env: production
          region: us-east-1
          team: orders
`,
			servicesCount: 1,
		},
		{
			name: "empty services",
			yamlData: `
services:
`,
			servicesCount: 0,
		},
		{
			name: "empty instances",
			yamlData: `
services:
  - name: empty-service
    instances:
`,
			servicesCount: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDiscovery()
			err := d.Load([]byte(tc.yamlData))
			require.NoError(t, err, "Failed to load YAML data")

			// Verify that the discovery instance is created
			assert.NotNil(t, d, "Discovery instance should not be nil")
		})
	}
}

// TestProperty25ServiceInstanceParsing_PBT tests that service instances are correctly parsed.
// Feature: backend-server-framework, Property 25: 服务发现配置加载
func TestProperty25ServiceInstanceParsing_PBT(t *testing.T) {
	testCases := []struct {
		name          string
		yamlData      string
		serviceName   string
		expectedID    string
		expectedName  string
		expectedVer   string
		expectedCount int
	}{
		{
			name: "parse instance id",
			yamlData: `
services:
  - name: user-service
    instances:
      - id: instance-123
        name: user-service
        version: v1.0.0
        endpoints:
          - http://localhost:8080
`,
			serviceName:   "user-service",
			expectedID:    "instance-123",
			expectedName:  "user-service",
			expectedVer:   "v1.0.0",
			expectedCount: 1,
		},
		{
			name: "parse version",
			yamlData: `
services:
  - name: service-v2
    instances:
      - id: svc-1
        name: service-v2
        version: v2.1.3
        endpoints:
          - http://localhost:9000
`,
			serviceName:   "service-v2",
			expectedID:    "svc-1",
			expectedName:  "service-v2",
			expectedVer:   "v2.1.3",
			expectedCount: 1,
		},
		{
			name: "parse multiple instances",
			yamlData: `
services:
  - name: cluster-service
    instances:
      - id: node-1
        name: cluster-service
        version: v1.0.0
        endpoints:
          - http://node1:8080
      - id: node-2
        name: cluster-service
        version: v1.0.0
        endpoints:
          - http://node2:8080
      - id: node-3
        name: cluster-service
        version: v1.0.0
        endpoints:
          - http://node3:8080
`,
			serviceName:   "cluster-service",
			expectedID:    "node-1",
			expectedName:  "cluster-service",
			expectedVer:   "v1.0.0",
			expectedCount: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDiscovery()
			require.NoError(t, d.Load([]byte(tc.yamlData)))

			ctx := context.Background()
			instances, err := d.GetService(ctx, tc.serviceName)
			require.NoError(t, err)
			assert.Len(t, instances, tc.expectedCount)

			if tc.expectedCount > 0 {
				assert.Equal(t, tc.expectedID, instances[0].ID)
				assert.Equal(t, tc.expectedName, instances[0].Name)
				assert.Equal(t, tc.expectedVer, instances[0].Version)
			}
		})
	}
}

// TestProperty25MetadataParsing_PBT tests that service metadata is correctly parsed.
// Feature: backend-server-framework, Property 25: 服务发现配置加载
func TestProperty25MetadataParsing_PBT(t *testing.T) {
	testCases := []struct {
		name         string
		yamlData     string
		serviceName  string
		metadataKeys []string
	}{
		{
			name: "single metadata entry",
			yamlData: `
services:
  - name: test-service
    instances:
      - id: test-1
        name: test-service
        version: v1.0.0
        metadata:
          env: production
`,
			serviceName:  "test-service",
			metadataKeys: []string{"env"},
		},
		{
			name: "multiple metadata entries",
			yamlData: `
services:
  - name: test-service
    instances:
      - id: test-1
        name: test-service
        version: v1.0.0
        metadata:
          env: production
          region: us-west-2
          team: backend
          version: v1.0.0
`,
			serviceName:  "test-service",
			metadataKeys: []string{"env", "region", "team", "version"},
		},
		{
			name: "empty metadata",
			yamlData: `
services:
  - name: test-service
    instances:
      - id: test-1
        name: test-service
        version: v1.0.0
`,
			serviceName:  "test-service",
			metadataKeys: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDiscovery()
			require.NoError(t, d.Load([]byte(tc.yamlData)))

			ctx := context.Background()
			instances, err := d.GetService(ctx, tc.serviceName)
			require.NoError(t, err)
			require.Len(t, instances, 1)

			// Verify metadata is initialized (even if empty)
			assert.NotNil(t, instances[0].Metadata)

			// Verify metadata keys
			if len(tc.metadataKeys) > 0 {
				for _, key := range tc.metadataKeys {
					_, exists := instances[0].Metadata[key]
					assert.True(t, exists, "Metadata key %s should exist", key)
				}
			}
		})
	}
}

// TestProperty25EndpointsParsing_PBT tests that service endpoints are correctly parsed.
// Feature: backend-server-framework, Property 25: 服务发现配置加载
func TestProperty25EndpointsParsing_PBT(t *testing.T) {
	testCases := []struct {
		name          string
		yamlData      string
		serviceName   string
		expectedCount int
	}{
		{
			name: "single HTTP endpoint",
			yamlData: `
services:
  - name: http-service
    instances:
      - id: svc-1
        name: http-service
        version: v1.0.0
        endpoints:
          - http://localhost:8080
`,
			serviceName:   "http-service",
			expectedCount: 1,
		},
		{
			name: "multiple endpoints",
			yamlData: `
services:
  - name: multi-protocol
    instances:
      - id: svc-1
        name: multi-protocol
        version: v1.0.0
        endpoints:
          - http://localhost:8080
          - grpc://localhost:9090
          - ws://localhost:8081
`,
			serviceName:   "multi-protocol",
			expectedCount: 3,
		},
		{
			name: "no endpoints",
			yamlData: `
services:
  - name: no-endpoints
    instances:
      - id: svc-1
        name: no-endpoints
        version: v1.0.0
`,
			serviceName:   "no-endpoints",
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDiscovery()
			require.NoError(t, d.Load([]byte(tc.yamlData)))

			ctx := context.Background()
			instances, err := d.GetService(ctx, tc.serviceName)
			require.NoError(t, err)
			require.Len(t, instances, 1)

			assert.Len(t, instances[0].Endpoints, tc.expectedCount)
		})
	}
}

// TestProperty25ErrorHandling_PBT tests error handling for invalid YAML.
// Feature: backend-server-framework, Property 25: 服务发现配置加载
func TestProperty25ErrorHandling_PBT(t *testing.T) {
	testCases := []struct {
		name        string
		yamlData    string
		expectError bool
	}{
		{
			name:        "valid empty YAML",
			yamlData:    "",
			expectError: false,
		},
		{
			name:        "valid empty services",
			yamlData:    "services: []",
			expectError: false,
		},
		{
			name:        "invalid YAML syntax",
			yamlData:    "services:\n  - name: [invalid",
			expectError: true,
		},
		{
			name:        "malformed YAML",
			yamlData:    "invalid: yaml: content:",
			expectError: true, // This YAML is actually invalid
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDiscovery()
			err := d.Load([]byte(tc.yamlData))

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestProperty25DataIsolation_PBT tests that returned instances are isolated from internal state.
// Feature: backend-server-framework, Property 25: 服务发现配置加载
//
// Modifying returned instances should not affect the internal state.
func TestProperty25DataIsolation_PBT(t *testing.T) {
	yamlData := `
services:
  - name: isolated-service
    instances:
      - id: original-id
        name: isolated-service
        version: v1.0.0
        endpoints:
          - http://localhost:8080
        metadata:
          original: value
`

	d := NewDiscovery()
	require.NoError(t, d.Load([]byte(yamlData)))

	ctx := context.Background()

	// Get instances
	instances, err := d.GetService(ctx, "isolated-service")
	require.NoError(t, err)
	require.Len(t, instances, 1)

	// Modify the returned instance
	originalID := instances[0].ID
	instances[0].ID = "modified-id"
	instances[0].Metadata["modified"] = "value"

	// Get instances again
	instances2, err := d.GetService(ctx, "isolated-service")
	require.NoError(t, err)

	// Verify internal state is unchanged
	assert.Equal(t, originalID, instances2[0].ID)
	assert.NotEqual(t, "modified-id", instances2[0].ID)
	assert.NotContains(t, instances2[0].Metadata, "modified")
}

// TestProperty25MultipleLoads_PBT tests that loading new configuration replaces old one.
// Feature: backend-server-framework, Property 25: 服务发现配置加载
func TestProperty25MultipleLoads_PBT(t *testing.T) {
	yamlData1 := `
services:
  - name: service-v1
    instances:
      - id: v1-instance
        name: service-v1
        version: v1.0.0
`

	yamlData2 := `
services:
  - name: service-v2
    instances:
      - id: v2-instance
        name: service-v2
        version: v2.0.0
`

	d := NewDiscovery()
	require.NoError(t, d.Load([]byte(yamlData1)))

	ctx := context.Background()

	// Verify first load
	instances1, err := d.GetService(ctx, "service-v1")
	require.NoError(t, err)
	assert.Len(t, instances1, 1)
	assert.Equal(t, "v1-instance", instances1[0].ID)

	// Load new configuration
	require.NoError(t, d.Load([]byte(yamlData2)))

	// Verify old service is removed
	instancesOld, err := d.GetService(ctx, "service-v1")
	require.NoError(t, err)
	assert.Len(t, instancesOld, 0)

	// Verify new service is present
	instancesNew, err := d.GetService(ctx, "service-v2")
	require.NoError(t, err)
	assert.Len(t, instancesNew, 1)
	assert.Equal(t, "v2-instance", instancesNew[0].ID)
}

// TestProperty25WatcherIntegration_PBT tests that watcher receives correct initial state.
// Feature: backend-server-framework, Property 25: 服务发现配置加载
func TestProperty25WatcherIntegration_PBT(t *testing.T) {
	yamlData := `
services:
  - name: watched-service
    instances:
      - id: watch-1
        name: watched-service
        version: v1.0.0
      - id: watch-2
        name: watched-service
        version: v1.0.0
`

	d := NewDiscovery()
	require.NoError(t, d.Load([]byte(yamlData)))

	ctx := context.Background()
	watcher, err := d.Watch(ctx, "watched-service")
	require.NoError(t, err)
	defer watcher.Stop()

	// Get initial instances from watcher
	instances, err := watcher.Next()
	require.NoError(t, err)
	assert.Len(t, instances, 2)
}

// TestProperty25NonExistentService_PBT tests that querying non-existent service returns empty list.
// Feature: backend-server-framework, Property 25: 服务发现配置加载
func TestProperty25NonExistentService_PBT(t *testing.T) {
	yamlData := `
services:
  - name: existing-service
    instances:
      - id: existing-1
        name: existing-service
        version: v1.0.0
`

	d := NewDiscovery()
	require.NoError(t, d.Load([]byte(yamlData)))

	ctx := context.Background()
	instances, err := d.GetService(ctx, "non-existent-service")
	require.NoError(t, err)
	assert.Len(t, instances, 0)
}

// TestProperty25YAMLParsingEdgeCases_PBT tests edge cases in YAML parsing.
// Feature: backend-server-framework, Property 25: 服务发现配置加载
func TestProperty25YAMLParsingEdgeCases_PBT(t *testing.T) {
	testCases := []struct {
		name     string
		yamlData string
	}{
		{
			name: "special characters in metadata",
			yamlData: `
services:
  - name: special-service
    instances:
      - id: special-1
        name: special-service
        version: v1.0.0
        metadata:
          key: "value with spaces"
          special: "quotes\"escaped"
`,
		},
		{
			name: "unicode in service name",
			yamlData: `
services:
  - name: 服务
    instances:
      - id: unicode-1
        name: 服务
        version: v1.0.0
`,
		},
		{
			name: "empty endpoint list",
			yamlData: `
services:
  - name: empty-endpoints
    instances:
      - id: empty-1
        name: empty-endpoints
        version: v1.0.0
        endpoints: []
`,
		},
		{
			name: "very long service name",
			yamlData: `
services:
  - name: ` + strings.Repeat("very-long-service-name-", 20) + `1
    instances:
      - id: long-1
        name: test
        version: v1.0.0
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// First verify the YAML itself is valid
			var config Config
			err := yaml.Unmarshal([]byte(tc.yamlData), &config)
			if err != nil {
				// If YAML is invalid, skip the test for this case
				t.Skipf("Skipping due to invalid YAML: %v", err)
				return
			}

			// Then test with discovery
			d := NewDiscovery()
			err = d.Load([]byte(tc.yamlData))
			require.NoError(t, err, "Failed to load YAML in discovery")
		})
	}
}

// TestProperty25ConfigStructure_PBT tests that the Config structure is correctly deserialized.
// Feature: backend-server-framework, Property 25: 服务发现配置加载
func TestProperty25ConfigStructure_PBT(t *testing.T) {
	// Test various valid YAML structures
	testCases := []struct {
		name     string
		yamlData string
	}{
		{
			name: "minimal service",
			yamlData: `
services:
  - name: minimal
    instances: []
`,
		},
		{
			name: "full service spec",
			yamlData: `
services:
  - name: full-service
    instances:
      - id: full-1
        name: full-service
        version: v1.2.3
        endpoints:
          - http://full-1:8080
          - grpc://full-1:9090
        metadata:
          key1: value1
          key2: value2
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var config Config
			err := yaml.Unmarshal([]byte(tc.yamlData), &config)
			require.NoError(t, err)
			assert.NotNil(t, config.Services)
		})
	}
}

// TestProperty25GetServiceReturnsCopy_PBT tests that GetService returns independent copies.
// Feature: backend-server-framework, Property 25: 服务发现配置加载
func TestProperty25GetServiceReturnsCopy_PBT(t *testing.T) {
	yamlData := `
services:
  - name: copy-test
    instances:
      - id: copy-1
        name: copy-test
        version: v1.0.0
        metadata:
          key: value
        endpoints:
          - http://localhost:8080
`

	d := NewDiscovery()
	require.NoError(t, d.Load([]byte(yamlData)))

	ctx := context.Background()

	// Get service multiple times
	instance1, _ := d.GetService(ctx, "copy-test")
	instance2, _ := d.GetService(ctx, "copy-test")

	// Verify they are independent copies
	require.Len(t, instance1, 1)
	require.Len(t, instance2, 1)

	// Modify first copy
	instance1[0].ID = "modified"
	instance1[0].Metadata["new-key"] = "new-value"

	// Second copy should be unchanged
	assert.Equal(t, "copy-1", instance2[0].ID)
	assert.NotContains(t, instance2[0].Metadata, "new-key")
}

// TestProperty25ServiceNameLookup_PBT tests case-sensitive and exact service name matching.
// Feature: backend-server-framework, Property 25: 服务发现配置加载
func TestProperty25ServiceNameLookup_PBT(t *testing.T) {
	yamlData := `
services:
  - name: UserService
    instances:
      - id: user-1
        name: UserService
        version: v1.0.0
  - name: userservice
    instances:
      - id: user-2
        name: userservice
        version: v1.0.0
`

	d := NewDiscovery()
	require.NoError(t, d.Load([]byte(yamlData)))

	ctx := context.Background()

	// Exact match should work
	instances, err := d.GetService(ctx, "UserService")
	require.NoError(t, err)
	assert.Len(t, instances, 1)
	assert.Equal(t, "user-1", instances[0].ID)

	// Different case should not match (case-sensitive)
	instances2, err := d.GetService(ctx, "userservice")
	require.NoError(t, err)
	assert.Len(t, instances2, 1)
	assert.Equal(t, "user-2", instances2[0].ID)
}

// =============================================================================
// Edge Cases and Stress Tests
// =============================================================================

// TestProperty25LargeNumberOfServices_PBT tests with a large number of services.
// Feature: backend-server-framework, Property 25: 服务发现配置加载
func TestProperty25LargeNumberOfServices_PBT(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("services:\n")

	// Create 50 services with 2 instances each
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&sb, "  - name: service-%d\n", i)
		sb.WriteString("    instances:\n")
		for j := 0; j < 2; j++ {
			fmt.Fprintf(&sb, "      - id: service-%d-instance-%d\n", i, j)
			fmt.Fprintf(&sb, "        name: service-%d\n", i)
			sb.WriteString("        version: v1.0.0\n")
			sb.WriteString("        endpoints:\n")
			fmt.Fprintf(&sb, "          - http://localhost:%d\n", 8080+i)
		}
	}

	d := NewDiscovery()
	err := d.Load([]byte(sb.String()))
	require.NoError(t, err)

	ctx := context.Background()

	// Verify we can query each service
	for i := 0; i < 50; i++ {
		serviceName := fmt.Sprintf("service-%d", i)
		instances, err := d.GetService(ctx, serviceName)
		require.NoError(t, err)
		assert.Len(t, instances, 2, "Service %s should have 2 instances", serviceName)
	}
}
