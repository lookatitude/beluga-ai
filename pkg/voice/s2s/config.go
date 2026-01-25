package s2s

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

// Config represents the configuration for S2S providers.
// It includes common settings that apply to all S2S providers.
type Config struct {
	ProviderSpecific        map[string]any `mapstructure:"provider_specific" yaml:"provider_specific"`
	Provider                string         `mapstructure:"provider" yaml:"provider" validate:"required,oneof=amazon_nova grok gemini openai_realtime mock"`
	APIKey                  string         `mapstructure:"api_key" yaml:"api_key" validate:"required_unless=Provider mock"`
	ReasoningMode           string         `mapstructure:"reasoning_mode" yaml:"reasoning_mode" default:"built-in" validate:"oneof=built-in external"`
	LatencyTarget           string         `mapstructure:"latency_target" yaml:"latency_target" default:"medium" validate:"oneof=low medium high"`
	Language                string         `mapstructure:"language" yaml:"language" validate:"omitempty,len=5"`
	FallbackProviders       []string       `mapstructure:"fallback_providers" yaml:"fallback_providers"`
	RetryDelay              time.Duration  `mapstructure:"retry_delay" yaml:"retry_delay" default:"1s" validate:"min=100ms,max=30s"`
	Timeout                 time.Duration  `mapstructure:"timeout" yaml:"timeout" default:"30s" validate:"min=1s,max=5m"`
	MaxRetries              int            `mapstructure:"max_retries" yaml:"max_retries" default:"3" validate:"gte=0,lte=10"`
	RetryBackoff            float64        `mapstructure:"retry_backoff" yaml:"retry_backoff" default:"2.0" validate:"gte=1,lte=5"`
	Channels                int            `mapstructure:"channels" yaml:"channels" default:"1" validate:"oneof=1 2"`
	SampleRate              int            `mapstructure:"sample_rate" yaml:"sample_rate" default:"24000" validate:"oneof=8000 16000 24000 48000"`
	MaxConcurrentSessions   int            `mapstructure:"max_concurrent_sessions" yaml:"max_concurrent_sessions" default:"50" validate:"gte=1,lte=1000"`
	EnableTracing           bool           `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
	EnableMetrics           bool           `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
	EnableStructuredLogging bool           `mapstructure:"enable_structured_logging" yaml:"enable_structured_logging" default:"true"`
}

// ConfigOption is a functional option for configuring S2S instances.
type ConfigOption func(*Config)

// WithProvider sets the S2S provider.
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

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(maxRetries int) ConfigOption {
	return func(c *Config) {
		c.MaxRetries = maxRetries
	}
}

// WithLatencyTarget sets the latency target.
func WithLatencyTarget(target string) ConfigOption {
	return func(c *Config) {
		c.LatencyTarget = target
	}
}

// WithReasoningMode sets the reasoning mode.
func WithReasoningMode(mode string) ConfigOption {
	return func(c *Config) {
		c.ReasoningMode = mode
	}
}

// WithFallbackProviders sets the fallback providers list.
func WithFallbackProviders(providers ...string) ConfigOption {
	return func(c *Config) {
		c.FallbackProviders = providers
	}
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		Provider:                "mock",
		SampleRate:              24000,
		Channels:                1,
		Timeout:                 30 * time.Second,
		RetryDelay:              1 * time.Second,
		MaxRetries:              3,
		RetryBackoff:            2.0,
		EnableTracing:           true,
		EnableMetrics:           true,
		EnableStructuredLogging: true,
		LatencyTarget:           "medium",
		ReasoningMode:           "built-in",
		MaxConcurrentSessions:   50,
		ProviderSpecific:        make(map[string]any),
		FallbackProviders:       []string{},
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	return nil
}
