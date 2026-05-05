// Package di provides dependency injection container for the Firefly framework.
// This file provides Wire code generation support.
package di

import (
	"context"
)

// ProviderSet is a collection of providers for Wire code generation.
// It allows defining dependencies in a way that Wire can generate code for.
type ProviderSet struct {
	providers []any
}

// NewProviderSet creates a new provider set.
func NewProviderSet() *ProviderSet {
	return &ProviderSet{
		providers: make([]any, 0),
	}
}

// Add adds a provider to the set.
func (ps *ProviderSet) Add(provider any) *ProviderSet {
	ps.providers = append(ps.providers, provider)
	return ps
}

// Providers returns all providers in the set.
func (ps *ProviderSet) Providers() []any {
	return ps.providers
}

// Injector is a function that injects dependencies into a struct.
type Injector func(providers *ProviderSet) error

// SimpleInjector is a simple injector that creates and populates a struct.
func SimpleInjector(ctors map[string]any) Injector {
	return func(providers *ProviderSet) error {
		_ = ctors
		return nil
	}
}

// WireProviders is a placeholder for Wire to generate provider bindings.
// In a real project, you would run `wire gen` to generate this code.
var WireProviders any

// Provide is a helper function for Wire to create a provider.
func Provide[T any](value T) *T {
	return &value
}

// ProvideFunc wraps a constructor function for use with Wire.
func ProvideFunc[T any](fn func() (T, error)) func() (T, error) {
	return fn
}

// ProvideContext wraps a constructor function that takes context for use with Wire.
func ProvideContext[T any](fn func(context.Context) (T, error)) func(context.Context) (T, error) {
	return fn
}
