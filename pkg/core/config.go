// Package core provides configuration management for the core framework components.
// T016: Create config.go with CoreConfig struct and functional options pattern
package core

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

// CoreConfig defines configuration options for core package components.
// It follows the constitutional requirement for configuration management.
type CoreConfig struct {
	// Container Configuration
	EnableDependencyInjection bool          `yaml:"enable_dependency_injection" json:"enable_dependency_injection"`
	MaxRegistrations          int           `yaml:"max_registrations" json:"max_registrations" validate:"min=1"`
	ResolutionTimeout         time.Duration `yaml:"resolution_timeout" json:"resolution_timeout" validate:"min=1ms"`

	// Performance Configuration
	EnablePerformanceMonitoring bool              `yaml:"enable_performance_monitoring" json:"enable_performance_monitoring"`
	PerformanceTargets          PerformanceConfig `yaml:"performance_targets" json:"performance_targets"`

	// Health Monitoring Configuration
	EnableHealthChecking bool          `yaml:"enable_health_checking" json:"enable_health_checking"`
	HealthCheckInterval  time.Duration `yaml:"health_check_interval" json:"health_check_interval" validate:"min=1s"`
	HealthCheckTimeout   time.Duration `yaml:"health_check_timeout" json:"health_check_timeout" validate:"min=100ms"`

	// Observability Configuration
	EnableMetrics  bool   `yaml:"enable_metrics" json:"enable_metrics"`
	EnableTracing  bool   `yaml:"enable_tracing" json:"enable_tracing"`
	EnableLogging  bool   `yaml:"enable_logging" json:"enable_logging"`
	MetricsPrefix  string `yaml:"metrics_prefix" json:"metrics_prefix"`
	TracingService string `yaml:"tracing_service" json:"tracing_service"`

	// Concurrency Configuration
	MaxConcurrentOperations int  `yaml:"max_concurrent_operations" json:"max_concurrent_operations" validate:"min=1"`
	EnableThreadSafety      bool `yaml:"enable_thread_safety" json:"enable_thread_safety"`

	// Development Configuration
	EnableDebugMode      bool `yaml:"enable_debug_mode" json:"enable_debug_mode"`
	EnableVerboseLogging bool `yaml:"enable_verbose_logging" json:"enable_verbose_logging"`
}

// PerformanceConfig defines performance targets and thresholds.
type PerformanceConfig struct {
	// DI Container Performance Targets
	MaxDIResolutionTime   time.Duration `yaml:"max_di_resolution_time" json:"max_di_resolution_time" validate:"min=1ns"`
	MaxDIRegistrationTime time.Duration `yaml:"max_di_registration_time" json:"max_di_registration_time" validate:"min=1ns"`
	MinDIThroughput       int           `yaml:"min_di_throughput" json:"min_di_throughput" validate:"min=1"`

	// Runnable Performance Targets
	MaxRunnableInvokeOverhead time.Duration `yaml:"max_runnable_invoke_overhead" json:"max_runnable_invoke_overhead" validate:"min=1ns"`
	MaxStreamSetupTime        time.Duration `yaml:"max_stream_setup_time" json:"max_stream_setup_time" validate:"min=1ns"`
	MinRunnableThroughput     int           `yaml:"min_runnable_throughput" json:"min_runnable_throughput" validate:"min=1"`

	// Memory Performance Targets
	MaxMemoryOverhead   int64 `yaml:"max_memory_overhead" json:"max_memory_overhead" validate:"min=0"`
	MaxAllocationsPerOp int   `yaml:"max_allocations_per_op" json:"max_allocations_per_op" validate:"min=0"`
}

// Validate validates the CoreConfig struct using validator tags.
func (c *CoreConfig) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

// NewCoreConfig creates a new CoreConfig with validation and default values.
func NewCoreConfig(opts ...CoreOption) (*CoreConfig, error) {
	config := &CoreConfig{
		// Default values
		EnableDependencyInjection: true,
		MaxRegistrations:          1000,
		ResolutionTimeout:         time.Millisecond * 10,

		EnablePerformanceMonitoring: true,
		PerformanceTargets: PerformanceConfig{
			MaxDIResolutionTime:       time.Millisecond,       // 1ms target
			MaxDIRegistrationTime:     time.Millisecond,       // 1ms target
			MinDIThroughput:           10000,                  // 10k ops/sec
			MaxRunnableInvokeOverhead: 100 * time.Microsecond, // 100μs target
			MaxStreamSetupTime:        10 * time.Millisecond,  // 10ms target
			MinRunnableThroughput:     10000,                  // 10k ops/sec
			MaxMemoryOverhead:         1024 * 1024,            // 1MB
			MaxAllocationsPerOp:       10,                     // 10 allocations max
		},

		EnableHealthChecking: true,
		HealthCheckInterval:  time.Minute,
		HealthCheckTimeout:   time.Second * 5,

		EnableMetrics:  true,
		EnableTracing:  true,
		EnableLogging:  true,
		MetricsPrefix:  "beluga_core",
		TracingService: "beluga-ai-core",

		MaxConcurrentOperations: 1000,
		EnableThreadSafety:      true,

		EnableDebugMode:      false,
		EnableVerboseLogging: false,
	}

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, NewValidationError("config_validation_failed", err)
	}

	return config, nil
}

// CoreOption defines a function type for CoreConfig functional options.
type CoreOption func(*CoreConfig)

// WithDependencyInjection enables or disables dependency injection.
func WithDependencyInjection(enabled bool) CoreOption {
	return func(c *CoreConfig) {
		c.EnableDependencyInjection = enabled
	}
}

// WithMaxRegistrations sets the maximum number of registrations allowed.
func WithMaxRegistrations(max int) CoreOption {
	return func(c *CoreConfig) {
		c.MaxRegistrations = max
	}
}

// WithResolutionTimeout sets the timeout for dependency resolution.
func WithResolutionTimeout(timeout time.Duration) CoreOption {
	return func(c *CoreConfig) {
		c.ResolutionTimeout = timeout
	}
}

// WithPerformanceMonitoring enables or disables performance monitoring.
func WithPerformanceMonitoring(enabled bool) CoreOption {
	return func(c *CoreConfig) {
		c.EnablePerformanceMonitoring = enabled
	}
}

// WithPerformanceTargets sets custom performance targets.
func WithPerformanceTargets(targets PerformanceConfig) CoreOption {
	return func(c *CoreConfig) {
		c.PerformanceTargets = targets
	}
}

// WithHealthChecking enables or disables health checking.
func WithHealthChecking(enabled bool) CoreOption {
	return func(c *CoreConfig) {
		c.EnableHealthChecking = enabled
	}
}

// WithHealthCheckInterval sets the interval for periodic health checks.
func WithHealthCheckInterval(interval time.Duration) CoreOption {
	return func(c *CoreConfig) {
		c.HealthCheckInterval = interval
	}
}

// WithObservability configures observability features.
func WithObservability(metrics, tracing, logging bool) CoreOption {
	return func(c *CoreConfig) {
		c.EnableMetrics = metrics
		c.EnableTracing = tracing
		c.EnableLogging = logging
	}
}

// WithMetricsPrefix sets the prefix for metrics.
func WithMetricsPrefix(prefix string) CoreOption {
	return func(c *CoreConfig) {
		c.MetricsPrefix = prefix
	}
}

// WithTracingService sets the service name for tracing.
func WithTracingService(service string) CoreOption {
	return func(c *CoreConfig) {
		c.TracingService = service
	}
}

// WithConcurrency configures concurrency settings.
func WithConcurrency(maxOps int, threadSafe bool) CoreOption {
	return func(c *CoreConfig) {
		c.MaxConcurrentOperations = maxOps
		c.EnableThreadSafety = threadSafe
	}
}

// WithDebugMode enables or disables debug mode.
func WithDebugMode(enabled bool) CoreOption {
	return func(c *CoreConfig) {
		c.EnableDebugMode = enabled
		c.EnableVerboseLogging = enabled // Enable verbose logging with debug
	}
}

// ValidateConfig validates a CoreConfig instance and returns detailed errors.
func ValidateConfig(config *CoreConfig) error {
	if config == nil {
		return NewValidationError("config_nil", fmt.Errorf("CoreConfig cannot be nil"))
	}

	// Custom validation beyond struct tags
	if config.MaxRegistrations <= 0 {
		return NewValidationError("invalid_max_registrations",
			fmt.Errorf("MaxRegistrations must be greater than 0"))
	}

	if config.ResolutionTimeout <= 0 {
		return NewValidationError("invalid_resolution_timeout",
			fmt.Errorf("ResolutionTimeout must be positive"))
	}

	if config.EnableHealthChecking && config.HealthCheckInterval <= 0 {
		return NewValidationError("invalid_health_check_interval",
			fmt.Errorf("HealthCheckInterval must be positive when health checking is enabled"))
	}

	// Validate performance targets
	targets := config.PerformanceTargets
	if targets.MaxDIResolutionTime <= 0 {
		return NewValidationError("invalid_di_resolution_target",
			fmt.Errorf("DI resolution time target must be positive"))
	}

	if targets.MaxRunnableInvokeOverhead <= 0 {
		return NewValidationError("invalid_runnable_overhead_target",
			fmt.Errorf("Runnable invoke overhead target must be positive"))
	}

	return config.Validate()
}

// DefaultConfig returns a default CoreConfig with sensible defaults.
func DefaultConfig() *CoreConfig {
	config, _ := NewCoreConfig() // Error is impossible with no options
	return config
}

// DevelopmentConfig returns a CoreConfig optimized for development.
func DevelopmentConfig() *CoreConfig {
	config, _ := NewCoreConfig(
		WithDebugMode(true),
		WithHealthCheckInterval(time.Second*30),
		WithPerformanceMonitoring(true),
		WithObservability(true, true, true),
	)
	return config
}

// ProductionConfig returns a CoreConfig optimized for production.
func ProductionConfig() *CoreConfig {
	config, _ := NewCoreConfig(
		WithDebugMode(false),
		WithHealthCheckInterval(time.Minute*5),
		WithPerformanceMonitoring(true),
		WithObservability(true, true, false), // No verbose logging in production
		WithMaxRegistrations(10000),
	)
	return config
}
