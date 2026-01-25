package gemini

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
)

// GeminiNativeConfig extends the base S2S config with Gemini 2.5 Flash Native Audio specific settings.
type GeminiNativeConfig struct {
	*s2s.Config
	LanguageCode               string        `mapstructure:"language_code" yaml:"language_code" default:"en-US"`
	Model                      string        `mapstructure:"model" yaml:"model" default:"gemini-2.5-flash"`
	ProjectID                  string        `mapstructure:"project_id" yaml:"project_id"`
	Location                   string        `mapstructure:"location" yaml:"location" default:"us-central1"`
	VoiceID                    string        `mapstructure:"voice_id" yaml:"voice_id" default:"default"`
	APIEndpoint                string        `mapstructure:"api_endpoint" yaml:"api_endpoint" default:"https://generativelanguage.googleapis.com/v1beta"`
	AudioFormat                string        `mapstructure:"audio_format" yaml:"audio_format" default:"pcm"`
	Timeout                    time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
	SampleRate                 int           `mapstructure:"sample_rate" yaml:"sample_rate" default:"24000"`
	Temperature                float64       `mapstructure:"temperature" yaml:"temperature" default:"0.8"`
	EnableStreaming            bool          `mapstructure:"enable_streaming" yaml:"enable_streaming" default:"true"`
	EnableAutomaticPunctuation bool          `mapstructure:"enable_automatic_punctuation" yaml:"enable_automatic_punctuation" default:"true"`
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
