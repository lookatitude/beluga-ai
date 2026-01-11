// Package registry provides a global registry for multimodal provider management.
package registry

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/multimodal/iface"
	registryiface "github.com/lookatitude/beluga-ai/pkg/multimodal/registry/iface"
)

// ProviderRegistry manages multimodal provider registration and retrieval.
type ProviderRegistry struct {
	creators map[string]registryiface.MultimodalModelFactory
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
			creators: make(map[string]registryiface.MultimodalModelFactory),
		}
	})
	return globalRegistry
}

// Register registers a new multimodal provider with the registry.
// Uses any for config to avoid import cycles with provider packages.
func (r *ProviderRegistry) Register(name string, creator registryiface.MultimodalModelFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.creators[name] = creator
}

// Create creates a new multimodal model instance using the registered provider.
func (r *ProviderRegistry) Create(ctx context.Context, name string, config any) (iface.MultimodalModel, error) {
	r.mu.RLock()
	creator, exists := r.creators[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("multimodal provider '%s' not found", name)
	}

	return creator(ctx, config)
}

// ListProviders returns a list of all registered provider names.
func (r *ProviderRegistry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.creators))
	for name := range r.creators {
		names = append(names, name)
	}
	return names
}

// IsRegistered checks if a provider is registered.
func (r *ProviderRegistry) IsRegistered(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.creators[name]
	return exists
}
