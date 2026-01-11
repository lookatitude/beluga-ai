// Package voice provides configuration structures for the voice package.
// This file aggregates common configuration patterns used across voice sub-packages.
package voice

// Config represents top-level configuration for voice package components.
// Individual sub-packages (stt, tts, s2s, etc.) have their own specific configs.
type Config struct {
	// EnableTracing enables OpenTelemetry tracing for voice operations.
	EnableTracing bool `mapstructure:"enable_tracing" yaml:"enable_tracing" env:"VOICE_ENABLE_TRACING"`

	// EnableMetrics enables OpenTelemetry metrics for voice operations.
	EnableMetrics bool `mapstructure:"enable_metrics" yaml:"enable_metrics" env:"VOICE_ENABLE_METRICS"`

	// LogLevel sets the logging level (debug, info, warn, error).
	LogLevel string `mapstructure:"log_level" yaml:"log_level" env:"VOICE_LOG_LEVEL" validate:"oneof=debug info warn error"`

	// DefaultSampleRate sets the default audio sample rate in Hz.
	DefaultSampleRate int `mapstructure:"default_sample_rate" yaml:"default_sample_rate" env:"VOICE_DEFAULT_SAMPLE_RATE"`

	// DefaultChannels sets the default number of audio channels.
	DefaultChannels int `mapstructure:"default_channels" yaml:"default_channels" env:"VOICE_DEFAULT_CHANNELS"`
}

// DefaultConfig returns a default configuration for voice package.
func DefaultConfig() *Config {
	return &Config{
		EnableTracing:     true,
		EnableMetrics:     true,
		LogLevel:          "info",
		DefaultSampleRate: 16000,
		DefaultChannels:   1,
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
			return NewVoiceError("Validate", ErrCodeInvalidConfig, nil)
		}
	}
	if c.DefaultSampleRate <= 0 {
		return NewVoiceErrorWithMessage("Validate", ErrCodeInvalidConfig, "default_sample_rate must be positive", nil)
	}
	if c.DefaultChannels <= 0 {
		return NewVoiceErrorWithMessage("Validate", ErrCodeInvalidConfig, "default_channels must be positive", nil)
	}
	return nil
}
