// Package core provides configuration structures for core package components.
package core

// Config represents configuration for core package components.
// Since core uses functional options for most configuration,
// this struct provides a minimal configuration structure for cases
// where a config struct is needed.
type Config struct {
	// LogLevel sets the logging level (debug, info, warn, error).
	LogLevel string `mapstructure:"log_level" yaml:"logLevel" env:"CORE_LOG_LEVEL" validate:"oneof=debug info warn error"`

	// EnableTracing enables OpenTelemetry tracing for core operations.
	EnableTracing bool `mapstructure:"enable_tracing" yaml:"enableTracing" env:"CORE_ENABLE_TRACING"`

	// EnableMetrics enables OpenTelemetry metrics for core operations.
	EnableMetrics bool `mapstructure:"enable_metrics" yaml:"enableMetrics" env:"CORE_ENABLE_METRICS"`
}

// DefaultConfig returns a default configuration for core package.
func DefaultConfig() *Config {
	return &Config{
		EnableTracing: true,
		EnableMetrics: true,
		LogLevel:      "info",
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.LogLevel != "" {
		validLevels := map[string]bool{
			"debug": true,
			"info":  true,
			"warn":  true,
			"error": true,
		}
		if !validLevels[c.LogLevel] {
			return NewValidationError("invalid log level", nil)
		}
	}
	return nil
}
