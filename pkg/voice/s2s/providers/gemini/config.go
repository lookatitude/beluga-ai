package gemini

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
)

// GeminiNativeConfig extends the base S2S config with Gemini 2.5 Flash Native Audio specific settings.
type GeminiNativeConfig struct {
	*s2s.Config
	// APIEndpoint is the Google Cloud/Vertex AI API endpoint URL
	APIEndpoint string `mapstructure:"api_endpoint" yaml:"api_endpoint" default:"https://generativelanguage.googleapis.com/v1beta"`

	// Model is the Gemini model identifier (default: "gemini-2.5-flash")
	Model string `mapstructure:"model" yaml:"model" default:"gemini-2.5-flash"`

	// ProjectID is the Google Cloud project ID
	ProjectID string `mapstructure:"project_id" yaml:"project_id"`

	// Location is the Google Cloud region (e.g., "us-central1")
	Location string `mapstructure:"location" yaml:"location" default:"us-central1"`

	// VoiceID is the voice identifier for speech synthesis
	VoiceID string `mapstructure:"voice_id" yaml:"voice_id" default:"default"`

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

// DefaultGeminiNativeConfig returns a default Gemini 2.5 Flash Native Audio configuration.
func DefaultGeminiNativeConfig() *GeminiNativeConfig {
	return &GeminiNativeConfig{
		Config:                     s2s.DefaultConfig(),
		APIEndpoint:                "https://generativelanguage.googleapis.com/v1beta",
		Model:                      "gemini-2.5-flash",
		Location:                   "us-central1",
		VoiceID:                    "default",
		LanguageCode:               "en-US",
		Timeout:                    30 * time.Second,
		EnableStreaming:            true,
		AudioFormat:                "pcm",
		SampleRate:                 24000,
		Temperature:                0.8,
		EnableAutomaticPunctuation: true,
	}
}
