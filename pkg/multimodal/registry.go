// Package multimodal provides convenience functions for accessing the multimodal registry.
// The actual registry implementation is in pkg/multimodal/registry.
package multimodal

import (
	"github.com/lookatitude/beluga-ai/pkg/multimodal/registry"
)

// GetRegistry returns the global multimodal provider registry instance.
// The registry manages provider registration and creation of multimodal model instances.
// This is a convenience wrapper around registry.GetRegistry().
//
// Example:
//
//	registry := multimodal.GetRegistry()
//	providers := registry.ListProviders()
//	fmt.Printf("Available providers: %v\n", providers)
func GetRegistry() *registry.ProviderRegistry {
	return registry.GetRegistry()
}
