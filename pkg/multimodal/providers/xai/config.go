// Package xai provides xAI (Grok) provider implementation for multimodal models.
package xai

import (
	"errors"
	"time"
)

// Config holds xAI-specific configuration for multimodal operations.
type Config struct {
	APIKey     string        `mapstructure:"api_key" yaml:"api_key" env:"XAI_API_KEY" validate:"required"`
	Model      string        `mapstructure:"model" yaml:"model" env:"XAI_MODEL" validate:"required"`
	BaseURL    string        `mapstructure:"base_url" yaml:"base_url" env:"XAI_BASE_URL" default:"https://api.x.ai/v1"`
	Timeout    time.Duration `mapstructure:"timeout" yaml:"timeout" env:"XAI_TIMEOUT" default:"30s"`
	MaxRetries int           `mapstructure:"max_retries" yaml:"max_retries" env:"XAI_MAX_RETRIES" default:"3"`
	Enabled    bool          `mapstructure:"enabled" yaml:"enabled" env:"XAI_ENABLED" default:"true"`
}

// Validate validates the xAI configuration.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("xAI API key is required")
	}
	if c.Model == "" {
		return errors.New("xAI model is required")
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

// FromMultimodalConfig extracts xAI-specific config from multimodal.Config.
func FromMultimodalConfig(multimodalConfig MultimodalConfig) *Config {
	cfg := &Config{
		APIKey:     multimodalConfig.APIKey,
		Model:      multimodalConfig.Model,
		BaseURL:    multimodalConfig.BaseURL,
		Timeout:    multimodalConfig.Timeout,
		MaxRetries: multimodalConfig.MaxRetries,
	}

	// Extract xAI-specific config from ProviderSpecific
	if multimodalConfig.ProviderSpecific != nil {
		if enabled, ok := multimodalConfig.ProviderSpecific["enabled"].(bool); ok {
			cfg.Enabled = enabled
		}
	}

	// Set defaults
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.x.ai/v1"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	return cfg
}
