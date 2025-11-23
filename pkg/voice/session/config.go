package session

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// Config represents the configuration for Session providers.
// It includes common settings that apply to all Session providers.
type Config struct {
	// SessionID specifies a custom session ID (auto-generated if empty)
	SessionID string `mapstructure:"session_id" yaml:"session_id"`

	// Timeout specifies the session timeout duration
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30m" validate:"min=1m,max=24h"`

	// AutoStart specifies whether to automatically start the session
	AutoStart bool `mapstructure:"auto_start" yaml:"auto_start" default:"false"`

	// EnableKeepAlive enables keep-alive mechanism
	EnableKeepAlive bool `mapstructure:"enable_keep_alive" yaml:"enable_keep_alive" default:"true"`

	// KeepAliveInterval specifies the keep-alive interval
	KeepAliveInterval time.Duration `mapstructure:"keep_alive_interval" yaml:"keep_alive_interval" default:"30s" validate:"min=5s,max=5m"`

	// MaxRetries specifies the maximum number of retry attempts
	MaxRetries int `mapstructure:"max_retries" yaml:"max_retries" default:"3" validate:"min=0,max=10"`

	// RetryDelay specifies the delay between retry attempts
	RetryDelay time.Duration `mapstructure:"retry_delay" yaml:"retry_delay" default:"1s" validate:"min=100ms,max=10s"`

	// Observability settings
	EnableTracing           bool `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
	EnableMetrics           bool `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
	EnableStructuredLogging bool `mapstructure:"enable_structured_logging" yaml:"enable_structured_logging" default:"true"`
}

// ConfigOption is a functional option for configuring Session instances
type ConfigOption func(*Config)

// WithSessionID sets the session ID
func WithSessionID(id string) ConfigOption {
	return func(c *Config) {
		c.SessionID = id
	}
}

// WithTimeout sets the session timeout
func WithTimeout(timeout time.Duration) ConfigOption {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithAutoStart sets auto-start enablement
func WithAutoStart(autoStart bool) ConfigOption {
	return func(c *Config) {
		c.AutoStart = autoStart
	}
}

// WithEnableKeepAlive sets keep-alive enablement
func WithEnableKeepAlive(enable bool) ConfigOption {
	return func(c *Config) {
		c.EnableKeepAlive = enable
	}
}

// WithKeepAliveInterval sets the keep-alive interval
func WithKeepAliveInterval(interval time.Duration) ConfigOption {
	return func(c *Config) {
		c.KeepAliveInterval = interval
	}
}

// WithMaxRetries sets the maximum number of retries
func WithMaxRetries(maxRetries int) ConfigOption {
	return func(c *Config) {
		c.MaxRetries = maxRetries
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
		SessionID:               "",
		Timeout:                 30 * time.Minute,
		AutoStart:               false,
		EnableKeepAlive:         true,
		KeepAliveInterval:       30 * time.Second,
		MaxRetries:              3,
		RetryDelay:              1 * time.Second,
		EnableTracing:           true,
		EnableMetrics:           true,
		EnableStructuredLogging: true,
	}
}
