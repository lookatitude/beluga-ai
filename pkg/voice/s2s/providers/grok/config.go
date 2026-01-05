package grok

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
)

// GrokVoiceConfig extends the base S2S config with Grok Voice Agent specific settings.
type GrokVoiceConfig struct {
	*s2s.Config
	// APIEndpoint is the xAI API endpoint URL
	APIEndpoint string `mapstructure:"api_endpoint" yaml:"api_endpoint" default:"https://api.x.ai/v1"`

	// Model is the Grok Voice Agent model identifier
	Model string `mapstructure:"model" yaml:"model" default:"grok-voice-agent"`

	// VoiceID is the voice identifier for speech synthesis
	VoiceID string `mapstructure:"voice_id" yaml:"voice_id" default:"alloy"`

	// LanguageCode is the language code (e.g., "en-US")
	LanguageCode string `mapstructure:"language_code" yaml:"language_code" default:"en-US"`

	// Timeout for API requests
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`

	// EnableStreaming enables bidirectional streaming
	EnableStreaming bool `mapstructure:"enable_streaming" yaml:"enable_streaming" default:"true"`

	// AudioFormat specifies the audio format (PCM, Opus, etc.)
	AudioFormat string `mapstructure:"audio_format" yaml:"audio_format" default:"pcm"`

	// SampleRate for audio input/output
	SampleRate int `mapstructure:"sample_rate" yaml:"sample_rate" default:"24000"`

	// Temperature for response generation (0.0 to 2.0)
	Temperature float64 `mapstructure:"temperature" yaml:"temperature" default:"0.8"`

	// EnableAutomaticPunctuation enables automatic punctuation
	EnableAutomaticPunctuation bool `mapstructure:"enable_automatic_punctuation" yaml:"enable_automatic_punctuation" default:"true"`
}

// DefaultGrokVoiceConfig returns a default Grok Voice Agent configuration.
func DefaultGrokVoiceConfig() *GrokVoiceConfig {
	return &GrokVoiceConfig{
		Config:                     s2s.DefaultConfig(),
		APIEndpoint:                "https://api.x.ai/v1",
		Model:                      "grok-voice-agent",
		VoiceID:                    "alloy",
		LanguageCode:               "en-US",
		Timeout:                    30 * time.Second,
		EnableStreaming:            true,
		AudioFormat:                "pcm",
		SampleRate:                 24000,
		Temperature:                0.8,
		EnableAutomaticPunctuation: true,
	}
}
