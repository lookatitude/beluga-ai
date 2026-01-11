// Package pixtral provides Pixtral (Mistral AI) provider implementation for multimodal models.
package pixtral

import (
	"errors"
	"time"
)

// Config holds Pixtral-specific configuration for multimodal operations.
type Config struct {
	APIKey     string        `mapstructure:"api_key" yaml:"apiKey" env:"PIXTRAL_API_KEY" validate:"required"`
	Model      string        `mapstructure:"model" yaml:"model" env:"PIXTRAL_MODEL" validate:"required"`
	BaseURL    string        `mapstructure:"base_url" yaml:"base_url" env:"PIXTRAL_BASE_URL" default:"https://api.mistral.ai/v1"`
	Timeout    time.Duration `mapstructure:"timeout" yaml:"timeout" env:"PIXTRAL_TIMEOUT" default:"30s"`
	MaxRetries int           `mapstructure:"max_retries" yaml:"maxRetries" env:"PIXTRAL_MAX_RETRIES" default:"3"`
	Enabled    bool          `mapstructure:"enabled" yaml:"enabled" env:"PIXTRAL_ENABLED" default:"true"`
}

// Validate validates the Pixtral configuration.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("pixtral API key is required")
	}
	if c.Model == "" {
		return errors.New("pixtral model is required")
	}
	return nil
}

// MultimodalConfig represents the multimodal.Config type without importing the package.
// This avoids import cycles.
type MultimodalConfig struct {
	Provider         string
	Model            string
	APIKey           string
	BaseURL          string
	Timeout          time.Duration
	MaxRetries       int
	ProviderSpecific map[string]any
}

// FromMultimodalConfig extracts Pixtral-specific config from multimodal.Config.
func FromMultimodalConfig(multimodalConfig MultimodalConfig) *Config {
	cfg := &Config{
		APIKey:     multimodalConfig.APIKey,
		Model:      multimodalConfig.Model,
		BaseURL:    multimodalConfig.BaseURL,
		Timeout:    multimodalConfig.Timeout,
		MaxRetries: multimodalConfig.MaxRetries,
	}

	// Extract Pixtral-specific config from ProviderSpecific
	if multimodalConfig.ProviderSpecific != nil {
		if enabled, ok := multimodalConfig.ProviderSpecific["enabled"].(bool); ok {
			cfg.Enabled = enabled
		}
	}

	// Set defaults
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.mistral.ai/v1"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	return cfg
}
