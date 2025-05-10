package factory

import (
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/registry" // Import the new registry
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// CreateRegisteredEmbedder creates a single embedder instance using a globally registered creator from the registry.
// providerType is the type of provider (e.g., "openai").
// configProvider is used by the creator if it needs to read further specific configurations.
// instanceConfig is the specific configuration for this instance, which the creator will use.
func CreateRegisteredEmbedder(providerType string, configProvider config.Provider, instanceConfig schema.EmbeddingProviderConfig) (iface.Embedder, error) {
	creator, found := registry.GetEmbedderCreator(providerType)
	if !found {
		available := registry.GetRegisteredEmbedders()
		return nil, fmt.Errorf("unknown or unregistered embedding provider specified: %s. Available types: %v", providerType, available)
	}
	return creator(configProvider, instanceConfig)
}

// EmbedderProviderFactory defines the interface for an embedder provider factory that manages multiple named instances.
type EmbedderProviderFactory interface {
	GetProvider(name string) (iface.Embedder, error)
}

// embedderProviderFactoryImpl is the concrete implementation of EmbedderProviderFactory.
type embedderProviderFactoryImpl struct {
	configProvider config.Provider
	providers      map[string]iface.Embedder // Stores instantiated providers by their unique name from config
}

// NewEmbedderProviderFactory creates a new embedder provider factory.
// It initializes providers based on the application configuration (list of embedding_providers) and globally registered creators.
func NewEmbedderProviderFactory(configProvider config.Provider) (EmbedderProviderFactory, error) {
	factory := &embedderProviderFactoryImpl{
		configProvider: configProvider,
		providers:      make(map[string]iface.Embedder),
	}

	embeddingConfigs, err := configProvider.GetEmbeddingProvidersConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding_providers config: %w", err)
	}

	fmt.Printf("EmbedderProviderFactory: Processing %d embedding configs from embedding_providers list\n", len(embeddingConfigs))

	for i, providerConfig := range embeddingConfigs { // providerConfig is schema.EmbeddingProviderConfig from the list
		fmt.Printf("EmbedderProviderFactory: List Config %d: Name=%s, ProviderType=%s, APIKeyLength=%d\n", i, providerConfig.Name, providerConfig.Provider, len(providerConfig.APIKey))

		// Use CreateRegisteredEmbedder to leverage the global registry and creator logic
		providerInstance, err := CreateRegisteredEmbedder(providerConfig.Provider, configProvider, providerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create embedder instance %s (type %s) from list: %w", providerConfig.Name, providerConfig.Provider, err)
		}

		factory.providers[providerConfig.Name] = providerInstance
		fmt.Printf("EmbedderProviderFactory: Successfully created and stored provider instance %s (type %s) from list\n", providerConfig.Name, providerConfig.Provider)
	}

	return factory, nil
}

// GetProvider returns an embedder provider by its unique configured name (from the embedding_providers list).
func (f *embedderProviderFactoryImpl) GetProvider(name string) (iface.Embedder, error) {
	provider, ok := f.providers[name]
	if !ok {
		return nil, fmt.Errorf("embedding provider instance with name %s not found or not configured in factory (check embedding_providers list)", name)
	}
	return provider, nil
}

// No more init() block here for static registration. Registration will happen in provider packages.
// The NewEmbedderProvider function has been removed from this file as it was a duplicate.
// The canonical version is in pkg/embeddings/provider.go

