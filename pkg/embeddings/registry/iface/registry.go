// Package iface defines the registry interface for embedder providers.
// This package is separate from the main embeddings package to avoid import cycles.
//
// The registry interface uses `any` for config to avoid importing the main package.
// Providers will need to import the main package to get the actual Config type
// and perform type assertions when implementing the factory function.
package iface

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
)

// EmbedderFactory defines the function signature for creating embedders.
// This type is used by providers to register themselves.
// The config parameter is `any` to avoid import cycles - providers should
// assert it to embeddings.Config when implementing.
type EmbedderFactory func(ctx context.Context, config any) (iface.Embedder, error)

// Registry defines the interface for embedder provider registration.
// Implementations of this interface manage provider registration and creation.
type Registry interface {
	// Register registers a provider factory function with the given name.
	Register(name string, factory EmbedderFactory)

	// Create creates a new embedder instance using the registered provider factory.
	Create(ctx context.Context, name string, config any) (iface.Embedder, error)

	// ListProviders returns a list of all registered provider names.
	ListProviders() []string

	// IsRegistered checks if a provider is registered.
	IsRegistered(name string) bool
}
