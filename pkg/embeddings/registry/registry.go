package registry

import (
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// EmbedderCreatorFunc defines the signature for a function that creates an Embedder.
// It takes a config.Provider for general configuration access and a specific
// schema.EmbeddingProviderConfig for the instance being created.
// This allows creators to fetch additional global or provider-specific settings if needed.
type EmbedderCreatorFunc func(generalConfig config.Provider, instanceConfig schema.EmbeddingProviderConfig) (iface.Embedder, error)

var embedderRegistry = make(map[string]EmbedderCreatorFunc)

// RegisterEmbedder registers an embedder provider with a given name and creation function.
// This function is typically called from the init() function of each embedder provider package.
func RegisterEmbedder(name string, creator EmbedderCreatorFunc) {
	if _, exists := embedderRegistry[name]; exists {
		// Allow re-registration, could be useful for testing or overriding.
		// For production, this might indicate a naming conflict.
		fmt.Printf("EmbedderRegistry: Warning - Reregistering embedder provider type %s\n", name)
	}
	embedderRegistry[name] = creator
	fmt.Printf("EmbedderRegistry: Registered embedder provider type %s\n", name)
}

// GetEmbedderCreator returns the creation function for a registered embedder provider.
func GetEmbedderCreator(name string) (EmbedderCreatorFunc, bool) {
	creator, found := embedderRegistry[name]
	return creator, found
}

// GetRegisteredEmbedders returns a list of all registered embedder provider names.
func GetRegisteredEmbedders() []string {
	names := make([]string, 0, len(embedderRegistry))
	for name := range embedderRegistry {
		names = append(names, name)
	}
	return names
}

