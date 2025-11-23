package turndetection

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// Config represents the configuration for Turn Detection providers.
// It includes common settings that apply to all Turn Detection providers.
type Config struct {
	// Provider specifies the Turn Detection provider (e.g., "onnx", "heuristic", "ml")
	Provider string `mapstructure:"provider" yaml:"provider" validate:"required,oneof=onnx heuristic ml"`

	// MinSilenceDuration specifies the minimum duration of silence to detect a turn end (ms)
	MinSilenceDuration time.Duration `mapstructure:"min_silence_duration" yaml:"min_silence_duration" default:"500ms" validate:"min=100ms,max=5s"`

	// MinTurnLength specifies the minimum length of a turn in characters
	MinTurnLength int `mapstructure:"min_turn_length" yaml:"min_turn_length" default:"10" validate:"min=1,max=1000"`

	// MaxTurnLength specifies the maximum length of a turn in characters
	MaxTurnLength int `mapstructure:"max_turn_length" yaml:"max_turn_length" default:"5000" validate:"min=10,max=50000"`

	// SentenceEndMarkers specifies characters that indicate sentence endings
	SentenceEndMarkers string `mapstructure:"sentence_end_markers" yaml:"sentence_end_markers" default:".!?"`

	// QuestionMarkers specifies phrases that indicate questions
	QuestionMarkers []string `mapstructure:"question_markers" yaml:"question_markers" default:"what,where,when,why,how,who,which"`

	// ModelPath specifies the path to the model file (for ML-based providers)
	ModelPath string `mapstructure:"model_path" yaml:"model_path"`

	// Threshold specifies the turn detection threshold (0.0-1.0, default: 0.5)
	Threshold float64 `mapstructure:"threshold" yaml:"threshold" default:"0.5" validate:"gte=0.0,lte=1.0"`

	// Timeout for processing operations
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" default:"1s" validate:"min=100ms,max=10s"`

	// Provider-specific configuration
	ProviderSpecific map[string]interface{} `mapstructure:"provider_specific" yaml:"provider_specific"`

	// Observability settings
	EnableTracing           bool `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
	EnableMetrics           bool `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
	EnableStructuredLogging bool `mapstructure:"enable_structured_logging" yaml:"enable_structured_logging" default:"true"`
}

// ConfigOption is a functional option for configuring Turn Detection instances
type ConfigOption func(*Config)

// WithProvider sets the Turn Detection provider
func WithProvider(provider string) ConfigOption {
	return func(c *Config) {
		c.Provider = provider
	}
}

// WithMinSilenceDuration sets the minimum silence duration
func WithMinSilenceDuration(duration time.Duration) ConfigOption {
	return func(c *Config) {
		c.MinSilenceDuration = duration
	}
}

// WithMinTurnLength sets the minimum turn length
func WithMinTurnLength(length int) ConfigOption {
	return func(c *Config) {
		c.MinTurnLength = length
	}
}

// WithMaxTurnLength sets the maximum turn length
func WithMaxTurnLength(length int) ConfigOption {
	return func(c *Config) {
		c.MaxTurnLength = length
	}
}

// WithSentenceEndMarkers sets the sentence end markers
func WithSentenceEndMarkers(markers string) ConfigOption {
	return func(c *Config) {
		c.SentenceEndMarkers = markers
	}
}

// WithThreshold sets the turn detection threshold
func WithThreshold(threshold float64) ConfigOption {
	return func(c *Config) {
		c.Threshold = threshold
	}
}

// WithModelPath sets the model path
func WithModelPath(path string) ConfigOption {
	return func(c *Config) {
		c.ModelPath = path
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
		Provider:                "heuristic",
		MinSilenceDuration:      500 * time.Millisecond,
		MinTurnLength:           10,
		MaxTurnLength:           5000,
		SentenceEndMarkers:      ".!?",
		QuestionMarkers:         []string{"what", "where", "when", "why", "how", "who", "which"},
		Threshold:               0.5,
		Timeout:                 1 * time.Second,
		EnableTracing:           true,
		EnableMetrics:           true,
		EnableStructuredLogging: true,
	}
}
