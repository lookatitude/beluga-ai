package embeddings

import (
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface" // Import the new interface package
)

const (
	ProviderMock   = "mock"
	ProviderOpenAI = "openai"
)

// EmbedderFactoryConfig defines the overall configuration for selecting an embedder provider.
// It will be part of the main application configuration.
type EmbedderFactoryConfig struct {
	Provider string `mapstructure:"provider"` // e.g., "mock", "openai"
}

// EmbedderConstructor defines the function signature for creating an embedder instance.
// It takes the global ViperProvider to allow unmarshalling of specific configurations.
type EmbedderConstructor func(appConfig *config.ViperProvider) (iface.Embedder, error)

var embedderRegistry = make(map[string]EmbedderConstructor)

// RegisterEmbedderProvider allows embedder providers to register their constructor functions.
// This is typically called from the init() function of each provider package.
func RegisterEmbedderProvider(name string, constructor EmbedderConstructor) {
	if _, exists := embedderRegistry[name]; exists {
		// Optionally, log a warning or panic if a provider is registered multiple times.
		// For simplicity, we might allow overriding, or panic as shown here.
		panic(fmt.Sprintf("Embedder provider %s already registered", name))
	}
	embedderRegistry[name] = constructor
}

// NewEmbedderProvider function removed from this file to resolve redeclaration error.
// The canonical version is in pkg/embeddings/provider.go

