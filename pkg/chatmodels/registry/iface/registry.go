// Package iface defines the registry interface for chat model providers.
// This package is separate from the main chatmodels package to avoid import cycles.
//
// The registry interface uses `any` for config to avoid importing the main package.
// Providers will need to import the main package to get the actual Config type
// and perform type assertions when implementing the factory function.
package iface

import (
	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
)

// ChatModelFactory defines the function signature for creating chat models.
// This type is used by providers to register themselves.
// The config parameter is `any` to avoid import cycles - providers should
// assert it to *chatmodels.Config when implementing.
type ChatModelFactory func(model string, config any, options *iface.Options) (iface.ChatModel, error)

// Registry defines the interface for chat model provider registration.
// Implementations of this interface manage provider registration and creation.
type Registry interface {
	// Register registers a provider factory function with the given name.
	Register(name string, factory ChatModelFactory)

	// GetProvider returns a provider factory for the given name.
	GetProvider(name string) (ChatModelFactory, error)

	// CreateProvider creates a chat model using the registered provider factory.
	CreateProvider(model string, config any, options *iface.Options) (iface.ChatModel, error)

	// IsRegistered checks if a provider is registered.
	IsRegistered(name string) bool

	// ListProviders returns a list of all registered provider names.
	ListProviders() []string
}
