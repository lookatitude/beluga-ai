package orchestration

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/lookatitude/beluga-ai/pkg/orchestration/iface"
)

// Config holds the main configuration for the orchestration package
type Config struct {
	// Chain configuration
	Chain ChainConfig `mapstructure:"chain" yaml:"chain"`

	// Graph configuration
	Graph GraphConfig `mapstructure:"graph" yaml:"graph"`

	// Workflow configuration
	Workflow WorkflowConfig `mapstructure:"workflow" yaml:"workflow"`

	// Observability configuration
	Observability ObservabilityConfig `mapstructure:"observability" yaml:"observability"`

	// Enabled features
	Enabled EnabledFeatures `mapstructure:"enabled" yaml:"enabled"`
}

// ChainConfig holds configuration specific to chain orchestration
type ChainConfig struct {
	DefaultTimeout      time.Duration `mapstructure:"default_timeout" yaml:"default_timeout" validate:"min=1ns,max=24h" default:"5m"`
	DefaultRetries      int           `mapstructure:"default_retries" yaml:"default_retries" validate:"min=0,max=20" default:"3"`
	MaxConcurrentChains int           `mapstructure:"max_concurrent_chains" yaml:"max_concurrent_chains" validate:"min=1,max=10000" default:"10"`
	EnableMemoryPooling bool          `mapstructure:"enable_memory_pooling" yaml:"enable_memory_pooling" default:"true"`
}

// GraphConfig holds configuration specific to graph orchestration
type GraphConfig struct {
	DefaultTimeout          time.Duration `mapstructure:"default_timeout" yaml:"default_timeout" validate:"min=1ns,max=24h" default:"10m"`
	DefaultRetries          int           `mapstructure:"default_retries" yaml:"default_retries" validate:"min=0,max=20" default:"3"`
	MaxWorkers              int           `mapstructure:"max_workers" yaml:"max_workers" validate:"min=1,max=1000" default:"5"`
	EnableParallelExecution bool          `mapstructure:"enable_parallel_execution" yaml:"enable_parallel_execution" default:"true"`
	QueueSize               int           `mapstructure:"queue_size" yaml:"queue_size" validate:"min=1,max=100000" default:"100"`
}

// ObservabilityConfig holds observability-related configuration
type ObservabilityConfig struct {
	EnableMetrics       bool          `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
	EnableTracing       bool          `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
	MetricsPrefix       string        `mapstructure:"metrics_prefix" yaml:"metrics_prefix" default:"beluga_orchestration"`
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval" yaml:"health_check_interval" default:"30s"`
}

// WorkflowConfig holds configuration specific to workflow orchestration
type WorkflowConfig struct {
	DefaultTimeout         time.Duration `mapstructure:"default_timeout" yaml:"default_timeout" validate:"min=1ns,max=24h" default:"30m"`
	DefaultRetries         int           `mapstructure:"default_retries" yaml:"default_retries" validate:"min=0,max=20" default:"5"`
	TaskQueue              string        `mapstructure:"task_queue" yaml:"task_queue" validate:"required,min=1,max=255" default:"beluga-workflows"`
	EnablePersistence      bool          `mapstructure:"enable_persistence" yaml:"enable_persistence" default:"false"`
	MaxConcurrentWorkflows int           `mapstructure:"max_concurrent_workflows" yaml:"max_concurrent_workflows" validate:"min=1,max=1000" default:"50"`
}

// EnabledFeatures holds configuration for enabling/disabling features
type EnabledFeatures struct {
	Chains     bool `mapstructure:"chains" yaml:"chains" default:"true"`
	Graphs     bool `mapstructure:"graphs" yaml:"graphs" default:"true"`
	Workflows  bool `mapstructure:"workflows" yaml:"workflows" default:"false"`
	Scheduler  bool `mapstructure:"scheduler" yaml:"scheduler" default:"true"`
	MessageBus bool `mapstructure:"message_bus" yaml:"message_bus" default:"true"`
}

// Validate validates the configuration
func (c *Config) Validate() error {
	validate := validator.New()

	if err := validate.Struct(c); err != nil {
		return iface.NewOrchestratorError("config_validation", err, "invalid_config")
	}

	// Custom validations
	if c.Chain.MaxConcurrentChains < 1 {
		return iface.NewOrchestratorError("config_validation", fmt.Errorf("max_concurrent_chains must be >= 1"), "invalid_config")
	}

	if c.Graph.MaxWorkers < 1 {
		return iface.NewOrchestratorError("config_validation", fmt.Errorf("max_workers must be >= 1"), "invalid_config")
	}

	if c.Workflow.MaxConcurrentWorkflows < 1 {
		return iface.NewOrchestratorError("config_validation", fmt.Errorf("max_concurrent_workflows must be >= 1"), "invalid_config")
	}

	return nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Chain: ChainConfig{
			DefaultTimeout:      5 * time.Minute,
			DefaultRetries:      3,
			MaxConcurrentChains: 10,
			EnableMemoryPooling: true,
		},
		Graph: GraphConfig{
			DefaultTimeout:          10 * time.Minute,
			DefaultRetries:          3,
			MaxWorkers:              5,
			EnableParallelExecution: true,
			QueueSize:               100,
		},
		Workflow: WorkflowConfig{
			DefaultTimeout:         30 * time.Minute,
			DefaultRetries:         5,
			TaskQueue:              "beluga-workflows",
			EnablePersistence:      false,
			MaxConcurrentWorkflows: 50,
		},
		Observability: ObservabilityConfig{
			EnableMetrics:       true,
			EnableTracing:       true,
			MetricsPrefix:       "beluga_orchestration",
			HealthCheckInterval: 30 * time.Second,
		},
		Enabled: EnabledFeatures{
			Chains:     true,
			Graphs:     true,
			Workflows:  false,
			Scheduler:  true,
			MessageBus: true,
		},
	}
}

// Option represents a functional option for configuring the orchestrator
type Option func(*Config) error

// WithChainTimeout sets the default chain timeout
func WithChainTimeout(timeout time.Duration) Option {
	return func(c *Config) error {
		if timeout <= 0 {
			return fmt.Errorf("timeout must be positive")
		}
		c.Chain.DefaultTimeout = timeout
		return nil
	}
}

// WithGraphMaxWorkers sets the maximum number of workers for graph execution
func WithGraphMaxWorkers(workers int) Option {
	return func(c *Config) error {
		if workers < 1 {
			return fmt.Errorf("max workers must be >= 1")
		}
		c.Graph.MaxWorkers = workers
		return nil
	}
}

// WithWorkflowTaskQueue sets the workflow task queue
func WithWorkflowTaskQueue(queue string) Option {
	return func(c *Config) error {
		if queue == "" {
			return fmt.Errorf("task queue cannot be empty")
		}
		c.Workflow.TaskQueue = queue
		return nil
	}
}

// WithMetricsPrefix sets the metrics prefix
func WithMetricsPrefix(prefix string) Option {
	return func(c *Config) error {
		if prefix == "" {
			return fmt.Errorf("metrics prefix cannot be empty")
		}
		c.Observability.MetricsPrefix = prefix
		return nil
	}
}

// WithFeatures enables/disables specific features
func WithFeatures(features EnabledFeatures) Option {
	return func(c *Config) error {
		c.Enabled = features
		return nil
	}
}

// NewConfig creates a new configuration with the given options
func NewConfig(opts ...Option) (*Config, error) {
	config := DefaultConfig()

	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, iface.NewOrchestratorError("config_creation", err, "invalid_config")
		}
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}
