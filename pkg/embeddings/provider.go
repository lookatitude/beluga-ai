package embeddings

import (
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/config"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/factory"
	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// NewEmbedderProvider creates a single, default embedding provider instance
// based on the 'embeddings.provider' and 'embeddings.<provider_type>' configuration.
// It relies on the global embedder registry populated by init() functions in the factory
// and provider-specific packages.
func NewEmbedderProvider(configProvider config.Provider) (iface.Embedder, error) {
	providerTypeRaw := configProvider.GetString("embeddings.provider") // Corrected: GetString returns a single value
	// Check for error if GetString could return an error, or if it panics on not found.
	// Assuming GetString returns empty string if not found or if an error occurs, based on typical Viper behavior.
	// If it can return an error, the signature would be (string, error) and handled accordingly.
	// For now, proceed as if it returns a string, potentially empty.

	providerType := strings.TrimSpace(providerTypeRaw)
	if providerType == "" {
		// Check if the key was actually missing or just empty.
		if !configProvider.IsSet("embeddings.provider") {
		    return nil, fmt.Errorf("embedding provider key 'embeddings.provider' not found in configuration")
		}
		return nil, fmt.Errorf("embedding provider not specified in configuration under 'embeddings.provider' or value is empty")
	}

	var instanceConfig schema.EmbeddingProviderConfig
	instanceConfig.Provider = providerType
	instanceConfig.Name = providerType // Use the providerType as the Name for this default, single instance.

	specificConfigKey := fmt.Sprintf("embeddings.%s", providerType)

	if configProvider.IsSet(specificConfigKey) {
		var specificProviderMap map[string]interface{}
		if err := configProvider.UnmarshalKey(specificConfigKey, &specificProviderMap); err != nil {
			fmt.Printf("Warning: NewEmbedderProvider: failed to unmarshal specific provider config at key '%s': %v. Proceeding with potentially incomplete specific config.\n", specificConfigKey, err)
			instanceConfig.ProviderSpecific = make(map[string]interface{}) // Ensure it's not nil
		} else {
			if model, ok := specificProviderMap["model"].(string); ok {
				instanceConfig.ModelName = model
				delete(specificProviderMap, "model")
			} else if model, ok := specificProviderMap["model_name"].(string); ok {
				instanceConfig.ModelName = model
				delete(specificProviderMap, "model_name")
			}

			if apiKey, ok := specificProviderMap["api_key"].(string); ok {
				instanceConfig.APIKey = apiKey
				delete(specificProviderMap, "api_key")
			}

			if len(specificProviderMap) > 0 {
				instanceConfig.ProviderSpecific = specificProviderMap
			} else {
				instanceConfig.ProviderSpecific = make(map[string]interface{}) // Ensure it's not nil if empty after extraction
			}
		}
	} else {
		fmt.Printf("NewEmbedderProvider: Specific config block '%s' not found. Creator for '%s' will use defaults or handle missing required fields.\n", specificConfigKey, providerType)
		instanceConfig.ProviderSpecific = make(map[string]interface{}) // Ensure it's not nil
	}

	return factory.CreateRegisteredEmbedder(providerType, configProvider, instanceConfig)
}

