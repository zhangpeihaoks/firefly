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
// Supported constructor signatures:
//   - func(...) T
//   - func(...) (T, error)
//   - func(...) (T, func())
//   - func(...) (T, func(), error)
//
// For singletons, use RegisterSingleton.
func (c *Container) Register(constructor any) error {
	ctorType := reflect.TypeOf(constructor)
	if ctorType.Kind() != reflect.Func {
		return fmt.Errorf("constructor must be a function")
	}

	if ctorType.NumOut() == 0 {
		return fmt.Errorf("constructor must return at least one value")
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
// Supported constructor signatures:
//   - func(...) T
//   - func(...) (T, error)
//   - func(...) (T, func())
//   - func(...) (T, func(), error)
func (c *Container) RegisterSingleton(constructor any) error {
	ctorType := reflect.TypeOf(constructor)
	if ctorType.Kind() != reflect.Func {
		return fmt.Errorf("constructor must be a function")
	}

	if ctorType.NumOut() == 0 {
		return fmt.Errorf("constructor must return at least one value")
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

// RegisterInterface registers a constructor that returns implementation type T
// under the interface type I. At registration time, a runtime check verifies
// that T satisfies I, providing a clear error message if not.
//
// When resolving by interface type I, the constructor is invoked and the
// returned T value (which implements I) is set as the resolved value.
//
// Example:
//
//	di.RegisterInterface[UserRepository, *MySQLRepo](container, func(db *sql.DB) *MySQLRepo { ... })
func RegisterInterface[I any, T any](c *Container, constructor any) error {
	ctorType := reflect.TypeOf(constructor)
	if ctorType.Kind() != reflect.Func {
		return fmt.Errorf("constructor must be a function")
	}
	if ctorType.NumOut() == 0 {
		return fmt.Errorf("constructor must return at least one value")
	}

	ifaceType := reflect.TypeOf((*I)(nil)).Elem()
	implType := reflect.TypeOf((*T)(nil)).Elem()

	// Runtime verification: T must implement I
	if ifaceType.Kind() == reflect.Interface && !implType.Implements(ifaceType) {
		return fmt.Errorf("type %s does not implement interface %s", implType, ifaceType)
	}

	c.providers[ifaceType] = &provider{
		constructor: constructor,
		isSingleton: false,
	}
	return nil
}

// RegisterSingletonInterface registers a singleton constructor that returns
// implementation type T under the interface type I. The constructor is called
// only once and the result is cached.
//
// A runtime check verifies that T satisfies I at registration time.
//
// Example:
//
//	di.RegisterSingletonInterface[ConfigProvider, *EnvConfig](container, func() *EnvConfig { ... })
func RegisterSingletonInterface[I any, T any](c *Container, constructor any) error {
	ctorType := reflect.TypeOf(constructor)
	if ctorType.Kind() != reflect.Func {
		return fmt.Errorf("constructor must be a function")
	}
	if ctorType.NumOut() == 0 {
		return fmt.Errorf("constructor must return at least one value")
	}

	ifaceType := reflect.TypeOf((*I)(nil)).Elem()
	implType := reflect.TypeOf((*T)(nil)).Elem()

	// Runtime verification: T must implement I
	if ifaceType.Kind() == reflect.Interface && !implType.Implements(ifaceType) {
		return fmt.Errorf("type %s does not implement interface %s", implType, ifaceType)
	}

	c.providers[ifaceType] = &provider{
		constructor: constructor,
		isSingleton: true,
	}
	c.singletons[ifaceType] = true
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
// Supported constructor signatures:
//   - func(...) T
//   - func(...) (T, error)
//   - func(...) (T, func())
//   - func(...) (T, func(), error)
func (c *Container) callConstructor(constructor any) (any, error) {
	ctorType := reflect.TypeOf(constructor)
	numParams := ctorType.NumIn()

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

	results := reflect.ValueOf(constructor).Call(args)
	numOut := len(results)

	// No return values
	if numOut == 0 {
		return nil, nil
	}

	// Parse return values based on known patterns
	// Pattern 1: (T) or (T, error) or (T, func()) or (T, func(), error)
	instance := results[0].Interface()
	if numOut == 1 {
		return instance, nil
	}

	// Check the second return value type
	secondIsError := isErrorType(results[1].Type())

	if numOut == 2 {
		if secondIsError {
			// Pattern: (T, error)
			if !results[1].IsNil() {
				return nil, results[1].Interface().(error)
			}
			return instance, nil
		}
		// Pattern: (T, func()) — cleanup function is returned but not tracked here
		return instance, nil
	}

	if numOut == 3 {
		// Pattern: (T, func(), error)
		if !results[2].IsNil() {
			err := results[2].Interface().(error)
			if err != nil {
				return nil, err
			}
		}
		return instance, nil
	}

	return instance, nil
}

// isErrorType checks if a reflect.Type implements the error interface.
func isErrorType(t reflect.Type) bool {
	return t.Implements(reflect.TypeOf((*error)(nil)).Elem())
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
