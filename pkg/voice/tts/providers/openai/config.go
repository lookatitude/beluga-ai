package openai

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
)

// OpenAIConfig extends the base TTS config with OpenAI TTS specific settings.
type OpenAIConfig struct {
	*tts.Config
	Model          string        `mapstructure:"model" yaml:"model" default:"tts-1"`
	Voice          string        `mapstructure:"voice" yaml:"voice" default:"alloy" validate:"oneof=alloy echo fable onyx nova shimmer"`
	ResponseFormat string        `mapstructure:"response_format" yaml:"response_format" default:"mp3" validate:"oneof=mp3 opus aac flac pcm"`
	BaseURL        string        `mapstructure:"base_url" yaml:"base_url" default:"https://api.openai.com"`
	Speed          float64       `mapstructure:"speed" yaml:"speed" default:"1.0" validate:"gte=0.25,lte=4.0"`
	Timeout        time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
}

// DefaultOpenAIConfig returns a default OpenAI TTS configuration.
func DefaultOpenAIConfig() *OpenAIConfig {
	return &OpenAIConfig{
		Config:         tts.DefaultConfig(),
		Model:          "tts-1",
		Voice:          "alloy",
		ResponseFormat: "mp3",
		Speed:          1.0,
		BaseURL:        "https://api.openai.com",
		Timeout:        30 * time.Second,
	}
}
