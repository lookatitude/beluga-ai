// Package registry provides the internal registry implementation for embeddings.
// This is separated from the main embeddings package to avoid import cycles
// when providers need to register themselves.
package registry

import (
	"context"
	"errors"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
)

// ProviderRegistry manages embedder provider registration and creation.
type ProviderRegistry struct {
	creators map[string]iface.EmbedderFactory
	mu       sync.RWMutex
}

// Global instance.
var globalRegistry = &ProviderRegistry{
	creators: make(map[string]iface.EmbedderFactory),
}

// GetRegistry returns the global provider registry.
func GetRegistry() *ProviderRegistry {
	return globalRegistry
}

// Register adds a new provider to the registry.
func (r *ProviderRegistry) Register(name string, creator iface.EmbedderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.creators[name] = creator
}

// Create creates an embedder using the registered provider.
func (r *ProviderRegistry) Create(ctx context.Context, name string, config any) (iface.Embedder, error) {
	r.mu.RLock()
	creator, exists := r.creators[name]
	r.mu.RUnlock()

	if !exists {
		return nil, iface.WrapError(
			errors.New("provider not found: "+name),
			iface.ErrCodeProviderNotFound,
			"provider not found",
		)
	}

	return creator(ctx, config)
}

// ListProviders returns all registered provider names.
func (r *ProviderRegistry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]string, 0, len(r.creators))
	for name := range r.creators {
		providers = append(providers, name)
	}
	return providers
}
