// Package embeddings provides a standardized factory pattern for embedder creation.
// This follows the Beluga AI Framework design patterns with consistent factory interfaces.
package embeddings

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
)

// Factory defines the interface for creating Embedder instances.
// This enables dependency injection and makes testing easier.
type Factory interface {
	// CreateEmbedder creates a new Embedder instance with the given configuration.
	// The config parameter contains provider-specific settings.
	CreateEmbedder(ctx context.Context, config Config) (iface.Embedder, error)
}

// RegisterGlobal registers a provider with the global factory.
// This is a convenience function that wraps the creator to handle Config type assertion,
// allowing providers to register with a typed Config instead of `any`.
//
// Parameters:
//   - name: Unique identifier for the provider
//   - creator: Factory function that takes a typed Config
//
// Example:
//
//	embeddings.RegisterGlobal("custom", func(ctx context.Context, config embeddings.Config) (iface.Embedder, error) {
//	    return NewCustomEmbedder(config.Custom)
//	})
func RegisterGlobal(name string, creator func(ctx context.Context, config Config) (iface.Embedder, error)) {
	// Wrap the creator to handle type assertion
	GetRegistry().Register(name, func(ctx context.Context, config any) (iface.Embedder, error) {
		embConfig, ok := config.(Config)
		if !ok {
			return nil, NewInvalidConfigError("register_global", "invalid config type: expected embeddings.Config", nil)
		}
		return creator(ctx, embConfig)
	})
}
