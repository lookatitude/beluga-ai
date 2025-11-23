package stt

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// Config represents the configuration for STT providers.
// It includes common settings that apply to all STT providers.
type Config struct {
	// Provider specifies the STT provider (e.g., "deepgram", "google", "azure", "openai")
	Provider string `mapstructure:"provider" yaml:"provider" validate:"required,oneof=deepgram google azure openai"`

	// APIKey for authentication (required for most providers)
	APIKey string `mapstructure:"api_key" yaml:"api_key" validate:"required_unless=Provider mock"`

	// BaseURL for custom API endpoints (optional)
	BaseURL string `mapstructure:"base_url" yaml:"base_url"`

	// Model specifies the model to use (provider-specific)
	Model string `mapstructure:"model" yaml:"model"`

	// Language specifies the language code (ISO 639-1, e.g., "en", "es")
	Language string `mapstructure:"language" yaml:"language" validate:"omitempty,len=2"`

	// Timeout for API calls
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s" validate:"min=1s,max=5m"`

	// SampleRate specifies the audio sample rate in Hz
	SampleRate int `mapstructure:"sample_rate" yaml:"sample_rate" default:"16000" validate:"oneof=8000 16000 48000"`

	// Channels specifies the number of audio channels (1 for mono, 2 for stereo)
	Channels int `mapstructure:"channels" yaml:"channels" default:"1" validate:"oneof=1 2"`

	// EnableStreaming enables streaming transcription
	EnableStreaming bool `mapstructure:"enable_streaming" yaml:"enable_streaming" default:"true"`

	// Retry configuration
	MaxRetries   int           `mapstructure:"max_retries" yaml:"max_retries" default:"3" validate:"gte=0,lte=10"`
	RetryDelay   time.Duration `mapstructure:"retry_delay" yaml:"retry_delay" default:"1s" validate:"min=100ms,max=30s"`
	RetryBackoff float64       `mapstructure:"retry_backoff" yaml:"retry_backoff" default:"2.0" validate:"gte=1,lte=5"`

	// Provider-specific configuration
	ProviderSpecific map[string]interface{} `mapstructure:"provider_specific" yaml:"provider_specific"`

	// Observability settings
	EnableTracing           bool `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
	EnableMetrics           bool `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
	EnableStructuredLogging bool `mapstructure:"enable_structured_logging" yaml:"enable_structured_logging" default:"true"`
}

// ConfigOption is a functional option for configuring STT instances
type ConfigOption func(*Config)

// WithProvider sets the STT provider
func WithProvider(provider string) ConfigOption {
	return func(c *Config) {
		c.Provider = provider
	}
}

// WithAPIKey sets the API key
func WithAPIKey(apiKey string) ConfigOption {
	return func(c *Config) {
		c.APIKey = apiKey
	}
}

// WithBaseURL sets the base URL
func WithBaseURL(baseURL string) ConfigOption {
	return func(c *Config) {
		c.BaseURL = baseURL
	}
}

// WithModel sets the model
func WithModel(model string) ConfigOption {
	return func(c *Config) {
		c.Model = model
	}
}

// WithLanguage sets the language
func WithLanguage(language string) ConfigOption {
	return func(c *Config) {
		c.Language = language
	}
}

// WithTimeout sets the timeout
func WithTimeout(timeout time.Duration) ConfigOption {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithSampleRate sets the sample rate
func WithSampleRate(sampleRate int) ConfigOption {
	return func(c *Config) {
		c.SampleRate = sampleRate
	}
}

// WithChannels sets the number of channels
func WithChannels(channels int) ConfigOption {
	return func(c *Config) {
		c.Channels = channels
	}
}

// WithEnableStreaming sets streaming enablement
func WithEnableStreaming(enable bool) ConfigOption {
	return func(c *Config) {
		c.EnableStreaming = enable
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

// DefaultConfig returns a default configuration
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
