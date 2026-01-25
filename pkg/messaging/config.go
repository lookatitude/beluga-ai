package messaging

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

// Config represents the configuration for messaging backends.
// It includes common settings that apply to all messaging providers.
type Config struct {
	ProviderSpecific        map[string]any `mapstructure:"provider_specific" yaml:"provider_specific"`
	Provider                string         `mapstructure:"provider" yaml:"provider" validate:"required"`
	Timeout                 time.Duration  `mapstructure:"timeout" yaml:"timeout" default:"30s" validate:"min=1s,max=5m"`
	MaxRetries              int            `mapstructure:"max_retries" yaml:"max_retries" default:"3" validate:"gte=0,lte=10"`
	RetryDelay              time.Duration  `mapstructure:"retry_delay" yaml:"retry_delay" default:"1s" validate:"min=100ms,max=30s"`
	RetryBackoff            float64        `mapstructure:"retry_backoff" yaml:"retry_backoff" default:"2.0" validate:"gte=1,lte=5"`
	EnableTracing           bool           `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
	EnableMetrics           bool           `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
	EnableStructuredLogging bool           `mapstructure:"enable_structured_logging" yaml:"enable_structured_logging" default:"true"`
}

// ConfigOption is a functional option for configuring messaging instances.
type ConfigOption func(*Config)

// WithProvider sets the messaging provider.
func WithProvider(provider string) ConfigOption {
	return func(c *Config) {
		c.Provider = provider
	}
}

// WithTimeout sets the timeout.
func WithTimeout(timeout time.Duration) ConfigOption {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithRetryConfig sets retry configuration.
func WithRetryConfig(maxRetries int, delay time.Duration, backoff float64) ConfigOption {
	return func(c *Config) {
		c.MaxRetries = maxRetries
		c.RetryDelay = delay
		c.RetryBackoff = backoff
	}
}

// WithObservability enables or disables observability features.
func WithObservability(tracing, metrics, logging bool) ConfigOption {
	return func(c *Config) {
		c.EnableTracing = tracing
		c.EnableMetrics = metrics
		c.EnableStructuredLogging = logging
	}
}

// WithProviderSpecific sets provider-specific configuration.
func WithProviderSpecific(key string, value any) ConfigOption {
	return func(c *Config) {
		if c.ProviderSpecific == nil {
			c.ProviderSpecific = make(map[string]any)
		}
		c.ProviderSpecific[key] = value
	}
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		Provider:                "",
		Timeout:                 30 * time.Second,
		MaxRetries:              3,
		RetryDelay:              time.Second,
		RetryBackoff:            2.0,
		EnableTracing:           true,
		EnableMetrics:           true,
		EnableStructuredLogging: true,
		ProviderSpecific:        make(map[string]any),
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("configuration cannot be nil")
	}

	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Additional validation
	if c.Timeout < time.Second {
		return fmt.Errorf("timeout must be at least 1 second, got %v", c.Timeout)
	}

	if c.Timeout > 5*time.Minute {
		return fmt.Errorf("timeout must be at most 5 minutes, got %v", c.Timeout)
	}

	return nil
}

// MergeOptions applies functional options to the configuration.
func (c *Config) MergeOptions(opts ...ConfigOption) {
	for _, opt := range opts {
		opt(c)
	}
}

// NewConfig creates a new configuration with the given options.
func NewConfig(opts ...ConfigOption) *Config {
	config := DefaultConfig()
	config.MergeOptions(opts...)
	return config
}

// ValidateConfig validates a messaging configuration.
func ValidateConfig(ctx context.Context, config *Config) error {
	if config == nil {
		return errors.New("configuration cannot be nil")
	}

	// Validate using the struct validator
	if err := config.Validate(); err != nil {
		return err
	}

	// Provider-specific validation can be added here
	switch config.Provider {
	case "twilio":
		// Twilio-specific validation will be in provider package
	case "mock":
		// Mock provider has minimal requirements
	default:
		if config.Provider == "" {
			return errors.New("provider is required")
		}
	}

	return nil
}
