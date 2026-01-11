// Package embeddings provides a standardized factory pattern for embedder creation.
// This follows the Beluga AI Framework design patterns with consistent factory interfaces.
package embeddings

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/registry"
	registryiface "github.com/lookatitude/beluga-ai/pkg/embeddings/registry/iface"
)

// Factory defines the interface for creating Embedder instances.
// This enables dependency injection and makes testing easier.
type Factory interface {
	// CreateEmbedder creates a new Embedder instance with the given configuration.
	// The config parameter contains provider-specific settings.
	CreateEmbedder(ctx context.Context, config Config) (iface.Embedder, error)
}

// ProviderRegistry is kept for backward compatibility but delegates to the new registry.
// Deprecated: Use registry.GetRegistry() directly or embeddings.GetRegistry().
type ProviderRegistry = registry.ProviderRegistry

// NewProviderRegistry creates a new ProviderRegistry instance.
// Deprecated: Use registry.GetRegistry() instead.
func NewProviderRegistry() *ProviderRegistry {
	return registry.GetRegistry()
}

// RegisterGlobal registers a provider with the global factory.
// Deprecated: Use registry.GetRegistry().Register() instead.
func RegisterGlobal(name string, creator func(ctx context.Context, config Config) (iface.Embedder, error)) {
	// Wrap the creator to handle type assertion
	registry.GetRegistry().Register(name, func(ctx context.Context, config any) (iface.Embedder, error) {
		embConfig, ok := config.(Config)
		if !ok {
		return nil, iface.WrapError(
			fmt.Errorf("invalid config type"),
			iface.ErrCodeInvalidConfig,
			"invalid config type",
		)
		}
		return creator(ctx, embConfig)
	})
}

// NewEmbedder creates an embedder using the global factory.
func NewEmbedder(ctx context.Context, name string, config Config) (iface.Embedder, error) {
	return registry.GetRegistry().Create(ctx, name, config)
}

// ListAvailableProviders returns all available providers from the global factory.
func ListAvailableProviders() []string {
	return registry.GetRegistry().ListProviders()
}

// GetGlobalRegistry returns the global registry instance for advanced usage.
// Deprecated: Use GetRegistry() instead for consistency.
func GetGlobalRegistry() registryiface.Registry {
	return registry.GetRegistry()
}

// GetRegistry returns the global registry instance.
// This follows the standard pattern used across all Beluga AI packages.
func GetRegistry() registryiface.Registry {
	return registry.GetRegistry()
}
