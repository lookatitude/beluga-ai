// Package prompts provides a standardized registry pattern for template engine creation.
// This follows the Beluga AI Framework design patterns with consistent factory interfaces.
package prompts

import (
	"context"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/prompts/iface"
)

// ProviderFactory defines the function signature for creating template engines.
// This type is used by providers to register themselves with the registry.
// The config parameter is `any` to avoid import cycles - providers should
// assert it to prompts.Config when implementing.
type ProviderFactory = iface.TemplateEngineFactory

// ProviderRegistry manages template engine provider registration and creation.
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
//	registry := prompts.GetRegistry()
//	providers := registry.ListProviders()
func GetRegistry() *ProviderRegistry {
	registryOnce.Do(func() {
		globalRegistry = &ProviderRegistry{
			providers: make(map[string]ProviderFactory),
		}
	})
	return globalRegistry
}

// Register registers a new template engine provider with the registry.
// This method is thread-safe and allows extending the framework with custom providers.
//
// Parameters:
//   - name: Unique identifier for the provider (e.g., "go", "mock")
//   - factory: Function that creates template engine instances for this provider
//
// Example:
//
//	registry.Register("custom", func(ctx context.Context, config any) (iface.TemplateEngine, error) {
//	    return NewCustomTemplateEngine(config)
//	})
func (r *ProviderRegistry) Register(name string, factory ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = factory
}

// Create creates a new template engine instance using the registered provider.
// This method is thread-safe and returns an error if the provider is not registered.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - name: Name of the provider to use (must be registered)
//   - config: Provider configuration (typically prompts.Config)
//
// Returns:
//   - iface.TemplateEngine: A new template engine instance
//   - error: Error if provider is not registered or creation fails
//
// Example:
//
//	engine, err := registry.Create(ctx, "go", config)
//	if err != nil {
//	    log.Fatal(err)
//	}
func (r *ProviderRegistry) Create(ctx context.Context, name string, config any) (iface.TemplateEngine, error) {
	r.mu.RLock()
	factory, exists := r.providers[name]
	r.mu.RUnlock()

	if !exists {
		return nil, NewProviderNotFoundError("create_template_engine", name)
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
//   - factory: Function that creates template engine instances
//
// Example:
//
//	prompts.Register("custom", customTemplateEngineFactory)
func Register(name string, factory ProviderFactory) {
	GetRegistry().Register(name, factory)
}

// Create creates a template engine using the global registry.
// This is a convenience function for creating template engines with the global registry.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - name: Name of the provider to use
//   - config: Provider configuration
//
// Returns:
//   - iface.TemplateEngine: A new template engine instance
//   - error: Error if provider is not registered or creation fails
//
// Example:
//
//	engine, err := prompts.Create(ctx, "go", config)
func Create(ctx context.Context, name string, config any) (iface.TemplateEngine, error) {
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
//	providers := prompts.ListProviders()
//	fmt.Printf("Available providers: %v\n", providers)
func ListProviders() []string {
	return GetRegistry().ListProviders()
}

// Ensure ProviderRegistry implements iface.TemplateRegistry.
var _ iface.TemplateRegistry = (*ProviderRegistry)(nil)
