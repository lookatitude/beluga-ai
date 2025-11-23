package transport

import (
	"time"

	"github.com/go-playground/validator/v10"
)

// Config represents the configuration for Transport providers.
// It includes common settings that apply to all Transport providers.
type Config struct {
	// Provider specifies the Transport provider (e.g., "webrtc", "websocket", "sip")
	Provider string `mapstructure:"provider" yaml:"provider" validate:"required,oneof=webrtc websocket sip"`

	// URL specifies the connection URL
	URL string `mapstructure:"url" yaml:"url" validate:"required"`

	// AudioFormat specifies the audio format
	SampleRate int `mapstructure:"sample_rate" yaml:"sample_rate" default:"16000" validate:"oneof=8000 16000 22050 24000 32000 44100 48000"`
	Channels   int `mapstructure:"channels" yaml:"channels" default:"1" validate:"oneof=1 2"`
	BitDepth   int `mapstructure:"bit_depth" yaml:"bit_depth" default:"16" validate:"oneof=16 24 32"`

	// Codec specifies the audio codec (e.g., "pcm", "opus", "g711")
	Codec string `mapstructure:"codec" yaml:"codec" default:"pcm" validate:"oneof=pcm opus g711"`

	// Connection settings
	ConnectTimeout    time.Duration `mapstructure:"connect_timeout" yaml:"connect_timeout" default:"10s" validate:"min=1s,max=60s"`
	ReconnectAttempts int           `mapstructure:"reconnect_attempts" yaml:"reconnect_attempts" default:"3" validate:"gte=0,lte=10"`
	ReconnectDelay    time.Duration `mapstructure:"reconnect_delay" yaml:"reconnect_delay" default:"1s" validate:"min=100ms,max=30s"`

	// Buffer settings
	SendBufferSize    int `mapstructure:"send_buffer_size" yaml:"send_buffer_size" default:"4096" validate:"min=1024,max=65536"`
	ReceiveBufferSize int `mapstructure:"receive_buffer_size" yaml:"receive_buffer_size" default:"4096" validate:"min=1024,max=65536"`

	// Timeout for operations
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s" validate:"min=1s,max=5m"`

	// Provider-specific configuration
	ProviderSpecific map[string]interface{} `mapstructure:"provider_specific" yaml:"provider_specific"`

	// Observability settings
	EnableTracing           bool `mapstructure:"enable_tracing" yaml:"enable_tracing" default:"true"`
	EnableMetrics           bool `mapstructure:"enable_metrics" yaml:"enable_metrics" default:"true"`
	EnableStructuredLogging bool `mapstructure:"enable_structured_logging" yaml:"enable_structured_logging" default:"true"`
}

// ConfigOption is a functional option for configuring Transport instances
type ConfigOption func(*Config)

// WithProvider sets the Transport provider
func WithProvider(provider string) ConfigOption {
	return func(c *Config) {
		c.Provider = provider
	}
}

// WithURL sets the connection URL
func WithURL(url string) ConfigOption {
	return func(c *Config) {
		c.URL = url
	}
}

// WithSampleRate sets the sample rate
func WithSampleRate(sampleRate int) ConfigOption {
	return func(c *Config) {
		c.SampleRate = sampleRate
	}
}

// WithChannels sets the number of channels
func WithChannels(channels int) ConfigOption {
	return func(c *Config) {
		c.Channels = channels
	}
}

// WithCodec sets the audio codec
func WithCodec(codec string) ConfigOption {
	return func(c *Config) {
		c.Codec = codec
	}
}

// WithConnectTimeout sets the connection timeout
func WithConnectTimeout(timeout time.Duration) ConfigOption {
	return func(c *Config) {
		c.ConnectTimeout = timeout
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
		Provider:                "webrtc",
		URL:                     "",
		SampleRate:              16000,
		Channels:                1,
		BitDepth:                16,
		Codec:                   "pcm",
		ConnectTimeout:          10 * time.Second,
		ReconnectAttempts:       3,
		ReconnectDelay:          1 * time.Second,
		SendBufferSize:          4096,
		ReceiveBufferSize:       4096,
		Timeout:                 30 * time.Second,
		EnableTracing:           true,
		EnableMetrics:           true,
		EnableStructuredLogging: true,
	}
}
