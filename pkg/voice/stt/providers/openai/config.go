package openai

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

// OpenAIConfig extends the base STT config with OpenAI Whisper specific settings
type OpenAIConfig struct {
	*stt.Config

	// Model specifies the Whisper model (e.g., "whisper-1")
	Model string `mapstructure:"model" yaml:"model" default:"whisper-1"`

	// Language specifies the language code (ISO 639-1, e.g., "en", "es")
	// If empty, language will be auto-detected
	Language string `mapstructure:"language" yaml:"language"`

	// Prompt specifies an optional text prompt to guide the model's style
	Prompt string `mapstructure:"prompt" yaml:"prompt"`

	// ResponseFormat specifies the response format ("json", "text", "srt", "verbose_json", "vtt")
	ResponseFormat string `mapstructure:"response_format" yaml:"response_format" default:"json"`

	// Temperature specifies the sampling temperature (0.0 to 1.0)
	Temperature float64 `mapstructure:"temperature" yaml:"temperature" default:"0.0" validate:"gte=0,lte=1"`

	// BaseURL for OpenAI API (default: https://api.openai.com)
	BaseURL string `mapstructure:"base_url" yaml:"base_url" default:"https://api.openai.com"`

	// Timeout for API calls
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
}

// DefaultOpenAIConfig returns a default OpenAI Whisper configuration
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
