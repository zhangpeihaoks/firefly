// Package di provides dependency injection container for the Firefly framework.
package di

import (
	"fmt"
	"reflect"
)

// Container is a simple dependency injection container.
// It supports constructor injection, interface injection, and lifecycle management.
type Container struct {
	providers   map[reflect.Type]*provider
	resolved    map[reflect.Type]any
	singletons  map[reflect.Type]bool
	invalidated map[reflect.Type]bool
	resolving   map[reflect.Type]bool // Track types currently being resolved for cycle detection
}

// provider holds information about a registered provider.
type provider struct {
	constructor any
	instance    any
	isSingleton bool
}

// New creates a new dependency injection container.
func New() *Container {
	return &Container{
		providers:   make(map[reflect.Type]*provider),
		resolved:    make(map[reflect.Type]any),
		singletons:  make(map[reflect.Type]bool),
		invalidated: make(map[reflect.Type]bool),
		resolving:   make(map[reflect.Type]bool),
	}
}

// Register registers a constructor for a type.
// The constructor should be a function that returns a single value.
// For singletons, use RegisterSingleton.
func (c *Container) Register(constructor any) error {
	ctorType := reflect.TypeOf(constructor)
	if ctorType.Kind() != reflect.Func {
		return fmt.Errorf("constructor must be a function")
	}

	if ctorType.NumOut() != 1 {
		return fmt.Errorf("constructor must return exactly one value")
	}

	outType := ctorType.Out(0)
	c.providers[outType] = &provider{
		constructor: constructor,
		isSingleton: false,
	}

	return nil
}

// RegisterSingleton registers a singleton constructor for a type.
// The constructor is called only once and the result is cached.
func (c *Container) RegisterSingleton(constructor any) error {
	ctorType := reflect.TypeOf(constructor)
	if ctorType.Kind() != reflect.Func {
		return fmt.Errorf("constructor must be a function")
	}

	if ctorType.NumOut() != 1 {
		return fmt.Errorf("constructor must return exactly one value")
	}

	outType := ctorType.Out(0)
	c.providers[outType] = &provider{
		constructor: constructor,
		isSingleton: true,
	}
	c.singletons[outType] = true

	return nil
}

// RegisterInstance registers an existing instance as a singleton.
func (c *Container) RegisterInstance(instance any) error {
	if instance == nil {
		return fmt.Errorf("instance cannot be nil")
	}

	outType := reflect.TypeOf(instance)
	c.providers[outType] = &provider{
		instance:    instance,
		isSingleton: true,
	}
	c.singletons[outType] = true

	return nil
}

// Resolve resolves a type from the container.
// It calls the registered constructor and injects dependencies.
func (c *Container) Resolve(target any) error {
	targetType := reflect.TypeOf(target)
	if targetType.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	elemType := targetType.Elem()
	provider, exists := c.providers[elemType]
	if !exists {
		return fmt.Errorf("no provider registered for %s", elemType)
	}

	// If we have a direct instance (from RegisterInstance), use it
	if provider.instance != nil {
		if provider.isSingleton {
			c.resolved[elemType] = provider.instance
		}
		reflect.ValueOf(target).Elem().Set(reflect.ValueOf(provider.instance))
		return nil
	}

	// Check if it's already resolved singleton
	if provider.isSingleton && !c.invalidated[elemType] {
		if resolved, ok := c.resolved[elemType]; ok {
			reflect.ValueOf(target).Elem().Set(reflect.ValueOf(resolved))
			return nil
		}
	}

	// Call constructor with dependency injection
	instance, err := c.callConstructor(provider.constructor)
	if err != nil {
		return fmt.Errorf("failed to resolve %s: %w", elemType, err)
	}

	// Cache if singleton
	if provider.isSingleton {
		c.resolved[elemType] = instance
	}

	reflect.ValueOf(target).Elem().Set(reflect.ValueOf(instance))
	return nil
}

// callConstructor calls a constructor with dependency injection.
func (c *Container) callConstructor(constructor any) (any, error) {
	ctorType := reflect.TypeOf(constructor)
	numParams := ctorType.NumIn()

	// If no parameters, just call the constructor
	if numParams == 0 {
		return reflect.ValueOf(constructor).Call(nil)[0].Interface(), nil
	}

	// Build arguments with dependency injection
	args := make([]reflect.Value, numParams)
	for i := 0; i < numParams; i++ {
		paramType := ctorType.In(i)
		arg, err := c.resolveDependency(paramType)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve dependency %s: %w", paramType, err)
		}
		args[i] = reflect.ValueOf(arg)
	}

	result := reflect.ValueOf(constructor).Call(args)
	if len(result) == 0 || result[0].IsNil() {
		// If there's an error, return it
		if len(result) > 1 && !result[1].IsNil() {
			return nil, result[1].Interface().(error)
		}
		return nil, fmt.Errorf("constructor returned nil")
	}

	return result[0].Interface(), nil
}

// resolveDependency resolves a single dependency.
func (c *Container) resolveDependency(paramType reflect.Type) (any, error) {
	// Check if provider exists
	provider, exists := c.providers[paramType]
	if !exists {
		return nil, fmt.Errorf("no provider registered for %s", paramType)
	}

	// Check for circular dependency - if we're already resolving this type
	if c.resolving[paramType] {
		return nil, fmt.Errorf("circular dependency detected for type %s", paramType)
	}

	// Check if it's already resolved singleton
	if provider.isSingleton && !c.invalidated[paramType] {
		if resolved, ok := c.resolved[paramType]; ok {
			return resolved, nil
		}
	}

	// Mark as resolving
	c.resolving[paramType] = true
	defer func() { c.resolving[paramType] = false }()

	// Resolve the dependency
	instance, err := c.callConstructor(provider.constructor)
	if err != nil {
		return nil, err
	}

	// Cache if singleton
	if provider.isSingleton {
		c.resolved[paramType] = instance
	}

	return instance, nil
}

// Invalidate removes a singleton instance from the container.
// The next Resolve call will create a new instance.
func (c *Container) Invalidate(targetType any) error {
	t := reflect.TypeOf(targetType)
	if c.singletons[t] {
		c.invalidated[t] = true
		delete(c.resolved, t)
	}
	return nil
}

// Clear removes all registered providers and resolved instances.
func (c *Container) Clear() {
	c.providers = make(map[reflect.Type]*provider)
	c.resolved = make(map[reflect.Type]any)
	c.singletons = make(map[reflect.Type]bool)
	c.invalidated = make(map[reflect.Type]bool)
}
