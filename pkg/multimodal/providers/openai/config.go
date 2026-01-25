// Package openai provides OpenAI provider implementation for multimodal models.
package openai

import (
	"errors"
	"time"
)

// Config holds OpenAI-specific configuration for multimodal operations.
type Config struct {
	APIKey     string        `mapstructure:"api_key" yaml:"api_key" env:"OPENAI_API_KEY" validate:"required"`
	Model      string        `mapstructure:"model" yaml:"model" env:"OPENAI_MODEL" validate:"required"`
	BaseURL    string        `mapstructure:"base_url" yaml:"base_url" env:"OPENAI_BASE_URL" default:"https://api.openai.com/v1"`
	APIVersion string        `mapstructure:"api_version" yaml:"api_version" env:"OPENAI_API_VERSION"`
	Timeout    time.Duration `mapstructure:"timeout" yaml:"timeout" env:"OPENAI_TIMEOUT" default:"30s"`
	MaxRetries int           `mapstructure:"max_retries" yaml:"max_retries" env:"OPENAI_MAX_RETRIES" default:"3"`
	Enabled    bool          `mapstructure:"enabled" yaml:"enabled" env:"OPENAI_ENABLED" default:"true"`
}

// Validate validates the OpenAI configuration.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("OpenAI API key is required")
	}
	if c.Model == "" {
		return errors.New("OpenAI model is required")
	}
	return nil
}

// MultimodalConfig represents the multimodal.Config type without importing the package.
// This avoids import cycles.
type MultimodalConfig struct {
	ProviderSpecific map[string]any
	Provider         string
	Model            string
	APIKey           string
	BaseURL          string
	Timeout          time.Duration
	MaxRetries       int
}

// FromMultimodalConfig extracts OpenAI-specific config from multimodal.Config.
func FromMultimodalConfig(multimodalConfig MultimodalConfig) *Config {
	cfg := &Config{
		APIKey:     multimodalConfig.APIKey,
		Model:      multimodalConfig.Model,
		BaseURL:    multimodalConfig.BaseURL,
		Timeout:    multimodalConfig.Timeout,
		MaxRetries: multimodalConfig.MaxRetries,
	}

	// Extract OpenAI-specific config from ProviderSpecific
	if multimodalConfig.ProviderSpecific != nil {
		if apiVersion, ok := multimodalConfig.ProviderSpecific["api_version"].(string); ok {
			cfg.APIVersion = apiVersion
		}
		if enabled, ok := multimodalConfig.ProviderSpecific["enabled"].(bool); ok {
			cfg.Enabled = enabled
		}
	}

	// Set defaults
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	return cfg
}
