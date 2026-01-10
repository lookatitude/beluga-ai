package chatmodels

import (
	"github.com/lookatitude/beluga-ai/pkg/chatmodels/registry"
	registryiface "github.com/lookatitude/beluga-ai/pkg/chatmodels/registry/iface"
)

// GetRegistry returns the global registry instance.
// This is a convenience function that delegates to the registry package.
// This follows the standard pattern used across all Beluga AI packages.
//
// Example:
//
//	registry := chatmodels.GetRegistry()
//	if registry.IsRegistered("openai") {
//	    model, err := registry.CreateProvider("gpt-4", config, options)
//	}
func GetRegistry() registryiface.Registry {
	return registry.GetRegistry()
}
