package registry

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	registryiface "github.com/lookatitude/beluga-ai/pkg/embeddings/registry/iface"
)

// ProviderRegistry manages embedder provider registration and retrieval.
// This implements the registryiface.Registry interface.
type ProviderRegistry struct {
	creators map[string]registryiface.EmbedderFactory
	mu       sync.RWMutex
}

// Global registry instance for easy access.
var (
	globalRegistry *ProviderRegistry
	registryOnce   sync.Once
)

// GetRegistry returns the global registry instance.
// This follows the standard pattern used across all Beluga AI packages.
// It uses sync.Once to ensure thread-safe initialization.
func GetRegistry() *ProviderRegistry {
	registryOnce.Do(func() {
		globalRegistry = &ProviderRegistry{
			creators: make(map[string]registryiface.EmbedderFactory),
		}
	})
	return globalRegistry
}

// Register registers a new embedder provider with the factory.
func (f *ProviderRegistry) Register(name string, creator registryiface.EmbedderFactory) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.creators[name] = creator
}

// Create creates a new embedder instance using the registered provider.
func (f *ProviderRegistry) Create(ctx context.Context, name string, config any) (iface.Embedder, error) {
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

// IsRegistered checks if a provider is registered.
func (f *ProviderRegistry) IsRegistered(name string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, exists := f.creators[name]
	return exists
}
