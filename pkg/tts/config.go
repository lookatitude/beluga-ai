package tts

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

// Config represents the configuration for TTS providers.
// It includes common settings that apply to all TTS providers.
type Config struct {
	ProviderSpecific        map[string]any `mapstructure:"provider_specific" yaml:"provider_specific"`
	Provider                string         `mapstructure:"provider" yaml:"provider" validate:"required,oneof=openai google azure elevenlabs"`
	APIKey                  string         `mapstructure:"api_key" yaml:"api_key" validate:"required_unless=Provider mock"`
	BaseURL                 string         `mapstructure:"base_url" yaml:"base_url"`
	Model                   string         `mapstructure:"model" yaml:"model"`
	Voice                   string         `mapstructure:"voice" yaml:"voice"`
	Language                string         `mapstructure:"language" yaml:"language" validate:"omitempty,len=2"`
	SampleRate              int            `mapstructure:"sample_rate" yaml:"sample_rate" default:"24000" validate:"oneof=8000 16000 22050 24000 44100 48000"`
	MaxRetries              int            `mapstructure:"max_retries" yaml:"max_retries" default:"3" validate:"gte=0,lte=10"`
	Timeout                 time.Duration  `mapstructure:"timeout" yaml:"timeout" default:"30s" validate:"min=1s,max=5m"`
	Pitch                   float64        `mapstructure:"pitch" yaml:"pitch" default:"0.0" validate:"gte=-20.0,lte=20.0"`
	BitDepth                int            `mapstructure:"bit_depth" yaml:"bit_depth" default:"16" validate:"oneof=16 24 32"`
	Speed                   float64        `mapstructure:"speed" yaml:"speed" default:"1.0" validate:"gte=0.25,lte=4.0"`
	RetryBackoff            float64        `mapstructure:"retry_backoff" yaml:"retry_backoff" default:"2.0" validate:"gte=1,lte=5"`
	Volume                  float64        `mapstructure:"volume" yaml:"volume" default:"1.0" validate:"gte=0.0,lte=1.0"`
	RetryDelay              time.Duration  `mapstructure:"retry_delay" yaml:"retry_delay" default:"1s" validate:"min=100ms,max=30s"`
	EnableSSML              bool           `mapstructure:"enable_ssml" yaml:"enable_ssml" default:"false"`
	EnableStreaming         bool           `mapstructure:"enable_streaming" yaml:"enable_streaming" default:"true"`
	EnableTracing           bool           `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
	EnableMetrics           bool           `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
	EnableStructuredLogging bool           `mapstructure:"enable_structured_logging" yaml:"enable_structured_logging" default:"true"`
}

// ConfigOption is a functional option for configuring TTS instances.
type ConfigOption func(*Config)

// WithProvider sets the TTS provider.
func WithProvider(provider string) ConfigOption {
	return func(c *Config) {
		c.Provider = provider
	}
}

// WithAPIKey sets the API key.
func WithAPIKey(apiKey string) ConfigOption {
	return func(c *Config) {
		c.APIKey = apiKey
	}
}

// WithBaseURL sets the base URL.
func WithBaseURL(baseURL string) ConfigOption {
	return func(c *Config) {
		c.BaseURL = baseURL
	}
}

// WithModel sets the model.
func WithModel(model string) ConfigOption {
	return func(c *Config) {
		c.Model = model
	}
}

// WithVoice sets the voice.
func WithVoice(voice string) ConfigOption {
	return func(c *Config) {
		c.Voice = voice
	}
}

// WithLanguage sets the language.
func WithLanguage(language string) ConfigOption {
	return func(c *Config) {
		c.Language = language
	}
}

// WithSpeed sets the speech speed.
func WithSpeed(speed float64) ConfigOption {
	return func(c *Config) {
		c.Speed = speed
	}
}

// WithPitch sets the pitch.
func WithPitch(pitch float64) ConfigOption {
	return func(c *Config) {
		c.Pitch = pitch
	}
}

// WithVolume sets the volume.
func WithVolume(volume float64) ConfigOption {
	return func(c *Config) {
		c.Volume = volume
	}
}

// WithTimeout sets the timeout.
func WithTimeout(timeout time.Duration) ConfigOption {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithSampleRate sets the sample rate.
func WithSampleRate(sampleRate int) ConfigOption {
	return func(c *Config) {
		c.SampleRate = sampleRate
	}
}

// WithEnableStreaming sets streaming enablement.
func WithEnableStreaming(enable bool) ConfigOption {
	return func(c *Config) {
		c.EnableStreaming = enable
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	validate := validator.New()
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("TTS config validation failed: %w", err)
	}
	return nil
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		Provider:                "openai",
		Timeout:                 30 * time.Second,
		SampleRate:              24000,
		BitDepth:                16,
		Speed:                   1.0,
		Pitch:                   0.0,
		Volume:                  1.0,
		EnableStreaming:         true,
		EnableSSML:              false,
		MaxRetries:              3,
		RetryDelay:              1 * time.Second,
		RetryBackoff:            2.0,
		EnableTracing:           true,
		EnableMetrics:           true,
		EnableStructuredLogging: true,
	}
}
