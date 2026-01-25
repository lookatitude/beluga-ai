package agents

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Config represents the configuration for the agents package.
// It includes settings for agent behavior, execution, and monitoring.
type Config struct {
	AgentConfigs         map[string]schema.AgentConfig `mapstructure:"agents" yaml:"agents"`
	MetricsPrefix        string                        `mapstructure:"metrics_prefix" yaml:"metrics_prefix" validate:"required" default:"beluga_agents"`
	TracingServiceName   string                        `mapstructure:"tracing_service_name" yaml:"tracing_service_name" validate:"required" default:"beluga-agents"`
	ExecutorConfig       ExecutorConfig                `mapstructure:"executor" yaml:"executor"`
	DefaultMaxRetries    int                           `mapstructure:"default_max_retries" yaml:"default_max_retries" validate:"min=0" default:"3"`
	DefaultRetryDelay    time.Duration                 `mapstructure:"default_retry_delay" yaml:"default_retry_delay" validate:"min=0" default:"2s"`
	DefaultTimeout       time.Duration                 `mapstructure:"default_timeout" yaml:"default_timeout" validate:"gt=0" default:"30s"`
	DefaultMaxIterations int                           `mapstructure:"default_max_iterations" yaml:"default_max_iterations" validate:"gt=0" default:"15"`
	EnableMetrics        bool                          `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
	EnableTracing        bool                          `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
}

// ExecutorConfig defines configuration for agent execution.
type ExecutorConfig struct {
	// Default execution settings
	DefaultMaxConcurrency int `mapstructure:"default_max_concurrency" yaml:"default_max_concurrency" validate:"gt=0" default:"10"`

	// Error handling
	HandleParsingErrors     bool `mapstructure:"handle_parsing_errors" yaml:"handle_parsing_errors" default:"true"`
	ReturnIntermediateSteps bool `mapstructure:"return_intermediate_steps" yaml:"return_intermediate_steps" default:"false"`

	// Resource limits
	MaxConcurrentExecutions int           `mapstructure:"max_concurrent_executions" yaml:"max_concurrent_executions" validate:"gt=0" default:"100"`
	ExecutionTimeout        time.Duration `mapstructure:"execution_timeout" yaml:"execution_timeout" validate:"gt=0" default:"5m"`
}

// Option represents a functional option for configuring agents.
type Option func(*options)

// options holds the configuration options for an agent.
type options struct {
	eventHandlers map[string][]func(any) error
	maxRetries    int
	retryDelay    time.Duration
	timeout       time.Duration
	maxIterations int
	enableMetrics bool
	enableTracing bool
}

// WithMaxRetries sets the maximum number of retries for agent operations.
func WithMaxRetries(retries int) iface.Option {
	return func(o *iface.Options) {
		o.MaxRetries = retries
	}
}

// WithRetryDelay sets the delay between retries.
func WithRetryDelay(delay time.Duration) iface.Option {
	return func(o *iface.Options) {
		o.RetryDelay = delay
	}
}

// WithTimeout sets the timeout for agent operations.
func WithTimeout(timeout time.Duration) iface.Option {
	return func(o *iface.Options) {
		o.Timeout = timeout
	}
}

// WithMaxIterations sets the maximum number of iterations for agent planning.
func WithMaxIterations(iterations int) iface.Option {
	return func(o *iface.Options) {
		o.MaxIterations = iterations
	}
}

// WithMetrics enables or disables metrics collection.
func WithMetrics(enabled bool) iface.Option {
	return func(o *iface.Options) {
		o.EnableMetrics = enabled
	}
}

// WithTracing enables or disables tracing.
func WithTracing(enabled bool) iface.Option {
	return func(o *iface.Options) {
		o.EnableTracing = enabled
	}
}

// WithEnableSafety enables or disables safety middleware for the agent.
func WithEnableSafety(enabled bool) iface.Option {
	return func(o *iface.Options) {
		o.EnableSafety = enabled
	}
}

// WithEventHandler registers an event handler for a specific event type.
func WithEventHandler(eventType string, handler iface.EventHandler) iface.Option {
	return func(o *iface.Options) {
		if o.EventHandlers == nil {
			o.EventHandlers = make(map[string][]iface.EventHandler)
		}
		o.EventHandlers[eventType] = append(o.EventHandlers[eventType], handler)
	}
}

// DefaultConfig returns a default configuration for the agents package.
func DefaultConfig() *Config {
	return &Config{
		DefaultMaxRetries:    3,
		DefaultRetryDelay:    2 * time.Second,
		DefaultTimeout:       30 * time.Second,
		DefaultMaxIterations: 15,
		EnableMetrics:        true,
		EnableTracing:        true,
		MetricsPrefix:        "beluga_agents",
		TracingServiceName:   "beluga-agents",
		ExecutorConfig: ExecutorConfig{
			DefaultMaxConcurrency:   10,
			HandleParsingErrors:     true,
			ReturnIntermediateSteps: false,
			MaxConcurrentExecutions: 100,
			ExecutionTimeout:        5 * time.Minute,
		},
		AgentConfigs: make(map[string]schema.AgentConfig),
	}
}

// Validate validates the configuration and returns an error if invalid.
func (c *Config) Validate() error {
	if c.DefaultMaxRetries < 0 {
		return NewValidationError("default_max_retries", "cannot be negative")
	}
	if c.DefaultRetryDelay < 0 {
		return NewValidationError("default_retry_delay", "cannot be negative")
	}
	if c.DefaultTimeout <= 0 {
		return NewValidationError("default_timeout", "must be positive")
	}
	if c.DefaultMaxIterations <= 0 {
		return NewValidationError("default_max_iterations", "must be positive")
	}
	if c.ExecutorConfig.DefaultMaxConcurrency <= 0 {
		return NewValidationError("executor.default_max_concurrency", "must be positive")
	}
	if c.ExecutorConfig.MaxConcurrentExecutions <= 0 {
		return NewValidationError("executor.max_concurrent_executions", "must be positive")
	}
	if c.ExecutorConfig.ExecutionTimeout <= 0 {
		return NewValidationError("executor.execution_timeout", "must be positive")
	}

	return nil
}

// WithStreaming enables streaming mode for agents.
// This is a convenience function that sets EnableStreaming to true in StreamingConfig.
func WithStreaming(enabled bool) iface.Option {
	return func(o *iface.Options) {
		o.StreamingConfig.EnableStreaming = enabled
		// Set defaults if enabling streaming
		if enabled && o.StreamingConfig.ChunkBufferSize == 0 {
			o.StreamingConfig.ChunkBufferSize = 20
		}
		if enabled && o.StreamingConfig.MaxStreamDuration == 0 {
			o.StreamingConfig.MaxStreamDuration = 30 * time.Minute
		}
	}
}

// WithStreamingConfig sets the complete streaming configuration.
func WithStreamingConfig(config iface.StreamingConfig) iface.Option {
	return func(o *iface.Options) {
		o.StreamingConfig = config
	}
}

// ValidateStreamingConfig validates streaming configuration values.
func ValidateStreamingConfig(config iface.StreamingConfig) error {
	if config.ChunkBufferSize <= 0 {
		return NewValidationError("streaming.chunk_buffer_size", "must be greater than 0")
	}
	if config.ChunkBufferSize > 100 {
		return NewValidationError("streaming.chunk_buffer_size", "must be less than or equal to 100")
	}
	if config.MaxStreamDuration <= 0 {
		return NewValidationError("streaming.max_stream_duration", "must be greater than 0")
	}
	return nil
}
