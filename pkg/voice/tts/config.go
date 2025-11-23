package tts

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// Config represents the configuration for TTS providers.
// It includes common settings that apply to all TTS providers.
type Config struct {
	// Provider specifies the TTS provider (e.g., "openai", "google", "azure", "elevenlabs")
	Provider string `mapstructure:"provider" yaml:"provider" validate:"required,oneof=openai google azure elevenlabs"`

	// APIKey for authentication (required for most providers)
	APIKey string `mapstructure:"api_key" yaml:"api_key" validate:"required_unless=Provider mock"`

	// BaseURL for custom API endpoints (optional)
	BaseURL string `mapstructure:"base_url" yaml:"base_url"`

	// Model specifies the model to use (provider-specific)
	Model string `mapstructure:"model" yaml:"model"`

	// Voice specifies the voice to use (provider-specific)
	Voice string `mapstructure:"voice" yaml:"voice"`

	// Language specifies the language code (ISO 639-1, e.g., "en", "es")
	Language string `mapstructure:"language" yaml:"language" validate:"omitempty,len=2"`

	// Speed specifies the speech speed (0.25-4.0, default: 1.0)
	Speed float64 `mapstructure:"speed" yaml:"speed" default:"1.0" validate:"gte=0.25,lte=4.0"`

	// Pitch specifies the pitch adjustment (-20.0 to 20.0 semitones, default: 0.0)
	Pitch float64 `mapstructure:"pitch" yaml:"pitch" default:"0.0" validate:"gte=-20.0,lte=20.0"`

	// Volume specifies the volume (0.0-1.0, default: 1.0)
	Volume float64 `mapstructure:"volume" yaml:"volume" default:"1.0" validate:"gte=0.0,lte=1.0"`

	// Timeout for API calls
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s" validate:"min=1s,max=5m"`

	// SampleRate specifies the audio sample rate in Hz
	SampleRate int `mapstructure:"sample_rate" yaml:"sample_rate" default:"24000" validate:"oneof=8000 16000 22050 24000 44100 48000"`

	// BitDepth specifies the audio bit depth
	BitDepth int `mapstructure:"bit_depth" yaml:"bit_depth" default:"16" validate:"oneof=16 24 32"`

	// EnableStreaming enables streaming TTS generation
	EnableStreaming bool `mapstructure:"enable_streaming" yaml:"enable_streaming" default:"true"`

	// EnableSSML enables SSML support
	EnableSSML bool `mapstructure:"enable_ssml" yaml:"enable_ssml" default:"false"`

	// Retry configuration
	MaxRetries   int           `mapstructure:"max_retries" yaml:"max_retries" default:"3" validate:"gte=0,lte=10"`
	RetryDelay   time.Duration `mapstructure:"retry_delay" yaml:"retry_delay" default:"1s" validate:"min=100ms,max=30s"`
	RetryBackoff float64       `mapstructure:"retry_backoff" yaml:"retry_backoff" default:"2.0" validate:"gte=1,lte=5"`

	// Provider-specific configuration
	ProviderSpecific map[string]interface{} `mapstructure:"provider_specific" yaml:"provider_specific"`

	// Observability settings
	EnableTracing           bool `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
	EnableMetrics           bool `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
	EnableStructuredLogging bool `mapstructure:"enable_structured_logging" yaml:"enable_structured_logging" default:"true"`
}

// ConfigOption is a functional option for configuring TTS instances
type ConfigOption func(*Config)

// WithProvider sets the TTS provider
func WithProvider(provider string) ConfigOption {
	return func(c *Config) {
		c.Provider = provider
	}
}

// WithAPIKey sets the API key
func WithAPIKey(apiKey string) ConfigOption {
	return func(c *Config) {
		c.APIKey = apiKey
	}
}

// WithBaseURL sets the base URL
func WithBaseURL(baseURL string) ConfigOption {
	return func(c *Config) {
		c.BaseURL = baseURL
	}
}

// WithModel sets the model
func WithModel(model string) ConfigOption {
	return func(c *Config) {
		c.Model = model
	}
}

// WithVoice sets the voice
func WithVoice(voice string) ConfigOption {
	return func(c *Config) {
		c.Voice = voice
	}
}

// WithLanguage sets the language
func WithLanguage(language string) ConfigOption {
	return func(c *Config) {
		c.Language = language
	}
}

// WithSpeed sets the speech speed
func WithSpeed(speed float64) ConfigOption {
	return func(c *Config) {
		c.Speed = speed
	}
}

// WithPitch sets the pitch
func WithPitch(pitch float64) ConfigOption {
	return func(c *Config) {
		c.Pitch = pitch
	}
}

// WithVolume sets the volume
func WithVolume(volume float64) ConfigOption {
	return func(c *Config) {
		c.Volume = volume
	}
}

// WithTimeout sets the timeout
func WithTimeout(timeout time.Duration) ConfigOption {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithSampleRate sets the sample rate
func WithSampleRate(sampleRate int) ConfigOption {
	return func(c *Config) {
		c.SampleRate = sampleRate
	}
}

// WithEnableStreaming sets streaming enablement
func WithEnableStreaming(enable bool) ConfigOption {
	return func(c *Config) {
		c.EnableStreaming = enable
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
