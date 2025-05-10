package config

import (
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/spf13/viper"
)

// ViperProvider implements the Provider interface using the Viper library.
type ViperProvider struct {
	v *viper.Viper
}

// NewViperProvider creates a new ViperProvider.
func NewViperProvider(configName string, configPaths []string, envPrefix string) (*ViperProvider, error) {
	v := viper.New()

	if configName != "" {
		v.SetConfigName(configName)
		v.SetConfigType("yaml")
		for _, path := range configPaths {
			v.AddConfigPath(path)
		}
	}

	if envPrefix != "" {
		v.SetEnvPrefix(envPrefix)
	}
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if configName != "" {
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
		}
	}

	return &ViperProvider{v: v}, nil
}

// Load unmarshals the configuration into the given struct.
func (vp *ViperProvider) Load(configStruct interface{}) error {
	if err := vp.v.Unmarshal(configStruct); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return nil
}

// UnmarshalKey decodes the configuration at a specific key into a struct.
func (vp *ViperProvider) UnmarshalKey(key string, rawVal interface{}) error {
	return vp.v.UnmarshalKey(key, rawVal)
}

// GetString retrieves a string configuration value by key.
func (vp *ViperProvider) GetString(key string) string {
	return vp.v.GetString(key)
}

// GetInt retrieves an integer configuration value by key.
func (vp *ViperProvider) GetInt(key string) int {
	return vp.v.GetInt(key)
}

// GetBool retrieves a boolean configuration value by key.
func (vp *ViperProvider) GetBool(key string) bool {
	return vp.v.GetBool(key)
}

// GetFloat64 retrieves a float64 configuration value by key.
func (vp *ViperProvider) GetFloat64(key string) float64 {
	return vp.v.GetFloat64(key)
}

// GetStringMapString retrieves a map[string]string configuration value by key.
func (vp *ViperProvider) GetStringMapString(key string) map[string]string {
	return vp.v.GetStringMapString(key)
}

// IsSet checks if a key is set in the configuration.
func (vp *ViperProvider) IsSet(key string) bool {
	return vp.v.IsSet(key)
}

// GetLLMProviderConfig retrieves a specific LLMProviderConfig by name.
func (vp *ViperProvider) GetLLMProviderConfig(name string) (schema.LLMProviderConfig, error) {
	var llmConfig schema.LLMProviderConfig
	llmProviders, err := vp.GetLLMProvidersConfig()
	if err != nil {
		return llmConfig, err
	}
	for _, cfg := range llmProviders {
		if cfg.Name == name {
			return cfg, nil
		}
	}
	return llmConfig, fmt.Errorf("LLM provider configuration for 	%s	 not found", name)
}

// GetLLMProvidersConfig retrieves all LLMProviderConfig.
func (vp *ViperProvider) GetLLMProvidersConfig() ([]schema.LLMProviderConfig, error) {
    var configs []schema.LLMProviderConfig
    key := "llm_providers"
    if !vp.v.IsSet(key) {
        return configs, nil
    }
    if err := vp.v.UnmarshalKey(key, &configs); err != nil {
        return nil, fmt.Errorf("failed to unmarshal LLM providers config from key 	%s	: %w", key, err)
    }
    return configs, nil
}

// GetEmbeddingProvidersConfig retrieves all EmbeddingProviderConfig.
func (vp *ViperProvider) GetEmbeddingProvidersConfig() ([]schema.EmbeddingProviderConfig, error) {
    var configs []schema.EmbeddingProviderConfig
    key := "embedding_providers"
    if !vp.v.IsSet(key) {
        fmt.Println("ViperProvider: embedding_providers key not set")
        return configs, nil
    }

    // Debug: Print the raw value from Viper for embedding_providers
    rawValue := vp.v.Get(key)
    fmt.Printf("ViperProvider: Raw value for 	%s	: %+v\n", key, rawValue)

    if err := vp.v.UnmarshalKey(key, &configs); err != nil {
        return nil, fmt.Errorf("failed to unmarshal embedding providers config from key 	%s	: %w", key, err)
    }
    
    // Debug: Print the unmarshalled configs
    for i, cfg := range configs {
        fmt.Printf("ViperProvider: Unmarshalled EmbeddingProviderConfig[%d]: Name=	%s	, Provider=	%s	, ModelName=	%s	, APIKey=	%s	 (Length: %d)\n", 
            i, cfg.Name, cfg.Provider, cfg.ModelName, cfg.APIKey, len(cfg.APIKey))
    }
    return configs, nil
}

// GetVectorStoresConfig retrieves all VectorStoreConfig.
func (vp *ViperProvider) GetVectorStoresConfig() ([]schema.VectorStoreConfig, error) {
    var configs []schema.VectorStoreConfig
    key := "vector_stores"
    if !vp.v.IsSet(key) {
        return configs, nil
    }
    if err := vp.v.UnmarshalKey(key, &configs); err != nil {
        return nil, fmt.Errorf("failed to unmarshal vector stores config from key 	%s	: %w", key, err)
    }
    return configs, nil
}

// GetAgentConfig retrieves a specific AgentConfig by name.
func (vp *ViperProvider) GetAgentConfig(name string) (schema.AgentConfig, error) {
	var agentConfig schema.AgentConfig
	agents, err := vp.GetAgentsConfig()
	if err != nil {
		return agentConfig, err
	}
	for _, cfg := range agents {
		if cfg.Name == name {
			return cfg, nil
		}
	}
	return agentConfig, fmt.Errorf("agent configuration for 	%s	 not found", name)
}

// GetAgentsConfig retrieves all AgentConfig.
func (vp *ViperProvider) GetAgentsConfig() ([]schema.AgentConfig, error) {
    var configs []schema.AgentConfig
    key := "agents"
    if !vp.v.IsSet(key) {
        return configs, nil
    }
    if err := vp.v.UnmarshalKey(key, &configs); err != nil {
        return nil, fmt.Errorf("failed to unmarshal agents config from key 	%s	: %w", key, err)
    }
    return configs, nil
}

// GetToolConfig retrieves a specific ToolConfig by name from the main config.
func (vp *ViperProvider) GetToolConfig(name string) (ToolConfig, error) {
    var toolConfig ToolConfig
    tools, err := vp.GetToolsConfig()
    if err != nil {
        return toolConfig, err
    }
    for _, cfg := range tools {
        if cfg.Name == name {
            return cfg, nil
        }
    }
    return toolConfig, fmt.Errorf("tool configuration for 	%s	 not found", name)
}

// GetToolsConfig retrieves all ToolConfig.
func (vp *ViperProvider) GetToolsConfig() ([]ToolConfig, error) {
    var configs []ToolConfig
    key := "tools"
    if !vp.v.IsSet(key) {
        return configs, nil
    }
    if err := vp.v.UnmarshalKey(key, &configs); err != nil {
        return nil, fmt.Errorf("failed to unmarshal tools config from key 	%s	: %w", key, err)
    }
    return configs, nil
}

var _ Provider = (*ViperProvider)(nil)

