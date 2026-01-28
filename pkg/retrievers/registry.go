// Package retrievers provides a standardized registry pattern for retriever creation.
// This follows the Beluga AI Framework design patterns with consistent factory interfaces.
package retrievers

import (
	"context"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/retrievers/iface"
)

// ProviderFactory defines the function signature for creating retrievers.
// This type is used by providers to register themselves with the registry.
type ProviderFactory = iface.RetrieverFactory

// ProviderRegistry manages retriever provider registration and creation.
// It maintains a registry of available providers and their factory functions.
type ProviderRegistry struct {
	providers map[string]ProviderFactory
	mu        sync.RWMutex
}

// Global registry instance and sync.Once for thread-safe initialization.
var (
	globalRegistry *ProviderRegistry
	registryOnce   sync.Once
)

// GetRegistry returns the global registry instance.
// This follows the standard pattern used across all Beluga AI packages.
// It uses sync.Once to ensure thread-safe initialization.
//
// Example:
//
//	registry := retrievers.GetRegistry()
//	providers := registry.ListProviders()
func GetRegistry() *ProviderRegistry {
	registryOnce.Do(func() {
		globalRegistry = &ProviderRegistry{
			providers: make(map[string]ProviderFactory),
		}
	})
	return globalRegistry
}

// Register registers a new retriever provider with the registry.
// This method is thread-safe and allows extending the framework with custom providers.
//
// Parameters:
//   - name: Unique identifier for the provider (e.g., "vectorstore", "multiquery")
//   - factory: Function that creates retriever instances for this provider
//
// Example:
//
//	registry.Register("custom", func(ctx context.Context, config any) (core.Retriever, error) {
//	    return NewCustomRetriever(config)
//	})
func (r *ProviderRegistry) Register(name string, factory ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = factory
}

// Create creates a new retriever instance using the registered provider.
// This method is thread-safe and returns an error if the provider is not registered.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - name: Name of the provider to use (must be registered)
//   - config: Provider configuration (type depends on the provider)
//
// Returns:
//   - core.Retriever: A new retriever instance
//   - error: Error if provider is not registered or creation fails
//
// Example:
//
//	retriever, err := registry.Create(ctx, "vectorstore", config)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (r *ProviderRegistry) Create(ctx context.Context, name string, config any) (core.Retriever, error) {
	r.mu.RLock()
	factory, exists := r.providers[name]
	r.mu.RUnlock()

	if !exists {
		return nil, NewProviderNotFoundError("create_retriever", name)
	}

	return factory(ctx, config)
}

// ListProviders returns a list of all registered provider names.
// This method is thread-safe and returns an empty slice if no providers are registered.
//
// Returns:
//   - []string: Slice of registered provider names
//
// Example:
//
//	providers := registry.ListProviders()
//	fmt.Printf("Available providers: %v\n", providers)
func (r *ProviderRegistry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// IsRegistered checks if a provider is registered.
//
// Parameters:
//   - name: Name of the provider to check
//
// Returns:
//   - bool: True if the provider is registered, false otherwise
func (r *ProviderRegistry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.providers[name]
	return exists
}

// Global convenience functions for working with the default registry.

// Register registers a provider with the global registry.
// This is a convenience function for registering with the global registry.
//
// Parameters:
//   - name: Unique identifier for the provider
//   - factory: Function that creates retriever instances
//
// Example:
//
//	retrievers.Register("custom", customRetrieverFactory)
func Register(name string, factory ProviderFactory) {
	GetRegistry().Register(name, factory)
}

// Create creates a retriever using the global registry.
// This is a convenience function for creating retrievers with the global registry.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - name: Name of the provider to use
//   - config: Provider configuration
//
// Returns:
//   - core.Retriever: A new retriever instance
//   - error: Error if provider is not registered or creation fails
//
// Example:
//
//	retriever, err := retrievers.Create(ctx, "vectorstore", config)
func Create(ctx context.Context, name string, config any) (core.Retriever, error) {
	return GetRegistry().Create(ctx, name, config)
}

// ListProviders returns all available providers from the global registry.
// This is a convenience function for listing providers from the global registry.
//
// Returns:
//   - []string: Slice of available provider names
//
// Example:
//
//	providers := retrievers.ListProviders()
//	fmt.Printf("Available providers: %v\n", providers)
func ListProviders() []string {
	return GetRegistry().ListProviders()
}

// IsRegistered checks if a provider is registered in the global registry.
//
// Parameters:
//   - name: Name of the provider to check
//
// Returns:
//   - bool: True if the provider is registered, false otherwise
func IsRegistered(name string) bool {
	return GetRegistry().IsRegistered(name)
}

// Ensure ProviderRegistry implements iface.Registry.
var _ iface.Registry = (*ProviderRegistry)(nil)
