package composite

import (
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// CompositeProvider combines multiple providers with fallback logic.
// It tries providers in order until one succeeds.
type CompositeProvider struct {
	providers []iface.Provider
}

// NewCompositeProvider creates a new composite provider with the given providers.
// Providers are tried in the order they are provided.
func NewCompositeProvider(providers ...iface.Provider) *CompositeProvider {
	return &CompositeProvider{
		providers: providers,
	}
}

// Load attempts to load configuration from the first provider that succeeds.
func (cp *CompositeProvider) Load(configStruct interface{}) error {
	var lastErr error

	for i, provider := range cp.providers {
		if err := provider.Load(configStruct); err != nil {
			lastErr = fmt.Errorf("provider %d failed: %w", i, err)
			continue
		}
		// Success - return
		return nil
	}

	if lastErr != nil {
		return iface.WrapError(lastErr, iface.ErrCodeAllProvidersFailed, "all providers failed")
	}
	return iface.NewConfigError(iface.ErrCodeInvalidParameters, "no providers configured")
}

// UnmarshalKey attempts to unmarshal a key from the first provider that has it.
func (cp *CompositeProvider) UnmarshalKey(key string, rawVal interface{}) error {
	for i, provider := range cp.providers {
		if provider.IsSet(key) {
			if err := provider.UnmarshalKey(key, rawVal); err != nil {
				return iface.WrapError(err, iface.ErrCodeParseFailed, "provider %d failed to unmarshal key %s", i, key)
			}
			return nil
		}
	}
	return iface.NewConfigError(iface.ErrCodeKeyNotFound, "key %s not found in any provider", key)
}

// GetString retrieves a string value from the first provider that has it.
func (cp *CompositeProvider) GetString(key string) string {
	for _, provider := range cp.providers {
		if provider.IsSet(key) {
			return provider.GetString(key)
		}
	}
	return ""
}

// GetInt retrieves an int value from the first provider that has it.
func (cp *CompositeProvider) GetInt(key string) int {
	for _, provider := range cp.providers {
		if provider.IsSet(key) {
			return provider.GetInt(key)
		}
	}
	return 0
}

// GetBool retrieves a bool value from the first provider that has it.
func (cp *CompositeProvider) GetBool(key string) bool {
	for _, provider := range cp.providers {
		if provider.IsSet(key) {
			return provider.GetBool(key)
		}
	}
	return false
}

// GetFloat64 retrieves a float64 value from the first provider that has it.
func (cp *CompositeProvider) GetFloat64(key string) float64 {
	for _, provider := range cp.providers {
		if provider.IsSet(key) {
			return provider.GetFloat64(key)
		}
	}
	return 0.0
}

// GetStringMapString retrieves a map[string]string value from the first provider that has it.
func (cp *CompositeProvider) GetStringMapString(key string) map[string]string {
	for _, provider := range cp.providers {
		if provider.IsSet(key) {
			return provider.GetStringMapString(key)
		}
	}
	return nil
}

// IsSet checks if a key is set in any of the providers.
func (cp *CompositeProvider) IsSet(key string) bool {
	for _, provider := range cp.providers {
		if provider.IsSet(key) {
			return true
		}
	}
	return false
}

// GetLLMProviderConfig retrieves LLM provider config from the first provider that has it.
func (cp *CompositeProvider) GetLLMProviderConfig(name string) (schema.LLMProviderConfig, error) {
	for _, provider := range cp.providers {
		if cfg, err := provider.GetLLMProviderConfig(name); err == nil {
			return cfg, nil
		}
	}
	return schema.LLMProviderConfig{}, iface.NewConfigError(iface.ErrCodeConfigNotFound, "LLM provider config %s not found", name)
}

// GetLLMProvidersConfig retrieves all LLM provider configs, merging from all providers.
func (cp *CompositeProvider) GetLLMProvidersConfig() ([]schema.LLMProviderConfig, error) {
	var allConfigs []schema.LLMProviderConfig
	configMap := make(map[string]schema.LLMProviderConfig)

	for _, provider := range cp.providers {
		if configs, err := provider.GetLLMProvidersConfig(); err == nil {
			for _, config := range configs {
				configMap[config.Name] = config
			}
		}
	}

	for _, config := range configMap {
		allConfigs = append(allConfigs, config)
	}

	return allConfigs, nil
}

// GetEmbeddingProvidersConfig retrieves all embedding provider configs, merging from all providers.
func (cp *CompositeProvider) GetEmbeddingProvidersConfig() ([]schema.EmbeddingProviderConfig, error) {
	var allConfigs []schema.EmbeddingProviderConfig
	configMap := make(map[string]schema.EmbeddingProviderConfig)

	for _, provider := range cp.providers {
		if configs, err := provider.GetEmbeddingProvidersConfig(); err == nil {
			for _, config := range configs {
				configMap[config.Name] = config
			}
		}
	}

	for _, config := range configMap {
		allConfigs = append(allConfigs, config)
	}

	return allConfigs, nil
}

// GetVectorStoresConfig retrieves all vector store configs, merging from all providers.
func (cp *CompositeProvider) GetVectorStoresConfig() ([]schema.VectorStoreConfig, error) {
	var allConfigs []schema.VectorStoreConfig
	configMap := make(map[string]schema.VectorStoreConfig)

	for _, provider := range cp.providers {
		if configs, err := provider.GetVectorStoresConfig(); err == nil {
			for _, config := range configs {
				configMap[config.Name] = config
			}
		}
	}

	for _, config := range configMap {
		allConfigs = append(allConfigs, config)
	}

	return allConfigs, nil
}

// GetAgentConfig retrieves agent config from the first provider that has it.
func (cp *CompositeProvider) GetAgentConfig(name string) (schema.AgentConfig, error) {
	for _, provider := range cp.providers {
		if cfg, err := provider.GetAgentConfig(name); err == nil {
			return cfg, nil
		}
	}
	return schema.AgentConfig{}, iface.NewConfigError(iface.ErrCodeConfigNotFound, "agent config %s not found", name)
}

// GetAgentsConfig retrieves all agent configs, merging from all providers.
func (cp *CompositeProvider) GetAgentsConfig() ([]schema.AgentConfig, error) {
	var allConfigs []schema.AgentConfig
	configMap := make(map[string]schema.AgentConfig)

	for _, provider := range cp.providers {
		if configs, err := provider.GetAgentsConfig(); err == nil {
			for _, config := range configs {
				configMap[config.Name] = config
			}
		}
	}

	for _, config := range configMap {
		allConfigs = append(allConfigs, config)
	}

	return allConfigs, nil
}

// GetToolConfig retrieves tool config from the first provider that has it.
func (cp *CompositeProvider) GetToolConfig(name string) (iface.ToolConfig, error) {
	for _, provider := range cp.providers {
		if cfg, err := provider.GetToolConfig(name); err == nil {
			return cfg, nil
		}
	}
	return iface.ToolConfig{}, iface.NewConfigError(iface.ErrCodeConfigNotFound, "tool config %s not found", name)
}

// GetToolsConfig retrieves all tool configs, merging from all providers.
func (cp *CompositeProvider) GetToolsConfig() ([]iface.ToolConfig, error) {
	var allConfigs []iface.ToolConfig
	configMap := make(map[string]iface.ToolConfig)

	for _, provider := range cp.providers {
		if configs, err := provider.GetToolsConfig(); err == nil {
			for _, config := range configs {
				configMap[config.Name] = config
			}
		}
	}

	for _, config := range configMap {
		allConfigs = append(allConfigs, config)
	}

	return allConfigs, nil
}

// Validate validates using all providers.
func (cp *CompositeProvider) Validate() error {
	for i, provider := range cp.providers {
		if err := provider.Validate(); err != nil {
			return iface.WrapError(err, iface.ErrCodeValidationFailed, "provider %d validation failed", i)
		}
	}
	return nil
}

// SetDefaults sets defaults using all providers.
func (cp *CompositeProvider) SetDefaults() error {
	for i, provider := range cp.providers {
		if err := provider.SetDefaults(); err != nil {
			return iface.WrapError(err, iface.ErrCodeLoadFailed, "provider %d set defaults failed", i)
		}
	}
	return nil
}

var _ iface.Provider = (*CompositeProvider)(nil)
