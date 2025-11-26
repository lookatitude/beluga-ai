package prompts

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/prompts/iface"
)

// Config is an alias for iface.Config.
type Config = iface.Config

// Option represents a functional option for configuring prompt components.
type Option = iface.Option

// options is an alias for iface.Options.
type options = iface.Options

// WithConfig sets the configuration.
func WithConfig(config *Config) Option {
	return func(o *iface.Options) {
		o.Config = config
	}
}

// WithValidator sets a custom variable validator.
func WithValidator(validator iface.VariableValidator) Option {
	return func(o *iface.Options) {
		o.Validator = validator
	}
}

// WithTemplateEngine sets a custom template engine.
func WithTemplateEngine(engine iface.TemplateEngine) Option {
	return func(o *iface.Options) {
		o.TemplateEngine = engine
	}
}

// WithMetrics sets the metrics collector.
func WithMetrics(metrics iface.Metrics) Option {
	return func(o *iface.Options) {
		o.Metrics = metrics
	}
}

// WithTracer sets the tracer.
func WithTracer(tracer iface.Tracer) Option {
	return func(o *iface.Options) {
		o.Tracer = tracer
	}
}

// WithLogger sets the logger.
func WithLogger(logger iface.Logger) Option {
	return func(o *iface.Options) {
		o.Logger = logger
	}
}

// WithHealthChecker sets the health checker.
func WithHealthChecker(checker iface.HealthChecker) Option {
	return func(o *iface.Options) {
		o.HealthChecker = checker
	}
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		DefaultTemplateTimeout: 30 * time.Second,
		MaxTemplateSize:        1048576, // 1MB
		ValidateVariables:      true,
		StrictVariableCheck:    false,
		EnableTemplateCache:    true,
		CacheTTL:               5 * time.Minute,
		MaxCacheSize:           100,
		EnableMetrics:          true,
		EnableTracing:          true,
		DefaultAdapterType:     "default",
	}
}
