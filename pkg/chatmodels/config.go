package chatmodels

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
)

// Config represents the configuration for the chatmodels package.
// It includes settings for model behavior, generation parameters, and observability.
type Config struct {
	// Model configuration
	DefaultModel       string        `mapstructure:"default_model" yaml:"default_model" default:"gpt-3.5-turbo"`
	DefaultProvider    string        `mapstructure:"default_provider" yaml:"default_provider" default:"openai"`
	DefaultTemperature float32       `mapstructure:"default_temperature" yaml:"default_temperature" default:"0.7"`
	DefaultMaxTokens   int           `mapstructure:"default_max_tokens" yaml:"default_max_tokens" default:"1000"`
	DefaultTimeout     time.Duration `mapstructure:"default_timeout" yaml:"default_timeout" default:"30s"`

	// Generation parameters
	DefaultTopP            float32  `mapstructure:"default_top_p" yaml:"default_top_p" default:"1.0"`
	DefaultStopSequences   []string `mapstructure:"default_stop_sequences" yaml:"default_stop_sequences"`
	DefaultSystemPrompt    string   `mapstructure:"default_system_prompt" yaml:"default_system_prompt"`
	DefaultFunctionCalling bool     `mapstructure:"default_function_calling" yaml:"default_function_calling" default:"false"`

	// Retry and error handling
	DefaultMaxRetries  int           `mapstructure:"default_max_retries" yaml:"default_max_retries" default:"3"`
	DefaultRetryDelay  time.Duration `mapstructure:"default_retry_delay" yaml:"default_retry_delay" default:"2s"`
	MaxRetryDelay      time.Duration `mapstructure:"max_retry_delay" yaml:"max_retry_delay" default:"30s"`
	RetryBackoffFactor float64       `mapstructure:"retry_backoff_factor" yaml:"retry_backoff_factor" default:"2.0"`

	// Observability settings
	EnableMetrics      bool   `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
	EnableTracing      bool   `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
	MetricsPrefix      string `mapstructure:"metrics_prefix" yaml:"metrics_prefix" default:"beluga_chatmodels"`
	TracingServiceName string `mapstructure:"tracing_service_name" yaml:"tracing_service_name" default:"beluga-chatmodels"`

	// Streaming configuration
	DefaultStreamingEnabled bool          `mapstructure:"default_streaming_enabled" yaml:"default_streaming_enabled" default:"false"`
	StreamBufferSize        int           `mapstructure:"stream_buffer_size" yaml:"stream_buffer_size" default:"100"`
	StreamTimeout           time.Duration `mapstructure:"stream_timeout" yaml:"stream_timeout" default:"5m"`

	// Resource limits
	MaxConcurrentRequests int           `mapstructure:"max_concurrent_requests" yaml:"max_concurrent_requests" default:"100"`
	RequestTimeout        time.Duration `mapstructure:"request_timeout" yaml:"request_timeout" default:"2m"`
	ConnectionTimeout     time.Duration `mapstructure:"connection_timeout" yaml:"connection_timeout" default:"10s"`

	// Provider-specific configurations
	Providers map[string]interface{} `mapstructure:"providers" yaml:"providers"`
}

// ProviderConfig represents configuration for a specific provider.
type ProviderConfig struct {
	APIKey     string          `mapstructure:"api_key" yaml:"api_key"`
	BaseURL    string          `mapstructure:"base_url" yaml:"base_url"`
	Timeout    time.Duration   `mapstructure:"timeout" yaml:"timeout" default:"30s"`
	MaxRetries int             `mapstructure:"max_retries" yaml:"max_retries" default:"3"`
	RateLimit  RateLimitConfig `mapstructure:"rate_limit" yaml:"rate_limit"`
}

// RateLimitConfig represents rate limiting configuration.
type RateLimitConfig struct {
	RequestsPerMinute int `mapstructure:"requests_per_minute" yaml:"requests_per_minute" default:"60"`
	RequestsPerHour   int `mapstructure:"requests_per_hour" yaml:"requests_per_hour" default:"1000"`
	BurstSize         int `mapstructure:"burst_size" yaml:"burst_size" default:"10"`
}

// WithTemperature sets the temperature for response generation.
func WithTemperature(temp float32) iface.Option {
	return iface.OptionFunc(func(config *map[string]any) {
		if *config == nil {
			*config = make(map[string]any)
		}
		(*config)["temperature"] = temp
	})
}

// WithMaxTokens sets the maximum number of tokens to generate.
func WithMaxTokens(maxTokens int) iface.Option {
	return iface.OptionFunc(func(config *map[string]any) {
		if *config == nil {
			*config = make(map[string]any)
		}
		(*config)["max_tokens"] = maxTokens
	})
}

// WithTopP sets the nucleus sampling parameter.
func WithTopP(topP float32) iface.Option {
	return iface.OptionFunc(func(config *map[string]any) {
		if *config == nil {
			*config = make(map[string]any)
		}
		(*config)["top_p"] = topP
	})
}

// WithStopSequences sets the stop sequences for generation.
func WithStopSequences(sequences []string) iface.Option {
	return iface.OptionFunc(func(config *map[string]any) {
		if *config == nil {
			*config = make(map[string]any)
		}
		(*config)["stop_sequences"] = sequences
	})
}

// WithSystemPrompt sets the system prompt.
func WithSystemPrompt(prompt string) iface.Option {
	return iface.OptionFunc(func(config *map[string]any) {
		if *config == nil {
			*config = make(map[string]any)
		}
		(*config)["system_prompt"] = prompt
	})
}

// WithFunctionCalling enables or disables function calling.
func WithFunctionCalling(enabled bool) iface.Option {
	return iface.OptionFunc(func(config *map[string]any) {
		if *config == nil {
			*config = make(map[string]any)
		}
		(*config)["function_calling"] = enabled
	})
}

// WithTimeout sets the timeout for operations.
func WithTimeout(timeout time.Duration) iface.Option {
	return iface.OptionFunc(func(config *map[string]any) {
		if *config == nil {
			*config = make(map[string]any)
		}
		(*config)["timeout"] = timeout
	})
}

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(retries int) iface.Option {
	return iface.OptionFunc(func(config *map[string]any) {
		if *config == nil {
			*config = make(map[string]any)
		}
		(*config)["max_retries"] = retries
	})
}

// WithMetrics enables or disables metrics collection.
func WithMetrics(enabled bool) iface.Option {
	return iface.OptionFunc(func(config *map[string]any) {
		if *config == nil {
			*config = make(map[string]any)
		}
		(*config)["enable_metrics"] = enabled
	})
}

// WithTracing enables or disables tracing.
func WithTracing(enabled bool) iface.Option {
	return iface.OptionFunc(func(config *map[string]any) {
		if *config == nil {
			*config = make(map[string]any)
		}
		(*config)["enable_tracing"] = enabled
	})
}

// DefaultConfig returns a default configuration for the chatmodels package.
func DefaultConfig() *Config {
	return &Config{
		DefaultModel:            "gpt-3.5-turbo",
		DefaultProvider:         "openai",
		DefaultTemperature:      0.7,
		DefaultMaxTokens:        1000,
		DefaultTimeout:          30 * time.Second,
		DefaultTopP:             1.0,
		DefaultStopSequences:    []string{},
		DefaultSystemPrompt:     "",
		DefaultFunctionCalling:  false,
		DefaultMaxRetries:       3,
		DefaultRetryDelay:       2 * time.Second,
		MaxRetryDelay:           30 * time.Second,
		RetryBackoffFactor:      2.0,
		EnableMetrics:           true,
		EnableTracing:           true,
		MetricsPrefix:           "beluga_chatmodels",
		TracingServiceName:      "beluga-chatmodels",
		DefaultStreamingEnabled: false,
		StreamBufferSize:        100,
		StreamTimeout:           5 * time.Minute,
		MaxConcurrentRequests:   100,
		RequestTimeout:          2 * time.Minute,
		ConnectionTimeout:       10 * time.Second,
		Providers:               make(map[string]interface{}),
	}
}

// Validate validates the configuration and returns an error if invalid.
func (c *Config) Validate() error {
	if c.DefaultTemperature < 0 || c.DefaultTemperature > 2 {
		return NewValidationError("default_temperature", "must be between 0 and 2")
	}
	if c.DefaultMaxTokens <= 0 {
		return NewValidationError("default_max_tokens", "must be positive")
	}
	if c.DefaultTimeout <= 0 {
		return NewValidationError("default_timeout", "must be positive")
	}
	if c.DefaultMaxRetries < 0 {
		return NewValidationError("default_max_retries", "cannot be negative")
	}
	if c.DefaultRetryDelay < 0 {
		return NewValidationError("default_retry_delay", "cannot be negative")
	}
	if c.MaxConcurrentRequests <= 0 {
		return NewValidationError("max_concurrent_requests", "must be positive")
	}
	if c.StreamBufferSize <= 0 {
		return NewValidationError("stream_buffer_size", "must be positive")
	}
	if c.StreamTimeout <= 0 {
		return NewValidationError("stream_timeout", "must be positive")
	}

	return nil
}

// GetProviderConfig returns the configuration for a specific provider.
func (c *Config) GetProviderConfig(provider string) (*ProviderConfig, error) {
	if config, exists := c.Providers[provider]; exists {
		if pc, ok := config.(*ProviderConfig); ok {
			return pc, nil
		}
		return nil, NewValidationError("provider_config", "invalid provider configuration type")
	}

	// Return default provider config if none exists
	return &ProviderConfig{
		Timeout:    c.DefaultTimeout,
		MaxRetries: c.DefaultMaxRetries,
	}, nil
}
