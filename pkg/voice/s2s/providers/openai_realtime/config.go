package openai_realtime

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
)

// OpenAIRealtimeConfig extends the base S2S config with OpenAI Realtime API specific settings.
type OpenAIRealtimeConfig struct {
	*s2s.Config
	// APIEndpoint is the OpenAI API endpoint URL
	APIEndpoint string `mapstructure:"api_endpoint" yaml:"api_endpoint" default:"https://api.openai.com/v1"`

	// Model is the GPT Realtime model identifier (default: "gpt-4o-realtime-preview")
	Model string `mapstructure:"model" yaml:"model" default:"gpt-4o-realtime-preview"`

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

// DefaultOpenAIRealtimeConfig returns a default OpenAI Realtime API configuration.
func DefaultOpenAIRealtimeConfig() *OpenAIRealtimeConfig {
	return &OpenAIRealtimeConfig{
		Config:                     s2s.DefaultConfig(),
		APIEndpoint:                "https://api.openai.com/v1",
		Model:                      "gpt-4o-realtime-preview",
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
