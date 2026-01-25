// Package gemini provides configuration for the Google Gemini LLM provider.
package gemini

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/llms"
)

// GeminiConfig holds provider-specific configuration for Gemini.
type GeminiConfig struct {
	// APIKey is the Google AI Studio API key for authentication.
	APIKey string `mapstructure:"api_key" yaml:"api_key" env:"GEMINI_API_KEY" validate:"required"`

	// BaseURL is the base URL for the Gemini API.
	// Default: "https://generativelanguage.googleapis.com/v1beta"
	BaseURL string `mapstructure:"base_url" yaml:"base_url" env:"GEMINI_BASE_URL"`

	// ModelName is the Gemini model to use.
	// Default: "gemini-1.5-pro"
	ModelName string `mapstructure:"model_name" yaml:"model_name" env:"GEMINI_MODEL_NAME"`

	// Location is the Google Cloud location (optional, for Vertex AI).
	Location string `mapstructure:"location" yaml:"location" env:"GEMINI_LOCATION"`
}

// Validate validates the Gemini configuration.
func (c *GeminiConfig) Validate() error {
	if c.APIKey == "" {
		return errors.New("gemini API key is required")
	}

	if c.BaseURL == "" {
		c.BaseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	if c.ModelName == "" {
		c.ModelName = "gemini-1.5-pro"
	}

	// Validate base URL format
	if !strings.HasPrefix(c.BaseURL, "http://") && !strings.HasPrefix(c.BaseURL, "https://") {
		return fmt.Errorf("invalid base URL format: %s", c.BaseURL)
	}

	return nil
}

// ToLLMConfig converts GeminiConfig to llms.Config.
func (c *GeminiConfig) ToLLMConfig() *llms.Config {
	config := llms.DefaultConfig()
	config.Provider = ProviderName
	config.APIKey = c.APIKey
	config.BaseURL = c.BaseURL
	config.ModelName = c.ModelName

	if c.Location != "" {
		if config.ProviderSpecific == nil {
			config.ProviderSpecific = make(map[string]any)
		}
		config.ProviderSpecific["location"] = c.Location
	}

	return config
}

// FromLLMConfig creates a GeminiConfig from llms.Config.
func FromLLMConfig(config *llms.Config) *GeminiConfig {
	geminiConfig := &GeminiConfig{
		APIKey:  config.APIKey,
		BaseURL: config.BaseURL,
	}

	if config.ModelName != "" {
		geminiConfig.ModelName = config.ModelName
	} else {
		geminiConfig.ModelName = "gemini-1.5-pro"
	}

	if config.BaseURL == "" {
		geminiConfig.BaseURL = "https://generativelanguage.googleapis.com/v1beta"
	}

	if loc, ok := config.ProviderSpecific["location"].(string); ok {
		geminiConfig.Location = loc
	}

	return geminiConfig
}
