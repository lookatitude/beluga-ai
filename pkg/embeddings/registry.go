package embeddings

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/internal/registry"
)

// ProviderRegistry manages embedder provider registration and retrieval.
// This wraps the internal registry to provide a stable public API.
type ProviderRegistry = registry.ProviderRegistry

// GetRegistry returns the global registry instance.
// This follows the standard pattern used across all Beluga AI packages.
func GetRegistry() *ProviderRegistry {
	return registry.GetRegistry()
}

// Register registers a new embedder provider with the factory.
// This is a convenience wrapper around GetRegistry().Register().
func Register(name string, creator iface.EmbedderFactory) {
	GetRegistry().Register(name, creator)
}

// Create creates a new embedder instance using the registered provider.
// This is a convenience wrapper around GetRegistry().Create().
func Create(ctx context.Context, name string, config any) (iface.Embedder, error) {
	return GetRegistry().Create(ctx, name, config)
}

// ListProviders returns a list of all registered provider names.
// This is a convenience wrapper around GetRegistry().ListProviders().
func ListProviders() []string {
	return GetRegistry().ListProviders()
}
