// Package gemini provides Google Gemini provider implementation for multimodal models.
package gemini

import (
	"errors"
	"time"
)

// Config holds Gemini-specific configuration for multimodal operations.
type Config struct {
	APIKey     string        `mapstructure:"api_key" yaml:"api_key" env:"GEMINI_API_KEY" validate:"required"`
	Model      string        `mapstructure:"model" yaml:"model" env:"GEMINI_MODEL" validate:"required"`
	BaseURL    string        `mapstructure:"base_url" yaml:"base_url" env:"GEMINI_BASE_URL" default:"https://generativelanguage.googleapis.com/v1beta"`
	Timeout    time.Duration `mapstructure:"timeout" yaml:"timeout" env:"GEMINI_TIMEOUT" default:"30s"`
	MaxRetries int           `mapstructure:"max_retries" yaml:"max_retries" env:"GEMINI_MAX_RETRIES" default:"3"`
	Enabled    bool          `mapstructure:"enabled" yaml:"enabled" env:"GEMINI_ENABLED" default:"true"`
}

// Validate validates the Gemini configuration.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("Gemini API key is required")
	}
	if c.Model == "" {
		return errors.New("Gemini model is required")
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

// FromMultimodalConfig extracts Gemini-specific config from multimodal.Config.
func FromMultimodalConfig(multimodalConfig MultimodalConfig) *Config {
	cfg := &Config{
		APIKey:     multimodalConfig.APIKey,
		Model:      multimodalConfig.Model,
		BaseURL:    multimodalConfig.BaseURL,
		Timeout:    multimodalConfig.Timeout,
		MaxRetries: multimodalConfig.MaxRetries,
	}

	// Extract Gemini-specific config from ProviderSpecific
	if multimodalConfig.ProviderSpecific != nil {
		if enabled, ok := multimodalConfig.ProviderSpecific["enabled"].(bool); ok {
			cfg.Enabled = enabled
		}
	}

	// Set defaults
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	return cfg
}
