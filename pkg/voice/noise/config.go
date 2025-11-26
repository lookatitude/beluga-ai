package noise

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

// Config represents the configuration for Noise Cancellation providers.
// It includes common settings that apply to all Noise Cancellation providers.
type Config struct {
	ProviderSpecific         map[string]any `mapstructure:"provider_specific" yaml:"provider_specific"`
	Provider                 string         `mapstructure:"provider" yaml:"provider" validate:"required,oneof=rnnoise webrtc spectral"`
	ModelPath                string         `mapstructure:"model_path" yaml:"model_path"`
	NoiseReductionLevel      float64        `mapstructure:"noise_reduction_level" yaml:"noise_reduction_level" default:"0.5" validate:"gte=0.0,lte=1.0"`
	SampleRate               int            `mapstructure:"sample_rate" yaml:"sample_rate" default:"16000" validate:"oneof=8000 16000 22050 24000 32000 44100 48000"`
	FrameSize                int            `mapstructure:"frame_size" yaml:"frame_size" default:"480" validate:"min=64,max=4096"`
	Timeout                  time.Duration  `mapstructure:"timeout" yaml:"timeout" default:"1s" validate:"min=100ms,max=10s"`
	EnableAdaptiveProcessing bool           `mapstructure:"enable_adaptive_processing" yaml:"enable_adaptive_processing" default:"true"`
	EnableTracing            bool           `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
	EnableMetrics            bool           `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
	EnableStructuredLogging  bool           `mapstructure:"enable_structured_logging" yaml:"enable_structured_logging" default:"true"`
}

// ConfigOption is a functional option for configuring Noise Cancellation instances.
type ConfigOption func(*Config)

// WithProvider sets the Noise Cancellation provider.
func WithProvider(provider string) ConfigOption {
	return func(c *Config) {
		c.Provider = provider
	}
}

// WithNoiseReductionLevel sets the noise reduction level.
func WithNoiseReductionLevel(level float64) ConfigOption {
	return func(c *Config) {
		c.NoiseReductionLevel = level
	}
}

// WithFrameSize sets the frame size.
func WithFrameSize(frameSize int) ConfigOption {
	return func(c *Config) {
		c.FrameSize = frameSize
	}
}

// WithSampleRate sets the sample rate.
func WithSampleRate(sampleRate int) ConfigOption {
	return func(c *Config) {
		c.SampleRate = sampleRate
	}
}

// WithEnableAdaptiveProcessing sets adaptive processing enablement.
func WithEnableAdaptiveProcessing(enable bool) ConfigOption {
	return func(c *Config) {
		c.EnableAdaptiveProcessing = enable
	}
}

// WithModelPath sets the model path.
func WithModelPath(path string) ConfigOption {
	return func(c *Config) {
		c.ModelPath = path
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("noise config validation failed: %w", err)
	}
	return nil
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		Provider:                 "rnnoise",
		NoiseReductionLevel:      0.5,
		SampleRate:               16000,
		FrameSize:                480,
		EnableAdaptiveProcessing: true,
		Timeout:                  1 * time.Second,
		EnableTracing:            true,
		EnableMetrics:            true,
		EnableStructuredLogging:  true,
	}
}
