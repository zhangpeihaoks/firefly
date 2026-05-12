// Package di provides dependency injection container for the Firefly framework.
package di

import (
	"testing"
)

// TestDependencyInjection_PBT tests dependency injection correctness (Property 27).
// Feature: backend-server-framework, Property 27: 依赖注入正确性
type ServiceA struct {
	Name string
}

type ServiceB struct {
	A *ServiceA
}

type ServiceC struct {
	B *ServiceB
}

func TestDependencyInjection_PBT(t *testing.T) {
	t.Run("simple constructor injection", func(t *testing.T) {
		container := New()

		// Register constructor for ServiceA
		err := container.Register(func() *ServiceA {
			return &ServiceA{Name: "ServiceA"}
		})
		if err != nil {
			t.Fatalf("failed to register ServiceA: %v", err)
		}

		// Resolve ServiceA
		var a *ServiceA
		err = container.Resolve(&a)
		if err != nil {
			t.Fatalf("failed to resolve ServiceA: %v", err)
		}

		if a == nil || a.Name != "ServiceA" {
			t.Errorf("expected ServiceA with Name='ServiceA', got %v", a)
		}
	})

	t.Run("constructor with dependency", func(t *testing.T) {
		container := New()

		// Register ServiceA first
		container.Register(func() *ServiceA {
			return &ServiceA{Name: "ServiceA"}
		})

		// Register ServiceB that depends on ServiceA
		err := container.Register(func(a *ServiceA) *ServiceB {
			return &ServiceB{A: a}
		})
		if err != nil {
			t.Fatalf("failed to register ServiceB: %v", err)
		}

		// Resolve ServiceB
		var b *ServiceB
		err = container.Resolve(&b)
		if err != nil {
			t.Fatalf("failed to resolve ServiceB: %v", err)
		}

		if b == nil || b.A == nil || b.A.Name != "ServiceA" {
			t.Errorf("expected ServiceB with ServiceA.Name='ServiceA', got %v", b)
		}
	})

	t.Run("singleton injection", func(t *testing.T) {
		container := New()

		// Register singleton
		err := container.RegisterSingleton(func() *ServiceA {
			return &ServiceA{Name: "Singleton"}
		})
		if err != nil {
			t.Fatalf("failed to register singleton: %v", err)
		}

		// Resolve twice
		var a1, a2 *ServiceA
		if err := container.Resolve(&a1); err != nil {
			t.Fatalf("failed to resolve: %v", err)
		}
		if err := container.Resolve(&a2); err != nil {
			t.Fatalf("failed to resolve: %v", err)
		}

		// Both should be the same instance
		if a1 != a2 {
			t.Errorf("singleton instances should be the same, got %p and %p", a1, a2)
		}
	})

	t.Run("instance registration", func(t *testing.T) {
		container := New()

		// Register instance directly
		instance := &ServiceA{Name: "Instance"}
		err := container.RegisterInstance(instance)
		if err != nil {
			t.Fatalf("failed to register instance: %v", err)
		}

		// Resolve
		var a *ServiceA
		err = container.Resolve(&a)
		if err != nil {
			t.Fatalf("failed to resolve: %v", err)
		}

		// Should be the same instance
		if a != instance {
			t.Errorf("expected same instance, got %p and %p", a, instance)
		}
	})
}

// TestCircularDependencyDetection_PBT tests circular dependency detection (Property 28).
// Feature: backend-server-framework, Property 28: 循环依赖检测
type CircularA struct {
	B *CircularB
}

type CircularB struct {
	A *CircularA
}

func TestCircularDependencyDetection_PBT(t *testing.T) {
	t.Run("circular dependency detection", func(t *testing.T) {
		container := New()

		// Register CircularA (depends on B)
		container.Register(func(b *CircularB) *CircularA {
			return &CircularA{B: b}
		})

		// Register CircularB (depends on A)
		container.Register(func(a *CircularA) *CircularB {
			return &CircularB{A: a}
		})

		// Try to resolve - should fail
		var a *CircularA
		err := container.Resolve(&a)
		if err == nil {
			t.Error("expected error for circular dependency, got nil")
		}
	})
}

// TestComponentLifecycle_PBT tests component lifecycle (Property 29).
// Feature: backend-server-framework, Property 29: 组件生命周期
func TestComponentLifecycle_PBT(t *testing.T) {
	t.Run("singleton lifecycle", func(t *testing.T) {
		container := New()

		var instanceCount int
		constructor := func() *ServiceA {
			instanceCount++
			return &ServiceA{Name: "Singleton"}
		}

		// Register as singleton
		err := container.RegisterSingleton(constructor)
		if err != nil {
			t.Fatalf("failed to register: %v", err)
		}

		// Resolve multiple times
		for i := 0; i < 5; i++ {
			var s *ServiceA
			if err := container.Resolve(&s); err != nil {
				t.Fatalf("failed to resolve: %v", err)
			}
		}

		// Should only create one instance
		if instanceCount != 1 {
			t.Errorf("expected 1 instance, got %d", instanceCount)
		}
	})

	t.Run("transient lifecycle", func(t *testing.T) {
		container := New()

		var instanceCount int
		constructor := func() *ServiceA {
			instanceCount++
			return &ServiceA{Name: "Transient"}
		}

		// Register as transient (not singleton)
		err := container.Register(constructor)
		if err != nil {
			t.Fatalf("failed to register: %v", err)
		}

		// Resolve multiple times
		for i := 0; i < 5; i++ {
			var s *ServiceA
			if err := container.Resolve(&s); err != nil {
				t.Fatalf("failed to resolve: %v", err)
			}
		}

		// Should create new instance each time
		if instanceCount != 5 {
			t.Errorf("expected 5 instances, got %d", instanceCount)
		}
	})

	t.Run("invalidate singleton", func(t *testing.T) {
		container := New()

		var instanceCount int
		constructor := func() *ServiceA {
			instanceCount++
			return &ServiceA{Name: "ServiceA"}
		}

		container.RegisterSingleton(constructor)

		// Resolve twice (should get same instance)
		var s1 *ServiceA
		container.Resolve(&s1)
		var s2 *ServiceA
		container.Resolve(&s2)

		if s1 != s2 {
			t.Error("expected same instance before invalidation")
		}

		// Invalidate and resolve again
		container.Invalidate(&ServiceA{})
		var s3 *ServiceA
		container.Resolve(&s3)

		if s1 == s3 {
			t.Error("expected different instance after invalidation")
		}

		// Should have created 2 instances total
		if instanceCount != 2 {
			t.Errorf("expected 2 instances, got %d", instanceCount)
		}
	})
}

// TestWireIntegration_PBT tests Wire integration support (Properties 27, 28, 29).
// Feature: backend-server-framework, Properties 27, 28, 29
func TestWireIntegration_PBT(t *testing.T) {
	t.Run("provider set creation", func(t *testing.T) {
		// Test that ProviderSet can be created
		ps := NewProviderSet()
		if ps == nil {
			t.Error("expected non-nil ProviderSet")
		}

		// Add providers
		ps.Add(func() *ServiceA { return &ServiceA{Name: "test"} })
		ps.Add("string provider")

		providers := ps.Providers()
		if len(providers) != 2 {
			t.Errorf("expected 2 providers, got %d", len(providers))
		}
	})

	t.Run("provide helper function", func(t *testing.T) {
		// Test the Provide helper
		s := ServiceA{Name: "test"}
		p := new(s)
		if p == nil || p.Name != "test" {
			t.Errorf("expected Provide to return pointer to value")
		}
	})
}

// Test types for interface binding tests
type testGreeter interface {
	Greet() string
}

type englishGreeter struct{
	id int
}
var englishCounter int
func (e *englishGreeter) Greet() string { return "Hello" }

type chineseGreeter struct{}
func (c *chineseGreeter) Greet() string { return "你好" }

type greeterWithDep struct {
	a *ServiceA
}
func (g *greeterWithDep) Greet() string { return "Hello with " + g.a.Name }

// TestInterfaceBinding tests RegisterInterface and RegisterSingletonInterface.
func TestInterfaceBinding(t *testing.T) {

	t.Run("interface binding with RegisterInterface", func(t *testing.T) {
		container := New()

		// Register implementation under interface
		err := RegisterInterface[testGreeter, *englishGreeter](container, func() *englishGreeter {
			englishCounter++
			return &englishGreeter{id: englishCounter}
		})
		if err != nil {
			t.Fatalf("RegisterInterface failed: %v", err)
		}

		// Resolve by interface type
		var g testGreeter
		if err := container.Resolve(&g); err != nil {
			t.Fatalf("failed to resolve Greeter: %v", err)
		}

		if g == nil {
			t.Fatal("resolved Greeter is nil")
		}
		if g.Greet() != "Hello" {
			t.Errorf("expected 'Hello', got %q", g.Greet())
		}
	})

	t.Run("interface binding creates new instance each time", func(t *testing.T) {
		container := New()

		RegisterInterface[testGreeter, *englishGreeter](container, func() *englishGreeter {
			englishCounter++
			return &englishGreeter{id: englishCounter}
		})

		var g1, g2 testGreeter
		container.Resolve(&g1)
		container.Resolve(&g2)

		// Transient: should be different instances
		if g1 == g2 {
			t.Error("expected different instances for non-singleton RegisterInterface")
		}
	})

	t.Run("singleton interface binding", func(t *testing.T) {
		container := New()

		RegisterSingletonInterface[testGreeter, *chineseGreeter](container, func() *chineseGreeter {
			return &chineseGreeter{}
		})

		var g1, g2 testGreeter
		container.Resolve(&g1)
		container.Resolve(&g2)

		if g1 == nil || g2 == nil {
			t.Fatal("resolved Greeter is nil")
		}
		if g1 != g2 {
			t.Error("expected same instance for singleton")
		}
		if g1.Greet() != "你好" {
			t.Errorf("expected '你好', got %q", g1.Greet())
		}
	})

	t.Run("RegisterInterface rejects incompatible types", func(t *testing.T) {
		container := New()

		// *ServiceA does not implement Greeter
		err := RegisterInterface[testGreeter, *ServiceA](container, func() *ServiceA {
			return &ServiceA{Name: "test"}
		})
		if err == nil {
			t.Error("expected error for incompatible types, got nil")
		}
	})

	t.Run("interface binding with constructor dependencies", func(t *testing.T) {
		container := New()

		// Register a dependency
		container.Register(func() *ServiceA {
			return &ServiceA{Name: "dep"}
		})

		// Interface binding with constructor that depends on ServiceA
		RegisterInterface[testGreeter, *greeterWithDep](container, func(a *ServiceA) *greeterWithDep {
			return &greeterWithDep{a: a}
		})

		var g testGreeter
		if err := container.Resolve(&g); err != nil {
			t.Fatalf("failed to resolve Greeter with deps: %v", err)
		}
		if g == nil {
			t.Fatal("resolved Greeter is nil")
		}
	})
}
