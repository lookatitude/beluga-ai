package elevenlabs

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
)

// ElevenLabsConfig extends the base TTS config with ElevenLabs specific settings
type ElevenLabsConfig struct {
	*tts.Config

	// VoiceID specifies the voice ID (e.g., "21m00Tcm4TlvDq8ikWAM")
	VoiceID string `mapstructure:"voice_id" yaml:"voice_id" validate:"required"`

	// ModelID specifies the model ID (e.g., "eleven_monolingual_v1", "eleven_multilingual_v1")
	ModelID string `mapstructure:"model_id" yaml:"model_id" default:"eleven_monolingual_v1"`

	// Stability specifies the stability (0.0-1.0, default: 0.5)
	Stability float64 `mapstructure:"stability" yaml:"stability" default:"0.5" validate:"gte=0.0,lte=1.0"`

	// SimilarityBoost specifies the similarity boost (0.0-1.0, default: 0.5)
	SimilarityBoost float64 `mapstructure:"similarity_boost" yaml:"similarity_boost" default:"0.5" validate:"gte=0.0,lte=1.0"`

	// Style specifies the style (0.0-1.0, default: 0.0)
	Style float64 `mapstructure:"style" yaml:"style" default:"0.0" validate:"gte=0.0,lte=1.0"`

	// UseSpeakerBoost enables speaker boost
	UseSpeakerBoost bool `mapstructure:"use_speaker_boost" yaml:"use_speaker_boost" default:"true"`

	// OutputFormat specifies the output format ("mp3_44100_128", "mp3_44100_192", "pcm_16000", "pcm_22050", "pcm_24000", "pcm_44100", "ulaw_8000")
	OutputFormat string `mapstructure:"output_format" yaml:"output_format" default:"mp3_44100_128"`

	// BaseURL for ElevenLabs API (default: https://api.elevenlabs.io)
	BaseURL string `mapstructure:"base_url" yaml:"base_url" default:"https://api.elevenlabs.io"`

	// Timeout for API calls
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
}

// DefaultElevenLabsConfig returns a default ElevenLabs configuration
func DefaultElevenLabsConfig() *ElevenLabsConfig {
	return &ElevenLabsConfig{
		Config:          tts.DefaultConfig(),
		VoiceID:         "",
		ModelID:         "eleven_monolingual_v1",
		Stability:       0.5,
		SimilarityBoost: 0.5,
		Style:           0.0,
		UseSpeakerBoost: true,
		OutputFormat:    "mp3_44100_128",
		BaseURL:         "https://api.elevenlabs.io",
		Timeout:         30 * time.Second,
	}
}
