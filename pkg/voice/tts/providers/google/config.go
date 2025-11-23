package google

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
)

// GoogleConfig extends the base TTS config with Google Cloud Text-to-Speech specific settings
type GoogleConfig struct {
	*tts.Config

	// VoiceName specifies the voice name (e.g., "en-US-Standard-A", "en-US-Wavenet-D")
	VoiceName string `mapstructure:"voice_name" yaml:"voice_name" default:"en-US-Standard-A"`

	// LanguageCode specifies the language code (BCP-47 format, e.g., "en-US", "es-ES")
	LanguageCode string `mapstructure:"language_code" yaml:"language_code" default:"en-US"`

	// SSMLGender specifies the gender ("NEUTRAL", "FEMALE", "MALE")
	SSMLGender string `mapstructure:"ssml_gender" yaml:"ssml_gender" default:"NEUTRAL" validate:"oneof=NEUTRAL FEMALE MALE"`

	// SpeakingRate specifies the speaking rate (0.25-4.0, default: 1.0)
	SpeakingRate float64 `mapstructure:"speaking_rate" yaml:"speaking_rate" default:"1.0" validate:"gte=0.25,lte=4.0"`

	// Pitch specifies the pitch adjustment (-20.0 to 20.0 semitones, default: 0.0)
	Pitch float64 `mapstructure:"pitch" yaml:"pitch" default:"0.0" validate:"gte=-20.0,lte=20.0"`

	// VolumeGainDb specifies the volume gain in dB (-96.0 to 16.0, default: 0.0)
	VolumeGainDb float64 `mapstructure:"volume_gain_db" yaml:"volume_gain_db" default:"0.0" validate:"gte=-96.0,lte=16.0"`

	// AudioEncoding specifies the audio encoding ("MP3", "LINEAR16", "OGG_OPUS", "AUDIO_ENCODING_UNSPECIFIED")
	AudioEncoding string `mapstructure:"audio_encoding" yaml:"audio_encoding" default:"MP3" validate:"oneof=MP3 LINEAR16 OGG_OPUS AUDIO_ENCODING_UNSPECIFIED"`

	// SampleRateHertz specifies the sample rate in Hz
	SampleRateHertz int `mapstructure:"sample_rate_hertz" yaml:"sample_rate_hertz" default:"24000" validate:"oneof=8000 11025 16000 22050 24000 32000 44100 48000"`

	// ProjectID specifies the Google Cloud project ID
	ProjectID string `mapstructure:"project_id" yaml:"project_id"`

	// CredentialsJSON specifies the path to Google Cloud credentials JSON file
	CredentialsJSON string `mapstructure:"credentials_json" yaml:"credentials_json"`

	// BaseURL for Google Cloud Text-to-Speech API (default: https://texttospeech.googleapis.com)
	BaseURL string `mapstructure:"base_url" yaml:"base_url" default:"https://texttospeech.googleapis.com"`

	// Timeout for API calls
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
}

// DefaultGoogleConfig returns a default Google Cloud Text-to-Speech configuration
func DefaultGoogleConfig() *GoogleConfig {
	return &GoogleConfig{
		Config:          tts.DefaultConfig(),
		VoiceName:       "en-US-Standard-A",
		LanguageCode:    "en-US",
		SSMLGender:      "NEUTRAL",
		SpeakingRate:    1.0,
		Pitch:           0.0,
		VolumeGainDb:    0.0,
		AudioEncoding:   "MP3",
		SampleRateHertz: 24000,
		BaseURL:         "https://texttospeech.googleapis.com",
		Timeout:         30 * time.Second,
	}
}
