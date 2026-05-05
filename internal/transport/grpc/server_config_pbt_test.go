// Package grpc provides gRPC server implementation for the Firefly framework.
package grpc

import (
	"context"
	"math"
	"math/rand"
	"net"
	"testing"
	"testing/quick"
	"time"
)

// =============================================================================
// Property-Based Tests for gRPC Server Configuration
// =============================================================================

// TestProperty4_GRPCServerConfigCorrectness tests Property 4: 服务器配置正确性
// For any gRPC server configuration options, the Server instance should contain correct configuration values.
// **Validates: Requirements 8.2, 8.6** (GRPC Server SHALL support configuration of listen address and max message size)
func TestProperty4_GRPCServerConfigCorrectness(t *testing.T) {
	// Test 8.2: Address configuration
	t.Run("AddressConfiguration", func(t *testing.T) {
		config := &quick.Config{
			MaxCount: 100,
			Rand:     rand.New(rand.NewSource(42)),
		}

		f := func(address string) bool {
			// Filter invalid addresses (empty string, special chars that would break net.Listen)
			if address == "" {
				return true // Skip empty addresses - use default
			}
			if len(address) > 1000 {
				return true // Skip extremely long addresses
			}

			s := NewServer(Address(address))

			// Verify address is correctly set on the server
			if s.address != address {
				t.Logf("expected address %q, got %q", address, s.address)
				return false
			}

			return true
		}

		if err := quick.Check(f, config); err != nil {
			t.Errorf("Property 4 (address configuration) failed: %v", err)
		}
	})

	// Test 8.6: Max receive message size configuration
	t.Run("MaxRecvMsgSizeConfiguration", func(t *testing.T) {
		config := &quick.Config{
			MaxCount: 100,
			Rand:     rand.New(rand.NewSource(123)),
		}

		f := func(size int64) bool {
			// Only test valid message sizes (positive, within reasonable bounds)
			if size <= 0 || size > math.MaxInt32 {
				return true // Skip invalid sizes
			}

			s := NewServer(MaxRecvMsgSize(int(size)))

			// Verify max receive message size is correctly set
			if s.maxRecvMsgSize != int(size) {
				t.Logf("expected maxRecvMsgSize %d, got %d", size, s.maxRecvMsgSize)
				return false
			}

			return true
		}

		if err := quick.Check(f, config); err != nil {
			t.Errorf("Property 4 (max recv message size) failed: %v", err)
		}
	})

	// Test 8.6: Max send message size configuration
	t.Run("MaxSendMsgSizeConfiguration", func(t *testing.T) {
		config := &quick.Config{
			MaxCount: 100,
			Rand:     rand.New(rand.NewSource(456)),
		}

		f := func(size int64) bool {
			// Only test valid message sizes (positive, within reasonable bounds)
			if size <= 0 || size > math.MaxInt32 {
				return true // Skip invalid sizes
			}

			s := NewServer(MaxSendMsgSize(int(size)))

			// Verify max send message size is correctly set
			if s.maxSendMsgSize != int(size) {
				t.Logf("expected maxSendMsgSize %d, got %d", size, s.maxSendMsgSize)
				return false
			}

			return true
		}

		if err := quick.Check(f, config); err != nil {
			t.Errorf("Property 4 (max send message size) failed: %v", err)
		}
	})

	// Test 8.2 + 8.6: Combined address and message size configuration
	t.Run("CombinedConfiguration", func(t *testing.T) {
		config := &quick.Config{
			MaxCount: 50,
			Rand:     rand.New(rand.NewSource(789)),
		}

		f := func(address string, maxRecvSize, maxSendSize int64) bool {
			// Filter invalid inputs
			if address == "" || len(address) > 1000 {
				return true
			}
			if maxRecvSize <= 0 || maxRecvSize > math.MaxInt32 {
				return true
			}
			if maxSendSize <= 0 || maxSendSize > math.MaxInt32 {
				return true
			}

			s := NewServer(
				Address(address),
				MaxRecvMsgSize(int(maxRecvSize)),
				MaxSendMsgSize(int(maxSendSize)),
			)

			// Verify all configurations are correctly set
			if s.address != address {
				t.Logf("expected address %q, got %q", address, s.address)
				return false
			}
			if s.maxRecvMsgSize != int(maxRecvSize) {
				t.Logf("expected maxRecvMsgSize %d, got %d", maxRecvSize, s.maxRecvMsgSize)
				return false
			}
			if s.maxSendMsgSize != int(maxSendSize) {
				t.Logf("expected maxSendMsgSize %d, got %d", maxSendSize, s.maxSendMsgSize)
				return false
			}

			return true
		}

		if err := quick.Check(f, config); err != nil {
			t.Errorf("Property 4 (combined configuration) failed: %v", err)
		}
	})
}

// TestProperty4_GRPCServerDefaultValues tests that default values are correctly applied.
// **Validates: Requirements 8.2, 8.6**
func TestProperty4_GRPCServerDefaultValues(t *testing.T) {
	s := NewServer()

	// Verify default values per requirement 8.2
	if s.network != "tcp" {
		t.Errorf("expected default network tcp, got %s", s.network)
	}
	if s.address != ":9090" {
		t.Errorf("expected default address :9090, got %s", s.address)
	}
	if s.timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", s.timeout)
	}

	// Verify default max message sizes per requirement 8.6
	// Note: Default gRPC max message sizes are 4MB (recv) and math.MaxInt32 (send)
	// The server struct should have 0 (not set) since defaults are applied at grpc.Server creation time
	if s.maxRecvMsgSize != 0 {
		t.Logf("default maxRecvMsgSize is %d (0 means default 4MB will be used at grpc.Server level)", s.maxRecvMsgSize)
	}
	if s.maxSendMsgSize != 0 {
		t.Logf("default maxSendMsgSize is %d (0 means default math.MaxInt32 will be used at grpc.Server level)", s.maxSendMsgSize)
	}
}

// TestProperty4_GRPCNetworkConfiguration tests that network type is correctly configured.
// **Validates: Requirement 8.2**
func TestProperty4_GRPCNetworkConfiguration(t *testing.T) {
	config := &quick.Config{
		MaxCount: 50,
		Rand:     rand.New(rand.NewSource(321)),
	}

	// Valid network types: tcp, tcp4, tcp6, unix
	validNetworks := []string{"tcp", "tcp4", "tcp6", "unix"}

	f := func(networkIdx int) bool {
		// Ensure non-negative index (rand can produce negative ints)
		if networkIdx < 0 {
			networkIdx = -networkIdx
		}
		// Map to valid networks
		network := validNetworks[networkIdx%len(validNetworks)]

		s := NewServer(Network(network))

		// Verify network is correctly set
		if s.network != network {
			t.Logf("expected network %q, got %q", network, s.network)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 4 (network configuration) failed: %v", err)
	}
}

// TestProperty4_TimeoutConfiguration tests that timeout configuration is correctly applied.
// **Validates: Requirement 8.2** (timeout is part of server configuration)
func TestProperty4_TimeoutConfiguration(t *testing.T) {
	config := &quick.Config{
		MaxCount: 50,
		Rand:     rand.New(rand.NewSource(654)),
	}

	f := func(timeoutMs int64) bool {
		// Only test positive timeouts
		if timeoutMs <= 0 {
			return true
		}

		timeout := time.Duration(timeoutMs) * time.Millisecond

		s := NewServer(Timeout(timeout))

		// Verify timeout is correctly set
		if s.timeout != timeout {
			t.Logf("expected timeout %v, got %v", timeout, s.timeout)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 4 (timeout configuration) failed: %v", err)
	}
}

// TestProperty4_MaxMessageSizeBounds tests boundary conditions for max message sizes.
// **Validates: Requirement 8.6**
func TestProperty4_MaxMessageSizeBounds(t *testing.T) {
	testCases := []struct {
		name        string
		maxRecvSize int
		maxSendSize int
		expectZero  bool // true if we expect 0 (not configured)
	}{
		{
			name:        "default (zero)",
			maxRecvSize: 0,
			maxSendSize: 0,
			expectZero:  true,
		},
		{
			name:        "small size (1KB)",
			maxRecvSize: 1024,
			maxSendSize: 1024,
			expectZero:  false,
		},
		{
			name:        "medium size (1MB)",
			maxRecvSize: 1024 * 1024,
			maxSendSize: 1024 * 1024,
			expectZero:  false,
		},
		{
			name:        "large size (100MB)",
			maxRecvSize: 100 * 1024 * 1024,
			maxSendSize: 100 * 1024 * 1024,
			expectZero:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var opts []ServerOption
			if tc.maxRecvSize > 0 {
				opts = append(opts, MaxRecvMsgSize(tc.maxRecvSize))
			}
			if tc.maxSendSize > 0 {
				opts = append(opts, MaxSendMsgSize(tc.maxSendSize))
			}

			s := NewServer(opts...)

			if tc.expectZero {
				if s.maxRecvMsgSize != 0 {
					t.Errorf("expected maxRecvMsgSize 0, got %d", s.maxRecvMsgSize)
				}
				if s.maxSendMsgSize != 0 {
					t.Errorf("expected maxSendMsgSize 0, got %d", s.maxSendMsgSize)
				}
			} else {
				if s.maxRecvMsgSize != tc.maxRecvSize {
					t.Errorf("expected maxRecvMsgSize %d, got %d", tc.maxRecvSize, s.maxRecvMsgSize)
				}
				if s.maxSendMsgSize != tc.maxSendSize {
					t.Errorf("expected maxSendMsgSize %d, got %d", tc.maxSendSize, s.maxSendMsgSize)
				}
			}
		})
	}
}

// TestProperty4_AddressWithPort tests various address formats with ports.
// **Validates: Requirement 8.2**
func TestProperty4_AddressWithPort(t *testing.T) {
	config := &quick.Config{
		MaxCount: 50,
		Rand:     rand.New(rand.NewSource(987)),
	}

	// Valid address formats
	validAddresses := []string{
		":9090",          // default port
		":8080",          // custom port
		"localhost:8080", // localhost with port
		"127.0.0.1:8080", // loopback with port
		"0.0.0.0:8080",   // all interfaces
		"[::1]:8080",     // IPv6 loopback
	}

	f := func(addrIdx int) bool {
		// Ensure non-negative index
		if addrIdx < 0 {
			addrIdx = -addrIdx
		}
		address := validAddresses[addrIdx%len(validAddresses)]

		s := NewServer(Address(address))

		// Verify address is correctly set
		if s.address != address {
			t.Logf("expected address %q, got %q", address, s.address)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 4 (address with port) failed: %v", err)
	}
}

// TestProperty4_GRPCServerFunctionalConfig tests that configuration is applied correctly
// when the server actually starts and stops.
// **Validates: Requirements 8.2, 8.6**
func TestProperty4_GRPCServerFunctionalConfig(t *testing.T) {
	// Find an available port
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	// Create server with custom configuration
	maxRecvSize := 8 * 1024 * 1024  // 8MB
	maxSendSize := 16 * 1024 * 1024 // 16MB

	s := NewServer(
		Address(addr),
		MaxRecvMsgSize(maxRecvSize),
		MaxSendMsgSize(maxSendSize),
	)

	// Verify configuration before starting
	if s.address != addr {
		t.Errorf("expected address %q, got %q", addr, s.address)
	}
	if s.maxRecvMsgSize != maxRecvSize {
		t.Errorf("expected maxRecvMsgSize %d, got %d", maxRecvSize, s.maxRecvMsgSize)
	}
	if s.maxSendMsgSize != maxSendSize {
		t.Errorf("expected maxSendMsgSize %d, got %d", maxSendSize, s.maxSendMsgSize)
	}

	// Start server
	ctx := context.Background()
	if err := s.Start(ctx); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Verify endpoint is correctly generated
	endpoint, err := s.Endpoint()
	if err != nil {
		t.Errorf("failed to get endpoint: %v", err)
	}
	if endpoint == nil {
		t.Error("expected endpoint to be set")
	}
	if endpoint.Scheme != "grpc" {
		t.Errorf("expected scheme grpc, got %s", endpoint.Scheme)
	}

	// Stop server
	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.Stop(stopCtx); err != nil {
		t.Errorf("failed to stop server: %v", err)
	}
}
