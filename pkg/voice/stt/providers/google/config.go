package google

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

// GoogleConfig extends the base STT config with Google Cloud Speech-to-Text specific settings.
type GoogleConfig struct {
	*stt.Config
	Model                               string        `mapstructure:"model" yaml:"model" default:"default"`
	LanguageCode                        string        `mapstructure:"language_code" yaml:"language_code" default:"en-US"`
	BaseURL                             string        `mapstructure:"base_url" yaml:"base_url" default:"https://speech.googleapis.com"`
	CredentialsJSON                     string        `mapstructure:"credentials_json" yaml:"credentials_json"`
	ProjectID                           string        `mapstructure:"project_id" yaml:"project_id"`
	AlternativeLanguageCodes            []string      `mapstructure:"alternative_language_codes" yaml:"alternative_language_codes"`
	DiarizationSpeakerCount             int           `mapstructure:"diarization_speaker_count" yaml:"diarization_speaker_count" default:"0"`
	AudioChannelCount                   int           `mapstructure:"audio_channel_count" yaml:"audio_channel_count" default:"1"`
	Timeout                             time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
	EnableSpeakerDiarization            bool          `mapstructure:"enable_speaker_diarization" yaml:"enable_speaker_diarization" default:"false"`
	EnableSeparateRecognitionPerChannel bool          `mapstructure:"enable_separate_recognition_per_channel" yaml:"enable_separate_recognition_per_channel" default:"false"`
	UseEnhanced                         bool          `mapstructure:"use_enhanced" yaml:"use_enhanced" default:"false"`
	EnableWordConfidence                bool          `mapstructure:"enable_word_confidence" yaml:"enable_word_confidence" default:"false"`
	EnableWordTimeOffsets               bool          `mapstructure:"enable_word_time_offsets" yaml:"enable_word_time_offsets" default:"false"`
	EnableAutomaticPunctuation          bool          `mapstructure:"enable_automatic_punctuation" yaml:"enable_automatic_punctuation" default:"true"`
}

// DefaultGoogleConfig returns a default Google Cloud Speech-to-Text configuration.
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
