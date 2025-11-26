package elevenlabs

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
)

// ElevenLabsConfig extends the base TTS config with ElevenLabs specific settings.
type ElevenLabsConfig struct {
	*tts.Config
	VoiceID         string        `mapstructure:"voice_id" yaml:"voice_id" validate:"required"`
	ModelID         string        `mapstructure:"model_id" yaml:"model_id" default:"eleven_monolingual_v1"`
	OutputFormat    string        `mapstructure:"output_format" yaml:"output_format" default:"mp3_44100_128"`
	BaseURL         string        `mapstructure:"base_url" yaml:"base_url" default:"https://api.elevenlabs.io"`
	Stability       float64       `mapstructure:"stability" yaml:"stability" default:"0.5" validate:"gte=0.0,lte=1.0"`
	SimilarityBoost float64       `mapstructure:"similarity_boost" yaml:"similarity_boost" default:"0.5" validate:"gte=0.0,lte=1.0"`
	Style           float64       `mapstructure:"style" yaml:"style" default:"0.0" validate:"gte=0.0,lte=1.0"`
	Timeout         time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
	UseSpeakerBoost bool          `mapstructure:"use_speaker_boost" yaml:"use_speaker_boost" default:"true"`
}

// DefaultElevenLabsConfig returns a default ElevenLabs configuration.
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
