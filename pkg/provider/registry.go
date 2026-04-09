package provider

import (
	"context"
	"fmt"
	"sync"
)

// ProviderFactory is a function that creates a StorageProvider instance
type ProviderFactory func() (StorageProvider, error)

// Registry manages storage provider registration and lookup
type Registry struct {
	mu         sync.RWMutex
	providers  map[string]ProviderFactory
	instances  map[string]StorageProvider
}

// defaultRegistry is the global provider registry
var defaultRegistry = &Registry{
	providers: make(map[string]ProviderFactory),
	instances: make(map[string]StorageProvider),
}

// Register registers a provider factory for a scheme
func (r *Registry) Register(scheme string, factory ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[scheme] = factory
}

// Unregister removes a provider from the registry
func (r *Registry) Unregister(scheme string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.providers, scheme)
	delete(r.instances, scheme)
}

// Get returns a provider instance for the given scheme
// Returns singleton instance if already created
func (r *Registry) Get(ctx context.Context, scheme string) (StorageProvider, error) {
	r.mu.RLock()
	if p, ok := r.instances[scheme]; ok {
		r.mu.RUnlock()
		return p, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if p, ok := r.instances[scheme]; ok {
		return p, nil
	}

	factory, ok := r.providers[scheme]
	if !ok {
		return nil, fmt.Errorf("no provider registered for scheme: %s", scheme)
	}

	provider, err := factory()
	if err != nil {
		return nil, fmt.Errorf("failed to create provider for %s: %w", scheme, err)
	}

	r.instances[scheme] = provider
	return provider, nil
}

// SupportedSchemes returns all registered scheme names
func (r *Registry) SupportedSchemes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, 0, len(r.providers))
	for s := range r.providers {
		result = append(result, s)
	}
	return result
}

// ClearInstances removes all cached provider instances
// Useful for testing or re-initialization
func (r *Registry) ClearInstances() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.instances = make(map[string]StorageProvider)
}

// Default returns the global registry instance
func Default() *Registry {
	return defaultRegistry
}

// Register is a convenience function to register a provider with the default registry
func Register(scheme string, factory ProviderFactory) {
	defaultRegistry.Register(scheme, factory)
}

// Get is a convenience function to get a provider from the default registry
func Get(ctx context.Context, scheme string) (StorageProvider, error) {
	return defaultRegistry.Get(ctx, scheme)
}

// SupportedSchemes returns all supported schemes from the default registry
func SupportedSchemes() []string {
	return defaultRegistry.SupportedSchemes()
}
