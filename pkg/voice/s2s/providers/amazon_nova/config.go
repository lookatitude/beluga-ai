package amazon_nova

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
)

// AmazonNovaConfig extends the base S2S config with Amazon Nova 2 Sonic specific settings.
type AmazonNovaConfig struct {
	*s2s.Config
	// Region is the AWS region (e.g., "us-east-1")
	Region string `mapstructure:"region" yaml:"region" default:"us-east-1"`

	// Model is the Nova model identifier (default: "nova-2-sonic")
	Model string `mapstructure:"model" yaml:"model" default:"nova-2-sonic"`

	// VoiceID is the voice identifier for speech synthesis
	VoiceID string `mapstructure:"voice_id" yaml:"voice_id" default:"Ruth"`

	// LanguageCode is the language code (e.g., "en-US")
	LanguageCode string `mapstructure:"language_code" yaml:"language_code" default:"en-US"`

	// EndpointURL is the AWS endpoint URL (optional, for testing)
	EndpointURL string `mapstructure:"endpoint_url" yaml:"endpoint_url"`

	// Timeout for API requests
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`

	// EnableStreaming enables bidirectional streaming
	EnableStreaming bool `mapstructure:"enable_streaming" yaml:"enable_streaming" default:"true"`

	// AudioFormat specifies the audio format (PCM, Opus, etc.)
	AudioFormat string `mapstructure:"audio_format" yaml:"audio_format" default:"pcm"`

	// SampleRate for audio input/output
	SampleRate int `mapstructure:"sample_rate" yaml:"sample_rate" default:"24000"`

	// EnableAutomaticPunctuation enables automatic punctuation
	EnableAutomaticPunctuation bool `mapstructure:"enable_automatic_punctuation" yaml:"enable_automatic_punctuation" default:"true"`
}

// DefaultAmazonNovaConfig returns a default Amazon Nova 2 Sonic configuration.
func DefaultAmazonNovaConfig() *AmazonNovaConfig {
	return &AmazonNovaConfig{
		Config:                     s2s.DefaultConfig(),
		Region:                     "us-east-1",
		Model:                      "nova-2-sonic",
		VoiceID:                    "Ruth",
		LanguageCode:               "en-US",
		Timeout:                    30 * time.Second,
		EnableStreaming:            true,
		AudioFormat:                "pcm",
		SampleRate:                 24000,
		EnableAutomaticPunctuation: true,
	}
}
