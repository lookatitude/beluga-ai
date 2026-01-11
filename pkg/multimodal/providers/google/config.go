// Package google provides Google Vertex AI provider implementation for multimodal models.
package google

import (
	"errors"
	"time"
)

// Config holds Google Vertex AI-specific configuration for multimodal operations.
type Config struct {
	// ProjectID is the Google Cloud project ID (required for Vertex AI).
	ProjectID string `mapstructure:"project_id" yaml:"project_id" env:"GOOGLE_PROJECT_ID" validate:"required"`

	// Location is the Google Cloud location/region (e.g., "us-central1").
	Location string `mapstructure:"location" yaml:"location" env:"GOOGLE_LOCATION" default:"us-central1"`

	// APIKey is optional - can use service account credentials instead.
	APIKey string `mapstructure:"api_key" yaml:"api_key" env:"GOOGLE_API_KEY"`

	// Model is the Vertex AI model name (e.g., "gemini-1.5-pro").
	Model string `mapstructure:"model" yaml:"model" env:"GOOGLE_MODEL" validate:"required"`

	// BaseURL is the Vertex AI API base URL.
	BaseURL string `mapstructure:"base_url" yaml:"base_url" env:"GOOGLE_BASE_URL" default:"https://us-central1-aiplatform.googleapis.com/v1"`

	// Timeout for API requests.
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" env:"GOOGLE_TIMEOUT" default:"30s"`

	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int `mapstructure:"max_retries" yaml:"max_retries" env:"GOOGLE_MAX_RETRIES" default:"3"`

	// Enabled indicates if the provider is enabled.
	Enabled bool `mapstructure:"enabled" yaml:"enabled" env:"GOOGLE_ENABLED" default:"true"`
}

// Validate validates the Google Vertex AI configuration.
func (c *Config) Validate() error {
	if c.ProjectID == "" {
		return errors.New("Google project ID is required for Vertex AI")
	}
	if c.Model == "" {
		return errors.New("Google model is required")
	}
	if c.Location == "" {
		c.Location = "us-central1"
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

// FromMultimodalConfig extracts Google-specific config from multimodal.Config.
func FromMultimodalConfig(multimodalConfig MultimodalConfig) *Config {
	cfg := &Config{
		APIKey:     multimodalConfig.APIKey,
		Model:      multimodalConfig.Model,
		BaseURL:    multimodalConfig.BaseURL,
		Timeout:    multimodalConfig.Timeout,
		MaxRetries: multimodalConfig.MaxRetries,
	}

	// Extract Google-specific config from ProviderSpecific
	if multimodalConfig.ProviderSpecific != nil {
		if projectID, ok := multimodalConfig.ProviderSpecific["project_id"].(string); ok {
			cfg.ProjectID = projectID
		}
		if location, ok := multimodalConfig.ProviderSpecific["location"].(string); ok {
			cfg.Location = location
		}
		if enabled, ok := multimodalConfig.ProviderSpecific["enabled"].(bool); ok {
			cfg.Enabled = enabled
		}
	}

	// Set defaults
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://us-central1-aiplatform.googleapis.com/v1"
	}
	if cfg.Location == "" {
		cfg.Location = "us-central1"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	return cfg
}
