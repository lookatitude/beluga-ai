package chatmodels

import (
	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
)

// GetRegistry returns the global registry instance.
// This is a convenience function that delegates to the iface package.
// This follows the standard pattern used across all Beluga AI packages.
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
