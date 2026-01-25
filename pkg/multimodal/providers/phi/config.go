// Package phi provides Phi (Microsoft) provider implementation for multimodal models.
package phi

import (
	"errors"
	"time"
)

// Config holds Phi-specific configuration for multimodal operations.
type Config struct {
	APIKey     string        `mapstructure:"api_key" yaml:"api_key" env:"PHI_API_KEY" validate:"required"`
	Model      string        `mapstructure:"model" yaml:"model" env:"PHI_MODEL" validate:"required"`
	BaseURL    string        `mapstructure:"base_url" yaml:"base_url" env:"PHI_BASE_URL" default:"https://api-inference.huggingface.co/models"`
	Timeout    time.Duration `mapstructure:"timeout" yaml:"timeout" env:"PHI_TIMEOUT" default:"30s"`
	MaxRetries int           `mapstructure:"max_retries" yaml:"max_retries" env:"PHI_MAX_RETRIES" default:"3"`
	Enabled    bool          `mapstructure:"enabled" yaml:"enabled" env:"PHI_ENABLED" default:"true"`
}

// Validate validates the Phi configuration.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return errors.New("Phi API key is required")
	}
	if c.Model == "" {
		return errors.New("Phi model is required")
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

// FromMultimodalConfig extracts Phi-specific config from multimodal.Config.
func FromMultimodalConfig(multimodalConfig MultimodalConfig) *Config {
	cfg := &Config{
		APIKey:     multimodalConfig.APIKey,
		Model:      multimodalConfig.Model,
		BaseURL:    multimodalConfig.BaseURL,
		Timeout:    multimodalConfig.Timeout,
		MaxRetries: multimodalConfig.MaxRetries,
	}

	// Extract Phi-specific config from ProviderSpecific
	if multimodalConfig.ProviderSpecific != nil {
		if enabled, ok := multimodalConfig.ProviderSpecific["enabled"].(bool); ok {
			cfg.Enabled = enabled
		}
	}

	// Set defaults
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api-inference.huggingface.co/models"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	return cfg
}
