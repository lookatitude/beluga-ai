// Package iface defines the registry interface for retriever providers.
// This contains factory types and registry interfaces used by providers
// to register themselves without importing the main retrievers package.
package iface

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/core"
)

// RetrieverFactory defines the function signature for creating retrievers.
// This type is used by providers to register themselves with the registry.
// The config parameter is `any` to avoid import cycles - providers should
// assert it to the appropriate config type when implementing.
type RetrieverFactory func(ctx context.Context, config any) (core.Retriever, error)

// Registry defines the interface for retriever provider registration.
// Implementations of this interface manage provider registration and creation.
type Registry interface {
	// Register registers a provider factory function with the given name.
	Register(name string, factory RetrieverFactory)

	// Create creates a new retriever instance using the registered provider factory.
	Create(ctx context.Context, name string, config any) (core.Retriever, error)

	// ListProviders returns a list of all registered provider names.
	ListProviders() []string

	// IsRegistered checks if a provider is registered.
	IsRegistered(name string) bool
}
