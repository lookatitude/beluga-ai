package openai

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

// OpenAIConfig extends the base STT config with OpenAI Whisper specific settings.
type OpenAIConfig struct {
	*stt.Config
	Model          string        `mapstructure:"model" yaml:"model" default:"whisper-1"`
	Language       string        `mapstructure:"language" yaml:"language"`
	Prompt         string        `mapstructure:"prompt" yaml:"prompt"`
	ResponseFormat string        `mapstructure:"response_format" yaml:"response_format" default:"json"`
	BaseURL        string        `mapstructure:"base_url" yaml:"base_url" default:"https://api.openai.com"`
	Temperature    float64       `mapstructure:"temperature" yaml:"temperature" default:"0.0" validate:"gte=0,lte=1"`
	Timeout        time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
}

// DefaultOpenAIConfig returns a default OpenAI Whisper configuration.
func DefaultOpenAIConfig() *OpenAIConfig {
	return &OpenAIConfig{
		Config:         stt.DefaultConfig(),
		Model:          "whisper-1",
		Language:       "",
		Prompt:         "",
		ResponseFormat: "json",
		Temperature:    0.0,
		BaseURL:        "https://api.openai.com",
		Timeout:        30 * time.Second,
	}
}
