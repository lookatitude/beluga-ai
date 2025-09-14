package agents

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Config represents the configuration for the agents package.
// It includes settings for agent behavior, execution, and monitoring.
type Config struct {
	// Default settings for all agents
	DefaultMaxRetries    int           `mapstructure:"default_max_retries" yaml:"default_max_retries" default:"3"`
	DefaultRetryDelay    time.Duration `mapstructure:"default_retry_delay" yaml:"default_retry_delay" default:"2s"`
	DefaultTimeout       time.Duration `mapstructure:"default_timeout" yaml:"default_timeout" default:"30s"`
	DefaultMaxIterations int           `mapstructure:"default_max_iterations" yaml:"default_max_iterations" default:"15"`

	// Monitoring and observability settings
	EnableMetrics      bool   `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
	EnableTracing      bool   `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
	MetricsPrefix      string `mapstructure:"metrics_prefix" yaml:"metrics_prefix" default:"beluga_agents"`
	TracingServiceName string `mapstructure:"tracing_service_name" yaml:"tracing_service_name" default:"beluga-agents"`

	// Executor settings
	ExecutorConfig ExecutorConfig `mapstructure:"executor" yaml:"executor"`

	// Agent-specific configurations
	AgentConfigs map[string]schema.AgentConfig `mapstructure:"agents" yaml:"agents"`
}

// ExecutorConfig defines configuration for agent execution.
type ExecutorConfig struct {
	// Default execution settings
	DefaultMaxConcurrency int `mapstructure:"default_max_concurrency" yaml:"default_max_concurrency" default:"10"`

	// Error handling
	HandleParsingErrors     bool `mapstructure:"handle_parsing_errors" yaml:"handle_parsing_errors" default:"true"`
	ReturnIntermediateSteps bool `mapstructure:"return_intermediate_steps" yaml:"return_intermediate_steps" default:"false"`

	// Resource limits
	MaxConcurrentExecutions int           `mapstructure:"max_concurrent_executions" yaml:"max_concurrent_executions" default:"100"`
	ExecutionTimeout        time.Duration `mapstructure:"execution_timeout" yaml:"execution_timeout" default:"5m"`
}

// Option represents a functional option for configuring agents.
type Option func(*options)

// options holds the configuration options for an agent.
type options struct {
	maxRetries    int
	retryDelay    time.Duration
	timeout       time.Duration
	maxIterations int
	enableMetrics bool
	enableTracing bool
	eventHandlers map[string][]func(interface{}) error
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
		// TODO: Add timeout support to iface.Options
	}
}

// WithMaxIterations sets the maximum number of iterations for agent planning.
func WithMaxIterations(iterations int) iface.Option {
	return func(o *iface.Options) {
		// TODO: Add max iterations support to iface.Options
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
