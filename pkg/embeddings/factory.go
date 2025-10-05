// Package embeddings provides a standardized factory pattern for embedder creation.
// This follows the Beluga AI Framework design patterns with consistent factory interfaces.
package embeddings

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
)

// Factory defines the interface for creating Embedder instances.
// This enables dependency injection and makes testing easier.
type Factory interface {
	// CreateEmbedder creates a new Embedder instance with the given configuration.
	// The config parameter contains provider-specific settings.
	CreateEmbedder(ctx context.Context, config Config) (iface.Embedder, error)
}

// ProviderRegistry is the global factory for creating embedder instances.
// It maintains a registry of available providers and their creation functions.
type ProviderRegistry struct {
	mu       sync.RWMutex
	creators map[string]func(ctx context.Context, config Config) (iface.Embedder, error)
}

// NewProviderRegistry creates a new ProviderRegistry instance.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		creators: make(map[string]func(ctx context.Context, config Config) (iface.Embedder, error)),
	}
}

// Register registers a new embedder provider with the factory.
func (f *ProviderRegistry) Register(name string, creator func(ctx context.Context, config Config) (iface.Embedder, error)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creators[name] = creator
}

// Create creates a new embedder instance using the registered provider.
func (f *ProviderRegistry) Create(ctx context.Context, name string, config Config) (iface.Embedder, error) {
	f.mu.RLock()
	creator, exists := f.creators[name]
	f.mu.RUnlock()

	if !exists {
		return nil, iface.WrapError(
			fmt.Errorf("embedder provider '%s' not found", name),
			iface.ErrCodeProviderNotFound,
			"unknown embedder provider: %s", name,
		)
	}
	return creator(ctx, config)
}

// ListProviders returns a list of all registered provider names.
func (f *ProviderRegistry) ListProviders() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.creators))
	for name := range f.creators {
		names = append(names, name)
	}
	return names
}

// Global factory instance for easy access
var globalRegistry = NewProviderRegistry()

// RegisterGlobal registers a provider with the global factory.
func RegisterGlobal(name string, creator func(ctx context.Context, config Config) (iface.Embedder, error)) {
	globalRegistry.Register(name, creator)
}

// NewEmbedder creates an embedder using the global factory.
func NewEmbedder(ctx context.Context, name string, config Config) (iface.Embedder, error) {
	return globalRegistry.Create(ctx, name, config)
}

// ListAvailableProviders returns all available providers from the global factory.
func ListAvailableProviders() []string {
	return globalRegistry.ListProviders()
}

// GetGlobalRegistry returns the global registry instance for advanced usage.
func GetGlobalRegistry() *ProviderRegistry {
	return globalRegistry
}
