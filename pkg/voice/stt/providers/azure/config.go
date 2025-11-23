package azure

import (
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

// AzureConfig extends the base STT config with Azure Speech Services specific settings
type AzureConfig struct {
	*stt.Config

	// Region specifies the Azure region (e.g., "eastus", "westus2")
	Region string `mapstructure:"region" yaml:"region" validate:"required"`

	// Language specifies the language code (BCP-47 format, e.g., "en-US", "es-ES")
	Language string `mapstructure:"language" yaml:"language" default:"en-US"`

	// EndpointID specifies a custom endpoint ID for custom models
	EndpointID string `mapstructure:"endpoint_id" yaml:"endpoint_id"`

	// Model specifies a custom model name
	Model string `mapstructure:"model" yaml:"model"`

	// EnablePunctuation enables automatic punctuation
	EnablePunctuation bool `mapstructure:"enable_punctuation" yaml:"enable_punctuation" default:"true"`

	// EnableWordLevelTimestamps enables word-level timestamps
	EnableWordLevelTimestamps bool `mapstructure:"enable_word_level_timestamps" yaml:"enable_word_level_timestamps" default:"false"`

	// EnableProfanityFilter enables profanity filtering
	EnableProfanityFilter bool `mapstructure:"enable_profanity_filter" yaml:"enable_profanity_filter" default:"false"`

	// ProfanityFilterMode specifies profanity filter mode ("masked", "removed", "raw")
	ProfanityFilterMode string `mapstructure:"profanity_filter_mode" yaml:"profanity_filter_mode" default:"masked"`

	// EnableSpeakerDiarization enables speaker diarization
	EnableSpeakerDiarization bool `mapstructure:"enable_speaker_diarization" yaml:"enable_speaker_diarization" default:"false"`

	// EnableLanguageIdentification enables automatic language identification
	EnableLanguageIdentification bool `mapstructure:"enable_language_identification" yaml:"enable_language_identification" default:"false"`

	// CandidateLanguages specifies candidate languages for identification
	CandidateLanguages []string `mapstructure:"candidate_languages" yaml:"candidate_languages"`

	// BaseURL for Azure Speech Services (default: https://{region}.stt.speech.microsoft.com)
	BaseURL string `mapstructure:"base_url" yaml:"base_url"`

	// WebSocketURL for Azure Speech Services WebSocket (default: wss://{region}.stt.speech.microsoft.com/speech/transcription/conversation/cognitiveservices/v1)
	WebSocketURL string `mapstructure:"websocket_url" yaml:"websocket_url"`

	// Timeout for API calls
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
}

// DefaultAzureConfig returns a default Azure Speech Services configuration
func DefaultAzureConfig() *AzureConfig {
	return &AzureConfig{
		Config:                       stt.DefaultConfig(),
		Region:                       "eastus",
		Language:                     "en-US",
		EnablePunctuation:            true,
		EnableWordLevelTimestamps:    false,
		EnableProfanityFilter:        false,
		ProfanityFilterMode:          "masked",
		EnableSpeakerDiarization:     false,
		EnableLanguageIdentification: false,
		Timeout:                      30 * time.Second,
	}
}

// GetBaseURL returns the base URL, constructing it from region if not set
func (c *AzureConfig) GetBaseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return fmt.Sprintf("https://%s.stt.speech.microsoft.com", c.Region)
}

// GetWebSocketURL returns the WebSocket URL, constructing it from region if not set
func (c *AzureConfig) GetWebSocketURL() string {
	if c.WebSocketURL != "" {
		return c.WebSocketURL
	}
	return fmt.Sprintf("wss://%s.stt.speech.microsoft.com/speech/transcription/conversation/cognitiveservices/v1", c.Region)
}
