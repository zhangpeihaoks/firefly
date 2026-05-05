// Package plugin provides the plugin system for the Firefly framework.
package plugin

import (
	"context"
	"errors"
	"testing"
)

// TestPluginConfig_PBT tests plugin configuration correctness (Property 30).
// Feature: backend-server-framework, Property 30: 插件配置正确性
type testPlugin struct {
	*BasePlugin
	config string
}

func (p *testPlugin) Init(ctx context.Context) error {
	p.config = "initialized"
	return nil
}

func (p *testPlugin) Start(ctx context.Context) error {
	p.config = "started"
	return nil
}

func (p *testPlugin) Stop(ctx context.Context) error {
	p.config = "stopped"
	return nil
}

func TestPluginConfig_PBT(t *testing.T) {
	t.Run("basic plugin registration", func(t *testing.T) {
		manager := NewPluginManager()

		p := &testPlugin{BasePlugin: NewBasePlugin("test")}
		err := manager.Register(p)
		if err != nil {
			t.Fatalf("failed to register plugin: %v", err)
		}

		// Verify plugin is registered
		retrieved, ok := manager.Get("test")
		if !ok {
			t.Error("plugin not found")
		}
		if retrieved.Name() != "test" {
			t.Errorf("expected name 'test', got %s", retrieved.Name())
		}
	})

	t.Run("plugin with dependencies", func(t *testing.T) {
		manager := NewPluginManager()

		// Register plugins with dependencies
		p1 := &testPlugin{BasePlugin: NewBasePlugin("plugin1")}
		p2 := &testPlugin{BasePlugin: NewBasePlugin("plugin2")}
		p3 := &testPlugin{BasePlugin: NewBasePlugin("plugin3")}

		manager.Register(p1, WithDependencies())
		manager.Register(p2, WithDependencies("plugin1"))
		manager.Register(p3, WithDependencies("plugin1", "plugin2"))

		// Verify plugins are registered
		if _, ok := manager.Get("plugin1"); !ok {
			t.Error("plugin1 not found")
		}
		if _, ok := manager.Get("plugin2"); !ok {
			t.Error("plugin2 not found")
		}
		if _, ok := manager.Get("plugin3"); !ok {
			t.Error("plugin3 not found")
		}
	})

	t.Run("plugin with priority", func(t *testing.T) {
		manager := NewPluginManager()

		p1 := &testPlugin{BasePlugin: NewBasePlugin("high")}
		p2 := &testPlugin{BasePlugin: NewBasePlugin("low")}
		p3 := &testPlugin{BasePlugin: NewBasePlugin("default")}

		manager.Register(p1, WithPriority(0))   // High priority (early)
		manager.Register(p2, WithPriority(100)) // Low priority (late)
		manager.Register(p3)                    // Default priority

		// All should be registered
		if _, ok := manager.Get("high"); !ok {
			t.Error("high priority plugin not found")
		}
		if _, ok := manager.Get("low"); !ok {
			t.Error("low priority plugin not found")
		}
		if _, ok := manager.Get("default"); !ok {
			t.Error("default priority plugin not found")
		}
	})

	t.Run("duplicate plugin registration", func(t *testing.T) {
		manager := NewPluginManager()

		p1 := &testPlugin{BasePlugin: NewBasePlugin("test")}
		p2 := &testPlugin{BasePlugin: NewBasePlugin("test")}

		manager.Register(p1)
		err := manager.Register(p2)

		if err == nil {
			t.Error("expected error for duplicate registration")
		}
	})
}

// TestPluginLifecycle_PBT tests plugin lifecycle (Property 31).
// Feature: backend-server-framework, Property 31: 插件生命周期
func TestPluginLifecycle_PBT(t *testing.T) {
	t.Run("init start stop lifecycle", func(t *testing.T) {
		manager := NewPluginManager()
		ctx := context.Background()

		p := &testPlugin{BasePlugin: NewBasePlugin("lifecycle")}
		manager.Register(p)

		// Init
		if err := manager.Init(ctx); err != nil {
			t.Fatalf("init failed: %v", err)
		}
		if p.config != "initialized" {
			t.Errorf("expected config 'initialized', got %s", p.config)
		}

		// Start
		if err := manager.Start(ctx); err != nil {
			t.Fatalf("start failed: %v", err)
		}
		if p.config != "started" {
			t.Errorf("expected config 'started', got %s", p.config)
		}

		// Stop
		if err := manager.Stop(ctx); err != nil {
			t.Fatalf("stop failed: %v", err)
		}
		if p.config != "stopped" {
			t.Errorf("expected config 'stopped', got %s", p.config)
		}
	})

	t.Run("init error propagation", func(t *testing.T) {
		manager := NewPluginManager()
		ctx := context.Background()

		errPlugin := &errorPlugin{BasePlugin: NewBasePlugin("error")}
		manager.Register(errPlugin)

		err := manager.Init(ctx)
		if err == nil {
			t.Error("expected error from failed init")
		}
	})

	t.Run("start error propagation", func(t *testing.T) {
		manager := NewPluginManager()
		ctx := context.Background()

		startErrPlugin := &startErrorPlugin{BasePlugin: NewBasePlugin("startError")}
		manager.Register(startErrPlugin)

		// Init first
		manager.Init(ctx)

		// Start should fail
		err := manager.Start(ctx)
		if err == nil {
			t.Error("expected error from failed start")
		}
	})

	t.Run("stop error handling", func(t *testing.T) {
		manager := NewPluginManager()
		ctx := context.Background()

		stopErrPlugin := &stopErrorPlugin{BasePlugin: NewBasePlugin("stopError")}
		manager.Register(stopErrPlugin)

		// Init and start first
		manager.Init(ctx)
		manager.Start(ctx)

		// Stop should report error but continue
		err := manager.Stop(ctx)
		if err == nil {
			t.Error("expected error from failed stop")
		}
	})
}

// TestPluginDependencyOrder_PBT tests that plugins are initialized in dependency order.
func TestPluginDependencyOrder_PBT(t *testing.T) {
	t.Run("dependency order", func(t *testing.T) {
		manager := NewPluginManager()
		ctx := context.Background()

		order := []string{}
		makeOrderedPlugin := func(name string, deps []string) *orderedPlugin {
			return &orderedPlugin{
				name:  name,
				deps:  deps,
				order: &order,
			}
		}

		// C depends on B, B depends on A
		manager.Register(makeOrderedPlugin("a", nil))
		manager.Register(makeOrderedPlugin("b", []string{"a"}))
		manager.Register(makeOrderedPlugin("c", []string{"b"}), WithDependencies("b", "a"))

		// Init
		err := manager.Init(ctx)
		if err != nil {
			t.Fatalf("init failed: %v", err)
		}

		// Verify order: a should come before b, b should come before c
		aIdx := -1
		bIdx := -1
		cIdx := -1
		for i, name := range order {
			switch name {
			case "a":
				aIdx = i
			case "b":
				bIdx = i
			case "c":
				cIdx = i
			}
		}

		if aIdx == -1 || bIdx == -1 || cIdx == -1 {
			t.Errorf("not all plugins initialized: %v", order)
		}

		if aIdx > bIdx || bIdx > cIdx {
			t.Errorf("plugins not in dependency order: %v", order)
		}
	})

	t.Run("circular dependency detection", func(t *testing.T) {
		manager := NewPluginManager()
		ctx := context.Background()

		// Create plugins with circular dependency
		p1 := &testPlugin{BasePlugin: NewBasePlugin("p1")}
		p2 := &testPlugin{BasePlugin: NewBasePlugin("p2")}

		manager.Register(p1, WithDependencies("p2"))
		manager.Register(p2, WithDependencies("p1"))

		err := manager.Init(ctx)
		if err == nil {
			t.Error("expected circular dependency error")
		}
	})

	t.Run("missing dependency detection", func(t *testing.T) {
		manager := NewPluginManager()
		ctx := context.Background()

		p := &testPlugin{BasePlugin: NewBasePlugin("p")}
		manager.Register(p, WithDependencies("nonexistent"))

		err := manager.Init(ctx)
		if err == nil {
			t.Error("expected missing dependency error")
		}
	})
}

// Helper types for testing

type errorPlugin struct {
	*BasePlugin
}

func (p *errorPlugin) Init(ctx context.Context) error {
	return errors.New("init error")
}

type startErrorPlugin struct {
	*BasePlugin
}

func (p *startErrorPlugin) Start(ctx context.Context) error {
	return errors.New("start error")
}

type stopErrorPlugin struct {
	*BasePlugin
}

func (p *stopErrorPlugin) Stop(ctx context.Context) error {
	return errors.New("stop error")
}

type orderedPlugin struct {
	name  string
	deps  []string
	order *[]string
}

func (p *orderedPlugin) Name() string { return p.name }
func (p *orderedPlugin) Init(ctx context.Context) error {
	*p.order = append(*p.order, p.name)
	return nil
}
func (p *orderedPlugin) Start(ctx context.Context) error { return nil }
func (p *orderedPlugin) Stop(ctx context.Context) error  { return nil }

func TestPluginList_PBT(t *testing.T) {
	manager := NewPluginManager()

	p1 := &testPlugin{BasePlugin: NewBasePlugin("plugin1")}
	p2 := &testPlugin{BasePlugin: NewBasePlugin("plugin2")}

	manager.Register(p1)
	manager.Register(p2)

	plugins := manager.List()
	if len(plugins) != 2 {
		t.Errorf("expected 2 plugins, got %d", len(plugins))
	}
}
