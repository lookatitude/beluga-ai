// Package chatmodels provides a standardized registry pattern for chat model creation.
// This follows the Beluga AI Framework design patterns with consistent factory interfaces.
package chatmodels

import (
	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
)

// ProviderFactory defines the function signature for creating chat models.
// This type is used by providers to register themselves with the registry.
type ProviderFactory = iface.ProviderFactory

// GetRegistry returns the global registry instance.
// This follows the standard pattern used across all Beluga AI packages.
// This delegates to the iface package's registry where providers register themselves.
//
// Example:
//
//	registry := chatmodels.GetRegistry()
//	if registry.IsRegistered("openai") {
//	    model, err := registry.CreateProvider("gpt-4", config, options)
//	}
func GetRegistry() iface.Registry {
	return iface.GetRegistry()
}

// Global convenience functions for working with the default registry.

// Register registers a provider with the global registry.
// This is a convenience function for registering with the global registry.
//
// Parameters:
//   - name: Unique identifier for the provider
//   - factory: Function that creates chat model instances
//
// Example:
//
//	chatmodels.Register("custom", customChatModelFactory)
func Register(name string, factory ProviderFactory) {
	GetRegistry().Register(name, factory)
}

// ListProviders returns all available providers from the global registry.
// This is a convenience function for listing providers from the global registry.
//
// Returns:
//   - []string: Slice of available provider names
func ListProviders() []string {
	return GetRegistry().ListProviders()
}
