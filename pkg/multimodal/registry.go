// Package multimodal provides convenience functions for accessing the multimodal registry.
// The actual registry implementation is in pkg/multimodal/registry.
package multimodal

import (
	"github.com/lookatitude/beluga-ai/pkg/multimodal/registry"
)

// GetRegistry returns the global registry instance.
// This is a convenience wrapper around registry.GetRegistry().
func GetRegistry() *registry.ProviderRegistry {
	return registry.GetRegistry()
}
