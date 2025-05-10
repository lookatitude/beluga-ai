package config

import (
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/spf13/viper"
)

// ViperProvider implements the Provider interface using the Viper library.
// Viper supports reading from JSON, TOML, YAML, HCL, envfile and Java properties config files.
// It can also read from environment variables, remote config systems (etcd or Consul), and command-line flags.
type ViperProvider struct {
	v *viper.Viper
}

// NewViperProvider creates a new ViperProvider.
// It initializes Viper with sensible defaults: automatic environment variable binding
// and a replacer for environment variables (e.g., MY_APP_DB_HOST -> my_app.db.host).
func NewViperProvider(configName string, configPaths []string, envPrefix string) (*ViperProvider, error) {
	v := viper.New()

	if configName != "" {
		v.SetConfigName(configName) // Name of config file (without extension)
		v.SetConfigType("yaml")     // REQUIRED if the config file does not have the extension in the name
		for _, path := range configPaths {
			v.AddConfigPath(path) // Path to look for the config file in
		}
	}

	if envPrefix != "" {
		v.SetEnvPrefix(envPrefix) // Will be uppercased automatically
	}
	v.AutomaticEnv() // Read in environment variables that match

	// Example of a replacer: allows env var DB_HOST to be mapped to db.host
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// Attempt to read the config file if specified
	if configName != "" {
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// Config file not found; ignore error if this is acceptable
				// fmt.Printf("ViperProvider: Config file \t%s\t not found in paths \t%v\t. Relying on defaults/env vars.\n", configName, configPaths)
			} else {
				// Config file was found but another error was produced
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
		}
	}

	return &ViperProvider{v: v}, nil
}

// Load unmarshals the configuration into the given struct.
// The struct should have `mapstructure` tags for proper mapping.
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
	// Construct the key for the specific LLM provider configuration.
	// This assumes LLM provider configs are stored under a top-level key like "llm_providers"
	// and then indexed by their name, e.g., "llm_providers.openai_default".
	key := fmt.Sprintf("llm_providers.%s", name)

	if !vp.v.IsSet(key) {
		return llmConfig, fmt.Errorf("LLM provider configuration for '%s' not found at key '%s'", name, key)
	}

	if err := vp.v.UnmarshalKey(key, &llmConfig); err != nil {
		return llmConfig, fmt.Errorf("failed to unmarshal LLM provider config for '%s' from key '%s': %w", name, key, err)
	}
	// Ensure the Name field is populated from the requested name if not set in the config itself
	if llmConfig.Name == "" {
		llmConfig.Name = name
	}
	return llmConfig, nil
}

// Ensure ViperProvider implements the Provider interface.
var _ Provider = (*ViperProvider)(nil)

