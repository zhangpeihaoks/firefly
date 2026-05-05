// Package plugin provides the plugin system for the Firefly framework.
package plugin

import (
	"context"
	"fmt"
)

// Plugin is the interface that all plugins must implement.
// It provides lifecycle management for plugins.
type Plugin interface {
	// Name returns the plugin name.
	Name() string
	// Init initializes the plugin.
	// This is called before Start.
	Init(ctx context.Context) error
	// Start starts the plugin.
	// This is called after all plugins are initialized.
	Start(ctx context.Context) error
	// Stop stops the plugin.
	// This is called when the application is shutting down.
	Stop(ctx context.Context) error
}

// PluginOption is a function that configures a plugin.
type PluginOption func(*PluginConfig)

// PluginConfig is the configuration for a plugin.
type PluginConfig struct {
	// Name is the plugin name.
	Name string
	// Dependencies lists the plugins that this plugin depends on.
	Dependencies []string
	// Priority is the plugin priority for ordering.
	// Lower values are initialized first.
	Priority int
	// Config is the plugin-specific configuration.
	Config any
}

// BasePlugin provides a base implementation of the Plugin interface.
// Embed this struct in your plugin to get default implementations.
type BasePlugin struct {
	name string
}

// NewBasePlugin creates a new base plugin with the given name.
func NewBasePlugin(name string) *BasePlugin {
	return &BasePlugin{name: name}
}

// Name returns the plugin name.
func (p *BasePlugin) Name() string {
	return p.name
}

// Init is a no-op by default.
func (p *BasePlugin) Init(ctx context.Context) error {
	return nil
}

// Start is a no-op by default.
func (p *BasePlugin) Start(ctx context.Context) error {
	return nil
}

// Stop is a no-op by default.
func (p *BasePlugin) Stop(ctx context.Context) error {
	return nil
}

// PluginManager manages plugin lifecycle and dependencies.
type PluginManager struct {
	plugins   map[string]Plugin
	dependsOn map[string][]string // plugin -> plugins it depends on
	priority  map[string]int
}

// NewPluginManager creates a new plugin manager.
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins:   make(map[string]Plugin),
		dependsOn: make(map[string][]string),
		priority:  make(map[string]int),
	}
}

// Register registers a plugin with the manager.
func (m *PluginManager) Register(p Plugin, opts ...PluginOption) error {
	name := p.Name()
	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	// Apply options
	cfg := &PluginConfig{
		Priority: 0,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	m.plugins[name] = p
	m.priority[name] = cfg.Priority
	m.dependsOn[name] = cfg.Dependencies

	return nil
}

// Init initializes all registered plugins in dependency order.
func (m *PluginManager) Init(ctx context.Context) error {
	// Get plugins in dependency order
	ordered, err := m.getOrderedPlugins()
	if err != nil {
		return fmt.Errorf("failed to order plugins: %w", err)
	}

	// Initialize each plugin
	for _, name := range ordered {
		p := m.plugins[name]
		if err := p.Init(ctx); err != nil {
			return fmt.Errorf("plugin %s failed to initialize: %w", name, err)
		}
	}

	return nil
}

// Start starts all registered plugins in dependency order.
func (m *PluginManager) Start(ctx context.Context) error {
	// Get plugins in dependency order
	ordered, err := m.getOrderedPlugins()
	if err != nil {
		return fmt.Errorf("failed to order plugins: %w", err)
	}

	// Start each plugin
	for _, name := range ordered {
		p := m.plugins[name]
		if err := p.Start(ctx); err != nil {
			return fmt.Errorf("plugin %s failed to start: %w", name, err)
		}
	}

	return nil
}

// Stop stops all registered plugins in reverse dependency order.
func (m *PluginManager) Stop(ctx context.Context) error {
	// Get plugins in reverse dependency order
	ordered, err := m.getOrderedPlugins()
	if err != nil {
		return fmt.Errorf("failed to order plugins: %w", err)
	}

	// Stop each plugin in reverse order
	for i := len(ordered) - 1; i >= 0; i-- {
		name := ordered[i]
		p := m.plugins[name]
		if err := p.Stop(ctx); err != nil {
			return fmt.Errorf("plugin %s failed to stop: %w", name, err)
		}
	}

	return nil
}

// getOrderedPlugins returns plugins in dependency order (dependencies first).
func (m *PluginManager) getOrderedPlugins() ([]string, error) {
	// Topological sort based on dependencies
	var ordered []string
	visited := make(map[string]bool)
	visiting := make(map[string]bool)

	var visit func(name string) error
	visit = func(name string) error {
		if visited[name] {
			return nil
		}
		if visiting[name] {
			return fmt.Errorf("circular dependency detected involving plugin %s", name)
		}

		visiting[name] = true

		// Visit dependencies first
		for _, dep := range m.dependsOn[name] {
			if _, exists := m.plugins[dep]; !exists {
				return fmt.Errorf("plugin %s depends on non-existent plugin %s", name, dep)
			}
			if err := visit(dep); err != nil {
				return err
			}
		}

		visiting[name] = false
		visited[name] = true
		ordered = append(ordered, name)
		return nil
	}

	// Sort by priority first (lower priority = earlier)
	var names []string
	for name := range m.plugins {
		names = append(names, name)
	}

	// Simple sort by priority, then name for stability
	for i := 0; i < len(names)-1; i++ {
		for j := i + 1; j < len(names); j++ {
			if m.priority[names[i]] > m.priority[names[j]] {
				names[i], names[j] = names[j], names[i]
			} else if m.priority[names[i]] == m.priority[names[j]] && names[i] > names[j] {
				// Sort by name for stability when priorities are equal
				names[i], names[j] = names[j], names[i]
			}
		}
	}

	// Visit all plugins
	for _, name := range names {
		if err := visit(name); err != nil {
			return nil, err
		}
	}

	return ordered, nil
}

// Get returns a registered plugin by name.
func (m *PluginManager) Get(name string) (Plugin, bool) {
	p, ok := m.plugins[name]
	return p, ok
}

// List returns all registered plugins.
func (m *PluginManager) List() []Plugin {
	plugins := make([]Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// WithDependencies sets the dependencies for a plugin.
func WithDependencies(deps ...string) PluginOption {
	return func(c *PluginConfig) {
		c.Dependencies = deps
	}
}

// WithPriority sets the priority for a plugin.
func WithPriority(priority int) PluginOption {
	return func(c *PluginConfig) {
		c.Priority = priority
	}
}

// WithConfig sets the configuration for a plugin.
func WithConfig(config any) PluginOption {
	return func(c *PluginConfig) {
		c.Config = config
	}
}
