package azure

import (
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
)

// AzureConfig extends the base TTS config with Azure Speech Services specific settings
type AzureConfig struct {
	*tts.Config

	// Region specifies the Azure region (e.g., "eastus", "westus2")
	Region string `mapstructure:"region" yaml:"region" validate:"required"`

	// VoiceName specifies the voice name (e.g., "en-US-AriaNeural", "en-US-GuyNeural")
	VoiceName string `mapstructure:"voice_name" yaml:"voice_name" default:"en-US-AriaNeural"`

	// Language specifies the language code (BCP-47 format, e.g., "en-US", "es-ES")
	Language string `mapstructure:"language" yaml:"language" default:"en-US"`

	// VoiceStyle specifies the voice style (e.g., "cheerful", "sad", "angry")
	VoiceStyle string `mapstructure:"voice_style" yaml:"voice_style"`

	// VoiceRate specifies the speaking rate ("x-slow", "slow", "medium", "fast", "x-fast", or percentage)
	VoiceRate string `mapstructure:"voice_rate" yaml:"voice_rate" default:"medium"`

	// VoicePitch specifies the pitch adjustment ("x-low", "low", "medium", "high", "x-high", or +/- percentage)
	VoicePitch string `mapstructure:"voice_pitch" yaml:"voice_pitch" default:"medium"`

	// AudioFormat specifies the audio format ("audio-16khz-128kbitrate-mono-mp3", "audio-24khz-48kbitrate-mono-mp3", etc.)
	AudioFormat string `mapstructure:"audio_format" yaml:"audio_format" default:"audio-24khz-48kbitrate-mono-mp3"`

	// BaseURL for Azure Speech Services (default: https://{region}.tts.speech.microsoft.com)
	BaseURL string `mapstructure:"base_url" yaml:"base_url"`

	// Timeout for API calls
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
}

// DefaultAzureConfig returns a default Azure Speech Services configuration
func DefaultAzureConfig() *AzureConfig {
	return &AzureConfig{
		Config:      tts.DefaultConfig(),
		Region:      "eastus",
		VoiceName:   "en-US-AriaNeural",
		Language:    "en-US",
		VoiceRate:   "medium",
		VoicePitch:  "medium",
		AudioFormat: "audio-24khz-48kbitrate-mono-mp3",
		Timeout:     30 * time.Second,
	}
}

// GetBaseURL returns the base URL, constructing it from region if not set
func (c *AzureConfig) GetBaseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return fmt.Sprintf("https://%s.tts.speech.microsoft.com", c.Region)
}
