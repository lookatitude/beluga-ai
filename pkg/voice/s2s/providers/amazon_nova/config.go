package amazon_nova

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
)

// AmazonNovaConfig extends the base S2S config with Amazon Nova 2 Sonic specific settings.
type AmazonNovaConfig struct {
	*s2s.Config
	Region                     string        `mapstructure:"region" yaml:"region" default:"us-east-1"`
	Model                      string        `mapstructure:"model" yaml:"model" default:"nova-2-sonic"`
	VoiceID                    string        `mapstructure:"voice_id" yaml:"voice_id" default:"Ruth"`
	LanguageCode               string        `mapstructure:"language_code" yaml:"language_code" default:"en-US"`
	EndpointURL                string        `mapstructure:"endpoint_url" yaml:"endpoint_url"`
	AudioFormat                string        `mapstructure:"audio_format" yaml:"audio_format" default:"pcm"`
	Timeout                    time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
	SampleRate                 int           `mapstructure:"sample_rate" yaml:"sample_rate" default:"24000"`
	EnableStreaming            bool          `mapstructure:"enable_streaming" yaml:"enable_streaming" default:"true"`
	EnableAutomaticPunctuation bool          `mapstructure:"enable_automatic_punctuation" yaml:"enable_automatic_punctuation" default:"true"`
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
