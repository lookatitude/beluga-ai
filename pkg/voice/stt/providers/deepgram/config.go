package deepgram

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

// DeepgramConfig extends the base STT config with Deepgram-specific settings.
type DeepgramConfig struct {
	*stt.Config
	Model          string        `mapstructure:"model" yaml:"model" default:"nova-2"`
	Language       string        `mapstructure:"language" yaml:"language" default:"en"`
	Tier           string        `mapstructure:"tier" yaml:"tier" default:"nova"`
	WebSocketURL   string        `mapstructure:"websocket_url" yaml:"websocket_url" default:"wss://api.deepgram.com/v1/listen"`
	BaseURL        string        `mapstructure:"base_url" yaml:"base_url" default:"https://api.deepgram.com"`
	Endpointing    int           `mapstructure:"endpointing" yaml:"endpointing" default:"300"`
	UtteranceEndMs int           `mapstructure:"utterance_end_ms" yaml:"utterance_end_ms" default:"700"`
	Timeout        time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
	Multichannel   bool          `mapstructure:"multichannel" yaml:"multichannel" default:"false"`
	InterimResults bool          `mapstructure:"interim_results" yaml:"interim_results" default:"true"`
	SmartFormat    bool          `mapstructure:"smart_format" yaml:"smart_format" default:"true"`
	VADEvents      bool          `mapstructure:"vad_events" yaml:"vad_events" default:"false"`
	Diarize        bool          `mapstructure:"diarize" yaml:"diarize" default:"false"`
	Punctuate      bool          `mapstructure:"punctuate" yaml:"punctuate" default:"true"`
}

// DefaultDeepgramConfig returns a default Deepgram configuration.
func DefaultDeepgramConfig() *DeepgramConfig {
	return &DeepgramConfig{
		Config:         stt.DefaultConfig(),
		Model:          "nova-2",
		Language:       "en",
		Tier:           "nova",
		Punctuate:      true,
		Diarize:        false,
		SmartFormat:    true,
		Multichannel:   false,
		InterimResults: true,
		Endpointing:    300,
		VADEvents:      false,
		UtteranceEndMs: 700,
		BaseURL:        "https://api.deepgram.com",
		WebSocketURL:   "wss://api.deepgram.com/v1/listen",
		Timeout:        30 * time.Second,
	}
}
