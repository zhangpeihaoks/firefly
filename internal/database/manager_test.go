// Package database provides database connection management for the Firefly framework.
// This file implements tests for the database manager.
package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConnector is a mock implementation of the Connector interface for testing.
type mockConnector struct {
	connected bool
	stats     *PoolStats
	err       error
}

func (m *mockConnector) Connect(ctx context.Context) error {
	if m.err != nil {
		return m.err
	}
	m.connected = true
	return nil
}

func (m *mockConnector) Disconnect(ctx context.Context) error {
	m.connected = false
	return nil
}

func (m *mockConnector) IsConnected() bool {
	return m.connected
}

func (m *mockConnector) Ping(ctx context.Context) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *mockConnector) Stats() *PoolStats {
	return m.stats
}

// mockFactory is a mock implementation of the Factory interface for testing.
type mockFactory struct {
	connector Connector
	err       error
}

func (f *mockFactory) Create(cfg *Config) (Connector, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.connector != nil {
		return f.connector, nil
	}
	return &mockConnector{connected: true}, nil
}

func (f *mockFactory) CreateDB(cfg *Config) (DB, error) {
	return nil, NewError("NOT_SUPPORTED", "mock does not implement DB", nil)
}

func (f *mockFactory) Type() DatabaseType {
	return TypeMySQL
}

// TestNewManager tests the NewManager function.
func TestNewManager(t *testing.T) {
	manager := NewManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.connections)
	assert.NotNil(t, manager.factories)
}

// TestManagerRegisterFactory tests the Manager.RegisterFactory method.
func TestManagerRegisterFactory(t *testing.T) {
	manager := NewManager()
	factory := &mockFactory{}

	manager.RegisterFactory(factory)

	assert.Contains(t, manager.factories, TypeMySQL)
}

// TestManagerConnect tests the Manager.Connect method.
func TestManagerConnect(t *testing.T) {
	t.Run("successful connection", func(t *testing.T) {
		manager := NewManager()
		manager.RegisterFactory(&mockFactory{})

		cfg := &Config{
			Driver: "mysql",
			DSN:    "user:pass@tcp(localhost:3306)/db",
		}

		connector, err := manager.Connect(context.Background(), "primary", cfg)
		require.NoError(t, err)
		assert.NotNil(t, connector)
		assert.True(t, connector.IsConnected())
	})

	t.Run("duplicate connection", func(t *testing.T) {
		manager := NewManager()
		manager.RegisterFactory(&mockFactory{})

		cfg := &Config{
			Driver: "mysql",
			DSN:    "user:pass@tcp(localhost:3306)/db",
		}

		_, err := manager.Connect(context.Background(), "primary", cfg)
		require.NoError(t, err)

		_, err = manager.Connect(context.Background(), "primary", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("unknown driver", func(t *testing.T) {
		manager := NewManager()

		cfg := &Config{
			Driver: "mysql",
			DSN:    "user:pass@tcp(localhost:3306)/db",
		}

		_, err := manager.Connect(context.Background(), "primary", cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no factory registered")
	})
}

// TestManagerGet tests the Manager.Get method.
func TestManagerGet(t *testing.T) {
	manager := NewManager()
	manager.RegisterFactory(&mockFactory{})

	cfg := &Config{
		Driver: "mysql",
		DSN:    "user:pass@tcp(localhost:3306)/db",
	}

	_, err := manager.Connect(context.Background(), "primary", cfg)
	require.NoError(t, err)

	connector, err := manager.Get("primary")
	require.NoError(t, err)
	assert.NotNil(t, connector)

	_, err = manager.Get("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestManagerDisconnect tests the Manager.Disconnect method.
func TestManagerDisconnect(t *testing.T) {
	manager := NewManager()
	manager.RegisterFactory(&mockFactory{})

	cfg := &Config{
		Driver: "mysql",
		DSN:    "user:pass@tcp(localhost:3306)/db",
	}

	_, err := manager.Connect(context.Background(), "primary", cfg)
	require.NoError(t, err)

	err = manager.Disconnect(context.Background(), "primary")
	require.NoError(t, err)

	_, err = manager.Get("primary")
	require.Error(t, err)

	err = manager.Disconnect(context.Background(), "nonexistent")
	require.Error(t, err)
}

// TestManagerGetAll tests the Manager.GetAll method.
func TestManagerGetAll(t *testing.T) {
	manager := NewManager()
	manager.RegisterFactory(&mockFactory{})

	// Empty manager
	all := manager.GetAll()
	assert.Empty(t, all)

	// Add connections
	cfg := &Config{
		Driver: "mysql",
		DSN:    "user:pass@tcp(localhost:3306)/db",
	}

	_, err := manager.Connect(context.Background(), "primary", cfg)
	require.NoError(t, err)

	_, err = manager.Connect(context.Background(), "secondary", cfg)
	require.NoError(t, err)

	all = manager.GetAll()
	assert.Len(t, all, 2)
}

// TestManagerCloseAll tests the Manager.CloseAll method.
func TestManagerCloseAll(t *testing.T) {
	manager := NewManager()
	manager.RegisterFactory(&mockFactory{})

	cfg := &Config{
		Driver: "mysql",
		DSN:    "user:pass@tcp(localhost:3306)/db",
	}

	_, err := manager.Connect(context.Background(), "primary", cfg)
	require.NoError(t, err)

	_, err = manager.Connect(context.Background(), "secondary", cfg)
	require.NoError(t, err)

	err = manager.CloseAll(context.Background())
	require.NoError(t, err)

	all := manager.GetAll()
	assert.Empty(t, all)
}

// TestManagerList tests the Manager.List method.
func TestManagerList(t *testing.T) {
	manager := NewManager()
	manager.RegisterFactory(&mockFactory{})

	cfg := &Config{
		Driver: "mysql",
		DSN:    "user:pass@tcp(localhost:3306)/db",
	}

	_, err := manager.Connect(context.Background(), "primary", cfg)
	require.NoError(t, err)

	_, err = manager.Connect(context.Background(), "secondary", cfg)
	require.NoError(t, err)

	names := manager.List()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "primary")
	assert.Contains(t, names, "secondary")
}
