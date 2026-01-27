package vad

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

// Config represents the configuration for VAD providers.
// It includes common settings that apply to all VAD providers.
type Config struct {
	ProviderSpecific        map[string]any `mapstructure:"provider_specific" yaml:"provider_specific"`
	Provider                string         `mapstructure:"provider" yaml:"provider" validate:"required,oneof=silero energy webrtc rnnoise"`
	ModelPath               string         `mapstructure:"model_path" yaml:"model_path"`
	SampleRate              int            `mapstructure:"sample_rate" yaml:"sample_rate" default:"16000" validate:"oneof=8000 16000 22050 24000 32000 44100 48000"`
	MinSpeechDuration       time.Duration  `mapstructure:"min_speech_duration" yaml:"min_speech_duration" default:"250ms" validate:"min=50ms,max=2s"`
	MaxSilenceDuration      time.Duration  `mapstructure:"max_silence_duration" yaml:"max_silence_duration" default:"500ms" validate:"min=100ms,max=5s"`
	FrameSize               int            `mapstructure:"frame_size" yaml:"frame_size" default:"512" validate:"min=64,max=4096"`
	Timeout                 time.Duration  `mapstructure:"timeout" yaml:"timeout" default:"1s" validate:"min=100ms,max=10s"`
	Threshold               float64        `mapstructure:"threshold" yaml:"threshold" default:"0.5" validate:"gte=0.0,lte=1.0"`
	EnablePreprocessing     bool           `mapstructure:"enable_preprocessing" yaml:"enable_preprocessing" default:"true"`
	EnableTracing           bool           `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
	EnableMetrics           bool           `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
	EnableStructuredLogging bool           `mapstructure:"enable_structured_logging" yaml:"enable_structured_logging" default:"true"`
}

// ConfigOption is a functional option for configuring VAD instances.
type ConfigOption func(*Config)

// WithProvider sets the VAD provider.
func WithProvider(provider string) ConfigOption {
	return func(c *Config) {
		c.Provider = provider
	}
}

// WithThreshold sets the speech detection threshold.
func WithThreshold(threshold float64) ConfigOption {
	return func(c *Config) {
		c.Threshold = threshold
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

// WithMinSpeechDuration sets the minimum speech duration.
func WithMinSpeechDuration(duration time.Duration) ConfigOption {
	return func(c *Config) {
		c.MinSpeechDuration = duration
	}
}

// WithMaxSilenceDuration sets the maximum silence duration.
func WithMaxSilenceDuration(duration time.Duration) ConfigOption {
	return func(c *Config) {
		c.MaxSilenceDuration = duration
	}
}

// WithEnablePreprocessing sets preprocessing enablement.
func WithEnablePreprocessing(enable bool) ConfigOption {
	return func(c *Config) {
		c.EnablePreprocessing = enable
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
		return fmt.Errorf("VAD config validation failed: %w", err)
	}
	return nil
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		Provider:                "silero",
		Threshold:               0.5,
		FrameSize:               512,
		SampleRate:              16000,
		MinSpeechDuration:       250 * time.Millisecond,
		MaxSilenceDuration:      500 * time.Millisecond,
		EnablePreprocessing:     true,
		Timeout:                 1 * time.Second,
		EnableTracing:           true,
		EnableMetrics:           true,
		EnableStructuredLogging: true,
	}
}
