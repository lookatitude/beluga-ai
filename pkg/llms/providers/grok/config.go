// Package grok provides configuration for the Grok (xAI) LLM provider.
package grok

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/llms"
)

// GrokConfig holds provider-specific configuration for Grok.
type GrokConfig struct {
	// APIKey is the xAI API key for authentication.
	APIKey string `mapstructure:"api_key" yaml:"api_key" env:"GROK_API_KEY" validate:"required"`

	// BaseURL is the base URL for the Grok API.
	// Default: "https://api.x.ai/v1"
	BaseURL string `mapstructure:"base_url" yaml:"base_url" env:"GROK_BASE_URL"`

	// ModelName is the Grok model to use.
	// Default: "grok-beta"
	ModelName string `mapstructure:"model_name" yaml:"model_name" env:"GROK_MODEL_NAME"`

	// Organization is the xAI organization ID (optional).
	Organization string `mapstructure:"organization" yaml:"organization" env:"GROK_ORGANIZATION"`
}

// Validate validates the Grok configuration.
func (c *GrokConfig) Validate() error {
	if c.APIKey == "" {
		return errors.New("grok API key is required")
	}

	if c.BaseURL == "" {
		c.BaseURL = "https://api.x.ai/v1"
	}

	if c.ModelName == "" {
		c.ModelName = "grok-beta"
	}

	// Validate base URL format
	if !strings.HasPrefix(c.BaseURL, "http://") && !strings.HasPrefix(c.BaseURL, "https://") {
		return fmt.Errorf("invalid base URL format: %s", c.BaseURL)
	}

	return nil
}

// ToLLMConfig converts GrokConfig to llms.Config.
func (c *GrokConfig) ToLLMConfig() *llms.Config {
	config := llms.DefaultConfig()
	config.Provider = ProviderName
	config.APIKey = c.APIKey
	config.BaseURL = c.BaseURL
	config.ModelName = c.ModelName

	if c.Organization != "" {
		if config.ProviderSpecific == nil {
			config.ProviderSpecific = make(map[string]any)
		}
		config.ProviderSpecific["organization"] = c.Organization
	}

	return config
}

// FromLLMConfig creates a GrokConfig from llms.Config.
func FromLLMConfig(config *llms.Config) *GrokConfig {
	grokConfig := &GrokConfig{
		APIKey:  config.APIKey,
		BaseURL: config.BaseURL,
	}

	if config.ModelName != "" {
		grokConfig.ModelName = config.ModelName
	} else {
		grokConfig.ModelName = "grok-beta"
	}

	if config.BaseURL == "" {
		grokConfig.BaseURL = "https://api.x.ai/v1"
	}

	if org, ok := config.ProviderSpecific["organization"].(string); ok {
		grokConfig.Organization = org
	}

	return grokConfig
}
