// Package safety provides configuration for safety validation.
package safety

import (
	"github.com/go-playground/validator/v10"
)

// Config holds configuration for safety validation.
type Config struct {
	// Enabled determines if safety validation is active
	Enabled bool `yaml:"enabled" mapstructure:"enabled" default:"true"`

	// RiskThreshold is the maximum risk score before content is considered unsafe
	// Content with risk_score >= threshold is marked as unsafe
	RiskThreshold float64 `yaml:"risk_threshold" mapstructure:"risk_threshold" default:"0.3" validate:"gte=0,lte=1"`

	// ToxicityWeight is the weight added to risk score for toxicity issues
	ToxicityWeight float64 `yaml:"toxicity_weight" mapstructure:"toxicity_weight" default:"0.4" validate:"gte=0,lte=1"`

	// BiasWeight is the weight added to risk score for bias issues
	BiasWeight float64 `yaml:"bias_weight" mapstructure:"bias_weight" default:"0.2" validate:"gte=0,lte=1"`

	// HarmfulWeight is the weight added to risk score for harmful content issues
	HarmfulWeight float64 `yaml:"harmful_weight" mapstructure:"harmful_weight" default:"0.5" validate:"gte=0,lte=1"`

	// CustomPatterns allows adding additional patterns for detection
	CustomPatterns *CustomPatternsConfig `yaml:"custom_patterns" mapstructure:"custom_patterns,omitempty"`

	// EnableMetrics determines if OTEL metrics are recorded
	EnableMetrics bool `yaml:"enable_metrics" mapstructure:"enable_metrics" default:"true"`
}

// CustomPatternsConfig holds custom regex patterns for safety checking.
type CustomPatternsConfig struct {
	// ToxicityPatterns are additional toxicity detection patterns
	ToxicityPatterns []string `yaml:"toxicity_patterns" mapstructure:"toxicity_patterns"`

	// BiasPatterns are additional bias detection patterns
	BiasPatterns []string `yaml:"bias_patterns" mapstructure:"bias_patterns"`

	// HarmfulPatterns are additional harmful content detection patterns
	HarmfulPatterns []string `yaml:"harmful_patterns" mapstructure:"harmful_patterns"`
}

// DefaultConfig returns the default safety configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled:        true,
		RiskThreshold:  0.3,
		ToxicityWeight: 0.4,
		BiasWeight:     0.2,
		HarmfulWeight:  0.5,
		EnableMetrics:  true,
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	validate := validator.New()
	return validate.Struct(c)
}

// ConfigOption is a functional option for Config.
type ConfigOption func(*Config)

// WithEnabled sets whether safety validation is enabled.
func WithEnabled(enabled bool) ConfigOption {
	return func(c *Config) {
		c.Enabled = enabled
	}
}

// WithRiskThreshold sets the risk threshold.
func WithRiskThreshold(threshold float64) ConfigOption {
	return func(c *Config) {
		c.RiskThreshold = threshold
	}
}

// WithToxicityWeight sets the toxicity weight.
func WithToxicityWeight(weight float64) ConfigOption {
	return func(c *Config) {
		c.ToxicityWeight = weight
	}
}

// WithBiasWeight sets the bias weight.
func WithBiasWeight(weight float64) ConfigOption {
	return func(c *Config) {
		c.BiasWeight = weight
	}
}

// WithHarmfulWeight sets the harmful content weight.
func WithHarmfulWeight(weight float64) ConfigOption {
	return func(c *Config) {
		c.HarmfulWeight = weight
	}
}

// WithEnableMetrics sets whether metrics are enabled.
func WithEnableMetrics(enabled bool) ConfigOption {
	return func(c *Config) {
		c.EnableMetrics = enabled
	}
}

// WithCustomPatterns sets custom detection patterns.
func WithCustomPatterns(patterns *CustomPatternsConfig) ConfigOption {
	return func(c *Config) {
		c.CustomPatterns = patterns
	}
}
