// Package anthropic provides Anthropic provider implementation for multimodal models.
package anthropic

import (
	"errors"
	"time"
)

// Config holds Anthropic-specific configuration for multimodal operations.
type Config struct {
	APIKey     string        `mapstructure:"api_key" yaml:"api_key" env:"ANTHROPIC_API_KEY" validate:"required"`
	Model      string        `mapstructure:"model" yaml:"model" env:"ANTHROPIC_MODEL" validate:"required"`
	BaseURL    string        `mapstructure:"base_url" yaml:"base_url" env:"ANTHROPIC_BASE_URL" default:"https://api.anthropic.com/v1"`
	APIVersion string        `mapstructure:"api_version" yaml:"api_version" env:"ANTHROPIC_API_VERSION" default:"2023-06-01"`
	Timeout    time.Duration `mapstructure:"timeout" yaml:"timeout" env:"ANTHROPIC_TIMEOUT" default:"30s"`
	MaxRetries int           `mapstructure:"max_retries" yaml:"max_retries" env:"ANTHROPIC_MAX_RETRIES" default:"3"`
	Enabled    bool          `mapstructure:"enabled" yaml:"enabled" env:"ANTHROPIC_ENABLED" default:"true"`
}

// Validate validates the Anthropic configuration.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("Anthropic API key is required")
	}
	if c.Model == "" {
		return errors.New("Anthropic model is required")
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

// FromMultimodalConfig extracts Anthropic-specific config from multimodal.Config.
func FromMultimodalConfig(multimodalConfig MultimodalConfig) *Config {
	cfg := &Config{
		APIKey:     multimodalConfig.APIKey,
		Model:      multimodalConfig.Model,
		BaseURL:    multimodalConfig.BaseURL,
		Timeout:    multimodalConfig.Timeout,
		MaxRetries: multimodalConfig.MaxRetries,
	}

	// Extract Anthropic-specific config from ProviderSpecific
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
		cfg.BaseURL = "https://api.anthropic.com/v1"
	}
	if cfg.APIVersion == "" {
		cfg.APIVersion = "2023-06-01"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	return cfg
}
