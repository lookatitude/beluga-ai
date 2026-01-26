// Package vectorstores provides registry for vector store providers.
package vectorstores

import (
	"context"
	"fmt"
	"sync"

	vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

// Registry manages vector store provider registration and retrieval.
// It is thread-safe and follows the standard Beluga AI registry pattern.
type Registry struct {
	factories map[string]func(ctx context.Context, config vectorstoresiface.Config) (VectorStore, error)
	mu        sync.RWMutex
}

// Global registry instance.
var (
	globalRegistry *Registry
	registryOnce   sync.Once
)

// GetGlobalRegistry returns the global registry instance.
// This is the preferred way to access the registry.
func GetGlobalRegistry() *Registry {
	registryOnce.Do(func() {
		globalRegistry = NewRegistry()
	})
	return globalRegistry
}

// NewRegistry creates a new Registry instance.
// Use GetGlobalRegistry() for most cases; this is mainly for testing.
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]func(ctx context.Context, config vectorstoresiface.Config) (VectorStore, error)),
	}
}

// Register registers a new vector store provider with the registry.
// The name should match the provider's identifier (e.g., "inmemory", "pgvector").
func (r *Registry) Register(name string, factory func(ctx context.Context, config vectorstoresiface.Config) (VectorStore, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// Create creates a new vector store instance using the registered provider.
// Returns an error if the provider is not registered or creation fails.
func (r *Registry) Create(ctx context.Context, name string, config vectorstoresiface.Config) (VectorStore, error) {
	r.mu.RLock()
	factory, exists := r.factories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, NewVectorStoreErrorWithMessage(
			"Create",
			ErrCodeProviderNotFound,
			fmt.Sprintf("vector store provider '%s' not registered", name),
			nil,
		)
	}

	return factory(ctx, config)
}

// GetProvider is an alias for Create to match the standard pattern.
func (r *Registry) GetProvider(name string, config vectorstoresiface.Config) (VectorStore, error) {
	return r.Create(context.Background(), name, config)
}

// ListProviders returns a list of all registered provider names.
func (r *Registry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// IsRegistered checks if a provider is registered.
func (r *Registry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.factories[name]
	return exists
}

// ProviderCount returns the number of registered providers.
func (r *Registry) ProviderCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.factories)
}

// Unregister removes a provider from the registry.
// This is mainly useful for testing.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.factories, name)
}

// Clear removes all providers from the registry.
// This is mainly useful for testing.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories = make(map[string]func(ctx context.Context, config vectorstoresiface.Config) (VectorStore, error))
}
