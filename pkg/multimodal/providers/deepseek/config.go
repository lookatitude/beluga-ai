// Package deepseek provides DeepSeek provider implementation for multimodal models.
package deepseek

import (
	"errors"
	"time"
)

// Config holds DeepSeek-specific configuration for multimodal operations.
type Config struct {
	APIKey     string        `mapstructure:"api_key" yaml:"api_key" env:"DEEPSEEK_API_KEY" validate:"required"`
	Model      string        `mapstructure:"model" yaml:"model" env:"DEEPSEEK_MODEL" validate:"required"`
	BaseURL    string        `mapstructure:"base_url" yaml:"base_url" env:"DEEPSEEK_BASE_URL" default:"https://api.deepseek.com"`
	Timeout    time.Duration `mapstructure:"timeout" yaml:"timeout" env:"DEEPSEEK_TIMEOUT" default:"30s"`
	MaxRetries int           `mapstructure:"max_retries" yaml:"max_retries" env:"DEEPSEEK_MAX_RETRIES" default:"3"`
	Enabled    bool          `mapstructure:"enabled" yaml:"enabled" env:"DEEPSEEK_ENABLED" default:"true"`
}

// Validate validates the DeepSeek configuration.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("DeepSeek API key is required")
	}
	if c.Model == "" {
		return errors.New("DeepSeek model is required")
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

// FromMultimodalConfig extracts DeepSeek-specific config from multimodal.Config.
func FromMultimodalConfig(multimodalConfig MultimodalConfig) *Config {
	cfg := &Config{
		APIKey:     multimodalConfig.APIKey,
		Model:      multimodalConfig.Model,
		BaseURL:    multimodalConfig.BaseURL,
		Timeout:    multimodalConfig.Timeout,
		MaxRetries: multimodalConfig.MaxRetries,
	}

	// Extract DeepSeek-specific config from ProviderSpecific
	if multimodalConfig.ProviderSpecific != nil {
		if enabled, ok := multimodalConfig.ProviderSpecific["enabled"].(bool); ok {
			cfg.Enabled = enabled
		}
	}

	// Set defaults
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.deepseek.com"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	return cfg
}
