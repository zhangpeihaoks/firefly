// Package database provides database connection management for the Firefly framework.
// This file implements the database manager for managing multiple connections.
package database

import (
	"context"
	"fmt"
	"sync"

	"log/slog"
)

// Manager manages multiple database connections.
type Manager struct {
	connections map[string]Connector
	configs     map[string]*Config
	factories   map[DatabaseType]Factory
	logger      *slog.Logger
	mu          sync.RWMutex
}

// ManagerOption is a function that configures the Manager.
type ManagerOption func(*Manager)

// NewManager creates a new database manager.
func NewManager(opts ...ManagerOption) *Manager {
	m := &Manager{
		connections: make(map[string]Connector),
		configs:     make(map[string]*Config),
		factories:   make(map[DatabaseType]Factory),
		logger:      slog.Default(),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// WithLogger sets the logger for the manager.
func WithLogger(logger *slog.Logger) ManagerOption {
	return func(m *Manager) {
		m.logger = logger
	}
}

// RegisterFactory registers a database factory.
func (m *Manager) RegisterFactory(factory Factory) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.factories[factory.Type()] = factory
}

// Connect creates and establishes a connection with the given name and configuration.
func (m *Manager) Connect(ctx context.Context, name string, cfg *Config) (Connector, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if connection already exists
	if _, exists := m.connections[name]; exists {
		return nil, fmt.Errorf("database connection %q already exists", name)
	}

	// Get factory for the driver type
	dbType := DatabaseType(cfg.Driver)
	factory, exists := m.factories[dbType]
	if !exists {
		return nil, fmt.Errorf("no factory registered for database type %q", dbType)
	}

	// Create connector
	connector, err := factory.Create(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection %q: %w", name, err)
	}

	// Store connection and config
	m.connections[name] = connector
	m.configs[name] = cfg

	m.logger.Info("database connection established",
		"name", name,
		"driver", cfg.Driver,
	)

	return connector, nil
}

// Disconnect closes the connection with the given name.
func (m *Manager) Disconnect(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	connector, exists := m.connections[name]
	if !exists {
		return fmt.Errorf("database connection %q not found", name)
	}

	if err := connector.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect database %q: %w", name, err)
	}

	delete(m.connections, name)
	delete(m.configs, name)

	m.logger.Info("database connection closed", "name", name)
	return nil
}

// Get returns the connection with the given name.
func (m *Manager) Get(name string) (Connector, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	connector, exists := m.connections[name]
	if !exists {
		return nil, fmt.Errorf("database connection %q not found", name)
	}
	return connector, nil
}

// GetDB returns the SQL database connection with the given name.
func (m *Manager) GetDB(name string) (DB, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	connector, exists := m.connections[name]
	if !exists {
		return nil, fmt.Errorf("database connection %q not found", name)
	}

	db, ok := connector.(DB)
	if !ok {
		return nil, fmt.Errorf("database connection %q is not a SQL database", name)
	}
	return db, nil
}

// GetAll returns all connections.
func (m *Manager) GetAll() map[string]Connector {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]Connector, len(m.connections))
	for k, v := range m.connections {
		result[k] = v
	}
	return result
}

// CloseAll closes all connections.
func (m *Manager) CloseAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name, connector := range m.connections {
		if err := connector.Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to close %q: %w", name, err))
		}
		m.logger.Info("database connection closed", "name", name)
	}

	// Clear connections and configs
	m.connections = make(map[string]Connector)
	m.configs = make(map[string]*Config)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing database connections: %v", errs)
	}
	return nil
}

// Reload disconnects all connections and reconnects them using their stored
// configurations. This is designed to be called from a config change callback
// to hot-reload database connections without restarting the application.
func (m *Manager) Reload(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("reloading all database connections")

	// Close all existing connections (but keep configs for reconnection)
	var errs []error
	for name, connector := range m.connections {
		if err := connector.Disconnect(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to close %q: %w", name, err))
		}
	}

	// Clear connections map
	m.connections = make(map[string]Connector)

	// Reconnect all using stored configs
	for name, cfg := range m.configs {
		dbType := DatabaseType(cfg.Driver)
		factory, exists := m.factories[dbType]
		if !exists {
			errs = append(errs, fmt.Errorf("no factory registered for database type %q", dbType))
			continue
		}

		connector, err := factory.Create(cfg)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to reconnect %q: %w", name, err))
			continue
		}

		m.connections[name] = connector
		m.logger.Info("database connection re-established", "name", name, "driver", cfg.Driver)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during database reload: %v", errs)
	}
	return nil
}

// CheckHealth checks the health of all connections.
func (m *Manager) CheckHealth(ctx context.Context) map[string]*HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]*HealthStatus)
	for name, connector := range m.connections {
		if checker, ok := connector.(HealthChecker); ok {
			results[name] = checker.CheckHealth(ctx)
		} else {
			results[name] = &HealthStatus{
				Status:  "unknown",
				Message: "connector does not support health checking",
			}
		}
	}
	return results
}

// StoreConfig stores a database configuration without establishing a connection.
// The connection will be established when Start is called.
// This is useful for deferring connection establishment to application startup.
func (m *Manager) StoreConfig(name string, cfg *Config) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.configs[name] = cfg
	m.logger.Info("database config stored", "name", name, "driver", cfg.Driver)
}

// Start implements the app.Lifecycle interface. It connects to all stored
// configurations that do not yet have active connections.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name, cfg := range m.configs {
		// Skip if already connected
		if _, exists := m.connections[name]; exists {
			continue
		}

		dbType := DatabaseType(cfg.Driver)
		factory, exists := m.factories[dbType]
		if !exists {
			errs = append(errs, fmt.Errorf("no factory registered for database type %q", dbType))
			continue
		}

		connector, err := factory.Create(cfg)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to connect %q: %w", name, err))
			continue
		}

		m.connections[name] = connector
		m.logger.Info("database connection established at startup", "name", name, "driver", cfg.Driver)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors starting database manager: %v", errs)
	}
	return nil
}

// Stop implements the app.Lifecycle interface. It closes all active database connections.
func (m *Manager) Stop(ctx context.Context) error {
	return m.CloseAll(ctx)
}

// List returns the names of all connections.
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.connections))
	for name := range m.connections {
		names = append(names, name)
	}
	return names
}
