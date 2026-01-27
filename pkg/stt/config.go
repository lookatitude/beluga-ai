package stt

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

// Config represents the configuration for STT providers.
// It includes common settings that apply to all STT providers.
type Config struct {
	ProviderSpecific        map[string]any `mapstructure:"provider_specific" yaml:"provider_specific"`
	Language                string         `mapstructure:"language" yaml:"language" validate:"omitempty,len=2"`
	BaseURL                 string         `mapstructure:"base_url" yaml:"base_url"`
	Model                   string         `mapstructure:"model" yaml:"model"`
	Provider                string         `mapstructure:"provider" yaml:"provider" validate:"required,oneof=deepgram google azure openai"`
	APIKey                  string         `mapstructure:"api_key" yaml:"api_key" validate:"required_unless=Provider mock"`
	RetryBackoff            float64        `mapstructure:"retry_backoff" yaml:"retry_backoff" default:"2.0" validate:"gte=1,lte=5"`
	SampleRate              int            `mapstructure:"sample_rate" yaml:"sample_rate" default:"16000" validate:"oneof=8000 16000 48000"`
	MaxRetries              int            `mapstructure:"max_retries" yaml:"max_retries" default:"3" validate:"gte=0,lte=10"`
	Channels                int            `mapstructure:"channels" yaml:"channels" default:"1" validate:"oneof=1 2"`
	RetryDelay              time.Duration  `mapstructure:"retry_delay" yaml:"retry_delay" default:"1s" validate:"min=100ms,max=30s"`
	Timeout                 time.Duration  `mapstructure:"timeout" yaml:"timeout" default:"30s" validate:"min=1s,max=5m"`
	EnableStreaming         bool           `mapstructure:"enable_streaming" yaml:"enable_streaming" default:"true"`
	EnableTracing           bool           `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
	EnableMetrics           bool           `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
	EnableStructuredLogging bool           `mapstructure:"enable_structured_logging" yaml:"enable_structured_logging" default:"true"`
}

// ConfigOption is a functional option for configuring STT instances.
type ConfigOption func(*Config)

// WithProvider sets the STT provider.
func WithProvider(provider string) ConfigOption {
	return func(c *Config) {
		c.Provider = provider
	}
}

// WithAPIKey sets the API key.
func WithAPIKey(apiKey string) ConfigOption {
	return func(c *Config) {
		c.APIKey = apiKey
	}
}

// WithBaseURL sets the base URL.
func WithBaseURL(baseURL string) ConfigOption {
	return func(c *Config) {
		c.BaseURL = baseURL
	}
}

// WithModel sets the model.
func WithModel(model string) ConfigOption {
	return func(c *Config) {
		c.Model = model
	}
}

// WithLanguage sets the language.
func WithLanguage(language string) ConfigOption {
	return func(c *Config) {
		c.Language = language
	}
}

// WithTimeout sets the timeout.
func WithTimeout(timeout time.Duration) ConfigOption {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithSampleRate sets the sample rate.
func WithSampleRate(sampleRate int) ConfigOption {
	return func(c *Config) {
		c.SampleRate = sampleRate
	}
}

// WithChannels sets the number of channels.
func WithChannels(channels int) ConfigOption {
	return func(c *Config) {
		c.Channels = channels
	}
}

// WithEnableStreaming sets streaming enablement.
func WithEnableStreaming(enable bool) ConfigOption {
	return func(c *Config) {
		c.EnableStreaming = enable
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("STT config validation failed: %w", err)
	}
	return nil
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		Provider:                "deepgram",
		Timeout:                 30 * time.Second,
		SampleRate:              16000,
		Channels:                1,
		EnableStreaming:         true,
		MaxRetries:              3,
		RetryDelay:              1 * time.Second,
		RetryBackoff:            2.0,
		EnableTracing:           true,
		EnableMetrics:           true,
		EnableStructuredLogging: true,
	}
}
