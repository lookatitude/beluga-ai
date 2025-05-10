package schema

// EmbeddingProviderConfig defines the configuration for a specific embedding provider instance.
type EmbeddingProviderConfig struct {
	Name      string `yaml:"name" json:"name" mapstructure:"name"`
	Provider  string `yaml:"provider" json:"provider" mapstructure:"provider"`
	ModelName string `yaml:"model_name" json:"model_name" mapstructure:"model_name"`
	APIKey    string `yaml:"api_key" json:"api_key" mapstructure:"api_key"`
	BaseURL   string `yaml:"base_url,omitempty" json:"base_url,omitempty" mapstructure:"base_url,omitempty"`
	ProviderSpecific map[string]interface{} `yaml:"provider_specific,omitempty" json:"provider_specific,omitempty" mapstructure:"provider_specific,omitempty"`
}

