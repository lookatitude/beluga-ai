package google

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

// GoogleConfig extends the base STT config with Google Cloud Speech-to-Text specific settings
type GoogleConfig struct {
	*stt.Config

	// Model specifies the Google Cloud Speech model (e.g., "default", "latest_short", "latest_long")
	Model string `mapstructure:"model" yaml:"model" default:"default"`

	// LanguageCode specifies the language code (BCP-47 format, e.g., "en-US", "es-ES")
	LanguageCode string `mapstructure:"language_code" yaml:"language_code" default:"en-US"`

	// AlternativeLanguageCodes specifies additional language codes to try
	AlternativeLanguageCodes []string `mapstructure:"alternative_language_codes" yaml:"alternative_language_codes"`

	// EnableAutomaticPunctuation enables automatic punctuation
	EnableAutomaticPunctuation bool `mapstructure:"enable_automatic_punctuation" yaml:"enable_automatic_punctuation" default:"true"`

	// EnableWordTimeOffsets enables word-level time offsets
	EnableWordTimeOffsets bool `mapstructure:"enable_word_time_offsets" yaml:"enable_word_time_offsets" default:"false"`

	// EnableWordConfidence enables word-level confidence scores
	EnableWordConfidence bool `mapstructure:"enable_word_confidence" yaml:"enable_word_confidence" default:"false"`

	// EnableSpeakerDiarization enables speaker diarization
	EnableSpeakerDiarization bool `mapstructure:"enable_speaker_diarization" yaml:"enable_speaker_diarization" default:"false"`

	// DiarizationSpeakerCount specifies the number of speakers (required if diarization enabled)
	DiarizationSpeakerCount int `mapstructure:"diarization_speaker_count" yaml:"diarization_speaker_count" default:"0"`

	// AudioChannelCount specifies the number of audio channels
	AudioChannelCount int `mapstructure:"audio_channel_count" yaml:"audio_channel_count" default:"1"`

	// EnableSeparateRecognitionPerChannel enables separate recognition per channel
	EnableSeparateRecognitionPerChannel bool `mapstructure:"enable_separate_recognition_per_channel" yaml:"enable_separate_recognition_per_channel" default:"false"`

	// UseEnhanced enables enhanced models (premium feature)
	UseEnhanced bool `mapstructure:"use_enhanced" yaml:"use_enhanced" default:"false"`

	// ProjectID specifies the Google Cloud project ID
	ProjectID string `mapstructure:"project_id" yaml:"project_id"`

	// CredentialsJSON specifies the path to Google Cloud credentials JSON file
	CredentialsJSON string `mapstructure:"credentials_json" yaml:"credentials_json"`

	// BaseURL for Google Cloud Speech API (default: https://speech.googleapis.com)
	BaseURL string `mapstructure:"base_url" yaml:"base_url" default:"https://speech.googleapis.com"`

	// Timeout for API calls
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
}

// DefaultGoogleConfig returns a default Google Cloud Speech-to-Text configuration
func DefaultGoogleConfig() *GoogleConfig {
	return &GoogleConfig{
		Config:                              stt.DefaultConfig(),
		Model:                               "default",
		LanguageCode:                        "en-US",
		AlternativeLanguageCodes:            []string{},
		EnableAutomaticPunctuation:          true,
		EnableWordTimeOffsets:               false,
		EnableWordConfidence:                false,
		EnableSpeakerDiarization:            false,
		DiarizationSpeakerCount:             0,
		AudioChannelCount:                   1,
		EnableSeparateRecognitionPerChannel: false,
		UseEnhanced:                         false,
		BaseURL:                             "https://speech.googleapis.com",
		Timeout:                             30 * time.Second,
	}
}
