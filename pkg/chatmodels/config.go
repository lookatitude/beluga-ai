package chatmodels

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
)

// ConfigProvider is an interface for getting the default provider name.
// This interface is used by the registry to avoid import cycles.
type ConfigProvider interface {
	GetDefaultProvider() string
}

// Config represents the configuration for the chatmodels package.
// It includes settings for model behavior, generation parameters, and observability.
type Config struct {
	Providers               map[string]any `mapstructure:"providers" yaml:"providers"`
	DefaultSystemPrompt     string         `mapstructure:"default_system_prompt" yaml:"default_system_prompt" env:"CHATMODEL_DEFAULT_SYSTEM_PROMPT"`
	DefaultProvider         string         `mapstructure:"default_provider" yaml:"default_provider" env:"CHATMODEL_DEFAULT_PROVIDER" default:"openai"`
	TracingServiceName      string         `mapstructure:"tracing_service_name" yaml:"tracing_service_name" env:"CHATMODEL_TRACING_SERVICE_NAME" default:"beluga-chatmodels"`
	MetricsPrefix           string         `mapstructure:"metrics_prefix" yaml:"metrics_prefix" env:"CHATMODEL_METRICS_PREFIX" default:"beluga_chatmodels"`
	DefaultModel            string         `mapstructure:"default_model" yaml:"default_model" env:"CHATMODEL_DEFAULT_MODEL" default:"gpt-3.5-turbo"`
	DefaultStopSequences    []string       `mapstructure:"default_stop_sequences" yaml:"default_stop_sequences" env:"CHATMODEL_DEFAULT_STOP_SEQUENCES"`
	RetryBackoffFactor      float64        `mapstructure:"retry_backoff_factor" yaml:"retry_backoff_factor" env:"CHATMODEL_RETRY_BACKOFF_FACTOR" validate:"gt=0" default:"2.0"`
	MaxConcurrentRequests   int            `mapstructure:"max_concurrent_requests" yaml:"max_concurrent_requests" env:"CHATMODEL_MAX_CONCURRENT_REQUESTS" validate:"gt=0" default:"100"`
	DefaultMaxRetries       int            `mapstructure:"default_max_retries" yaml:"default_max_retries" env:"CHATMODEL_DEFAULT_MAX_RETRIES" validate:"gte=0" default:"3"`
	DefaultRetryDelay       time.Duration  `mapstructure:"default_retry_delay" yaml:"default_retry_delay" env:"CHATMODEL_DEFAULT_RETRY_DELAY" validate:"gt=0" default:"2s"`
	MaxRetryDelay           time.Duration  `mapstructure:"max_retry_delay" yaml:"max_retry_delay" env:"CHATMODEL_MAX_RETRY_DELAY" validate:"gt=0" default:"30s"`
	ConnectionTimeout       time.Duration  `mapstructure:"connection_timeout" yaml:"connection_timeout" env:"CHATMODEL_CONNECTION_TIMEOUT" validate:"gt=0" default:"10s"`
	RequestTimeout          time.Duration  `mapstructure:"request_timeout" yaml:"request_timeout" env:"CHATMODEL_REQUEST_TIMEOUT" validate:"gt=0" default:"2m"`
	StreamTimeout           time.Duration  `mapstructure:"stream_timeout" yaml:"stream_timeout" env:"CHATMODEL_STREAM_TIMEOUT" validate:"gt=0" default:"5m"`
	DefaultTimeout          time.Duration  `mapstructure:"default_timeout" yaml:"default_timeout" env:"CHATMODEL_DEFAULT_TIMEOUT" validate:"gt=0" default:"30s"`
	DefaultMaxTokens        int            `mapstructure:"default_max_tokens" yaml:"default_max_tokens" env:"CHATMODEL_DEFAULT_MAX_TOKENS" validate:"gt=0" default:"1000"`
	StreamBufferSize        int            `mapstructure:"stream_buffer_size" yaml:"stream_buffer_size" env:"CHATMODEL_STREAM_BUFFER_SIZE" validate:"gt=0" default:"100"`
	DefaultTopP             float32        `mapstructure:"default_top_p" yaml:"default_top_p" env:"CHATMODEL_DEFAULT_TOP_P" validate:"gte=0,lte=1" default:"1.0"`
	DefaultTemperature      float32        `mapstructure:"default_temperature" yaml:"default_temperature" env:"CHATMODEL_DEFAULT_TEMPERATURE" validate:"gte=0,lte=2" default:"0.7"`
	DefaultStreamingEnabled bool           `mapstructure:"default_streaming_enabled" yaml:"default_streaming_enabled" env:"CHATMODEL_DEFAULT_STREAMING_ENABLED" default:"false"`
	EnableTracing           bool           `mapstructure:"enable_tracing" yaml:"enable_tracing" env:"CHATMODEL_ENABLE_TRACING" default:"true"`
	DefaultFunctionCalling  bool           `mapstructure:"default_function_calling" yaml:"default_function_calling" env:"CHATMODEL_DEFAULT_FUNCTION_CALLING" default:"false"`
	EnableMetrics           bool           `mapstructure:"enable_metrics" yaml:"enable_metrics" env:"CHATMODEL_ENABLE_METRICS" default:"true"`
}

// GetDefaultProvider returns the default provider name.
// This method implements ConfigProvider interface to avoid import cycles.
func (c *Config) GetDefaultProvider() string {
	return c.DefaultProvider
}

// ProviderConfig represents configuration for a specific provider.
type ProviderConfig struct {
	APIKey     string          `mapstructure:"api_key" yaml:"api_key" env:"CHATMODEL_PROVIDER_API_KEY"`
	BaseURL    string          `mapstructure:"base_url" yaml:"base_url" env:"CHATMODEL_PROVIDER_BASE_URL"`
	Timeout    time.Duration   `mapstructure:"timeout" yaml:"timeout" env:"CHATMODEL_PROVIDER_TIMEOUT" validate:"gt=0" default:"30s"`
	MaxRetries int             `mapstructure:"max_retries" yaml:"max_retries" env:"CHATMODEL_PROVIDER_MAX_RETRIES" validate:"gte=0" default:"3"`
	RateLimit  RateLimitConfig `mapstructure:"rate_limit" yaml:"rate_limit"`
}

// RateLimitConfig represents rate limiting configuration.
type RateLimitConfig struct {
	RequestsPerMinute int `mapstructure:"requests_per_minute" yaml:"requests_per_minute" env:"CHATMODEL_RATE_LIMIT_REQUESTS_PER_MINUTE" validate:"gt=0" default:"60"`
	RequestsPerHour   int `mapstructure:"requests_per_hour" yaml:"requests_per_hour" env:"CHATMODEL_RATE_LIMIT_REQUESTS_PER_HOUR" validate:"gt=0" default:"1000"`
	BurstSize         int `mapstructure:"burst_size" yaml:"burst_size" env:"CHATMODEL_RATE_LIMIT_BURST_SIZE" validate:"gt=0" default:"10"`
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
		Providers:               make(map[string]any),
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
