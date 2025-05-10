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

// NewEmbedderProvider creates an instance of an Embedder based on the provided global configuration.
// It reads the embedder provider choice and uses the registry to find the appropriate constructor.
func NewEmbedderProvider(appConfig *config.ViperProvider) (iface.Embedder, error) {
	var factoryCfg EmbedderFactoryConfig
	factoryCfg.Provider = appConfig.GetString("embeddings.provider")
	if factoryCfg.Provider == "" && !appConfig.IsSet("embeddings.provider") {
		// If GetString returns empty and it's not explicitly set to empty, it might mean the key is missing.
		// Or, GetString might return a zero value if key is not found, depending on Viper's behavior / our wrapper.
		// A more robust check is IsSet.
		return nil, fmt.Errorf("embedding provider key 'embeddings.provider' not found in configuration")
	}

	if factoryCfg.Provider == "" {
		return nil, fmt.Errorf("embedding provider not specified in configuration under 'embeddings.provider'")
	}

	constructor, ok := embedderRegistry[factoryCfg.Provider]
	if !ok {
		return nil, fmt.Errorf("unknown or unregistered embedding provider specified: %s", factoryCfg.Provider)
	}

	// The constructor is responsible for unmarshalling its own specific configuration.
	return constructor(appConfig)
}

