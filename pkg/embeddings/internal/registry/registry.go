// Package registry provides backward compatibility for provider registration.
// This package delegates to the root embeddings package's registry.
//
// Deprecated: Providers should migrate to using embeddings.Register() directly.
// This package exists for backward compatibility during the migration period.
package registry

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/embeddings"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
)

// ProviderRegistry wraps the root embeddings.ProviderRegistry.
// Deprecated: Use embeddings.ProviderRegistry directly.
type ProviderRegistry = embeddings.ProviderRegistry

// GetRegistry returns the global provider registry.
// This delegates to the root embeddings package.
// Deprecated: Use embeddings.GetRegistry() directly.
func GetRegistry() *ProviderRegistry {
	return embeddings.GetRegistry()
}

// Register adds a new provider to the registry.
// Deprecated: Use embeddings.Register() directly.
func Register(name string, factory iface.EmbedderFactory) {
	embeddings.Register(name, factory)
}

// Create creates an embedder using the registered provider.
// Deprecated: Use embeddings.Create() directly.
func Create(ctx context.Context, name string, config any) (iface.Embedder, error) {
	return embeddings.Create(ctx, name, config)
}

// ListProviders returns all registered provider names.
// Deprecated: Use embeddings.ListProviders() directly.
func ListProviders() []string {
	return embeddings.ListProviders()
}
