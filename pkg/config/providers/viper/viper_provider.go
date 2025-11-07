package viper

import (
	"fmt"
	"os"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/config/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/spf13/viper"
)

// ViperProvider implements the Provider interface using the Viper library.
type ViperProvider struct {
	v *viper.Viper
}

// NewViperProvider creates a new ViperProvider.
// The format parameter can be "yaml", "json", "toml", or empty for auto-detection.
func NewViperProvider(configName string, configPaths []string, envPrefix string, format string) (*ViperProvider, error) {
	v := viper.New()

	if configName != "" {
		v.SetConfigName(configName)
		// Set config type if specified, otherwise let viper auto-detect
		if format != "" {
			v.SetConfigType(format)
		}
		for _, path := range configPaths {
			v.AddConfigPath(path)
		}
	}

	if envPrefix != "" {
		v.SetEnvPrefix(envPrefix)
	}
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// Enable type inference for better array/slice handling from env vars
	v.SetTypeByDefaultValue(true)

	if configName != "" {
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
		}
	}

	// If loading from env only (no config file), manually construct arrays from env vars
	if configName == "" && envPrefix != "" {
		constructArraysFromEnv(v, envPrefix)
	}

	return &ViperProvider{v: v}, nil
}

// constructArraysFromEnv manually constructs array structures from environment variables
// This is needed because Viper doesn't automatically build arrays from indexed env vars
func constructArraysFromEnv(v *viper.Viper, prefix string) {
	prefixUpper := strings.ToUpper(prefix) + "_"

	// We'll manually parse common array structures
	// For each array type, find the highest index and construct the array
	arrayTypes := []string{"llm_providers", "embedding_providers", "vector_stores", "agents", "tools"}

	for _, arrayType := range arrayTypes {
		arrayTypeUpper := strings.ToUpper(strings.ReplaceAll(arrayType, "_", "_"))
		maxIndex := -1
		envMap := make(map[string]string) // key: "index_field", value: env value

		// Find the maximum index and collect all env vars for this array type
		for _, env := range os.Environ() {
			if strings.HasPrefix(env, prefixUpper+arrayTypeUpper+"_") {
				// Extract index and field from env var name like TEST_LLM_PROVIDERS_0_NAME
				parts := strings.Split(env, "=")
				if len(parts) == 2 {
					key := parts[0]
					value := parts[1]
					// Remove prefix and array type to get index and field
					remainder := strings.TrimPrefix(key, prefixUpper+arrayTypeUpper+"_")
					remainderParts := strings.Split(remainder, "_")
					if len(remainderParts) > 0 {
						// Try to parse index
						var idx int
						if n, err := fmt.Sscanf(remainderParts[0], "%d", &idx); n == 1 && err == nil {
							if idx > maxIndex {
								maxIndex = idx
							}
							// Store the field name and value
							fieldName := strings.Join(remainderParts[1:], "_")
							envMap[fmt.Sprintf("%d_%s", idx, fieldName)] = value
						}
					}
				}
			}
		}

		// If we found array elements, construct the array structure and set values
		if maxIndex >= 0 {
			// Build array as a slice of maps
			array := make([]interface{}, maxIndex+1)
			for i := 0; i <= maxIndex; i++ {
				item := make(map[string]interface{})
				// Set individual field values from env vars
				for key, value := range envMap {
					// Parse "index_fieldname" format
					parts := strings.SplitN(key, "_", 2)
					if len(parts) == 2 {
						var idx int
						if n, err := fmt.Sscanf(parts[0], "%d", &idx); n == 1 && err == nil && idx == i {
							fieldName := parts[1]
							// Convert SNAKE_CASE to lowercase (e.g., "NAME" -> "name", "MODEL_NAME" -> "model_name")
							fieldNameLower := strings.ToLower(fieldName)
							item[fieldNameLower] = value
						}
					}
				}
				array[i] = item
			}
			// Set the complete array structure
			v.Set(arrayType, array)
		}
	}
}

// Load unmarshals the configuration into the given struct.
func (vp *ViperProvider) Load(configStruct interface{}) error {
	// Try to unmarshal the entire config first
	if err := vp.v.Unmarshal(configStruct); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// For Config structs, also try UnmarshalKey for each array section
	// This ensures arrays are properly unmarshaled even if the main Unmarshal fails
	if cfg, ok := configStruct.(*iface.Config); ok {
		_ = vp.v.UnmarshalKey("llm_providers", &cfg.LLMProviders)
		_ = vp.v.UnmarshalKey("embedding_providers", &cfg.EmbeddingProviders)
		_ = vp.v.UnmarshalKey("vector_stores", &cfg.VectorStores)
		_ = vp.v.UnmarshalKey("agents", &cfg.Agents)
		_ = vp.v.UnmarshalKey("tools", &cfg.Tools)
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
	return llmConfig, fmt.Errorf("LLM provider configuration for %s not found", name)
}

// GetLLMProvidersConfig retrieves all LLMProviderConfig.
func (vp *ViperProvider) GetLLMProvidersConfig() ([]schema.LLMProviderConfig, error) {
	var configs []schema.LLMProviderConfig
	key := "llm_providers"
	if !vp.v.IsSet(key) {
		return configs, nil
	}
	if err := vp.v.UnmarshalKey(key, &configs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal LLM providers config from key %s: %w", key, err)
	}
	return configs, nil
}

// GetEmbeddingProvidersConfig retrieves all EmbeddingProviderConfig.
func (vp *ViperProvider) GetEmbeddingProvidersConfig() ([]schema.EmbeddingProviderConfig, error) {
	var configs []schema.EmbeddingProviderConfig
	key := "embedding_providers"
	if !vp.v.IsSet(key) {
		return configs, nil
	}

	if err := vp.v.UnmarshalKey(key, &configs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embedding providers config from key %s: %w", key, err)
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
		return nil, fmt.Errorf("failed to unmarshal vector stores config from key %s: %w", key, err)
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
	return agentConfig, fmt.Errorf("agent configuration for %s not found", name)
}

// GetAgentsConfig retrieves all AgentConfig.
func (vp *ViperProvider) GetAgentsConfig() ([]schema.AgentConfig, error) {
	var configs []schema.AgentConfig
	key := "agents"
	if !vp.v.IsSet(key) {
		return configs, nil
	}
	if err := vp.v.UnmarshalKey(key, &configs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agents config from key %s: %w", key, err)
	}
	return configs, nil
}

// GetToolConfig retrieves a specific ToolConfig by name from the main config.
func (vp *ViperProvider) GetToolConfig(name string) (iface.ToolConfig, error) {
	var toolConfig iface.ToolConfig
	tools, err := vp.GetToolsConfig()
	if err != nil {
		return toolConfig, err
	}
	for _, cfg := range tools {
		if cfg.Name == name {
			return cfg, nil
		}
	}
	return toolConfig, fmt.Errorf("tool configuration for %s not found", name)
}

// GetToolsConfig retrieves all ToolConfig.
func (vp *ViperProvider) GetToolsConfig() ([]iface.ToolConfig, error) {
	var configs []iface.ToolConfig
	key := "tools"
	if !vp.v.IsSet(key) {
		return configs, nil
	}
	if err := vp.v.UnmarshalKey(key, &configs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools config from key %s: %w", key, err)
	}
	return configs, nil
}

// Validate validates the configuration loaded by this provider.
func (vp *ViperProvider) Validate() error {
	return iface.ValidateProvider(vp)
}

// SetDefaults sets default values for unset configuration fields.
func (vp *ViperProvider) SetDefaults() error {
	var cfg iface.Config
	if err := vp.Load(&cfg); err != nil {
		return fmt.Errorf("failed to load config for setting defaults: %w", err)
	}

	iface.SetDefaults(&cfg)

	// Re-save the config with defaults set
	// Note: This is a simplified implementation. In a real scenario,
	// you might want to update the underlying viper instance or
	// provide a way to persist the defaults.
	return nil
}

var _ iface.Provider = (*ViperProvider)(nil)
