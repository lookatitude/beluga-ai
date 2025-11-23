package deepgram

import (
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

// DeepgramConfig extends the base STT config with Deepgram-specific settings
type DeepgramConfig struct {
	*stt.Config

	// Model specifies the Deepgram model (e.g., "nova-2", "nova-3", "whisper-large")
	Model string `mapstructure:"model" yaml:"model" default:"nova-2"`

	// Language specifies the language code (ISO 639-1)
	Language string `mapstructure:"language" yaml:"language" default:"en"`

	// Tier specifies the Deepgram tier (e.g., "nova", "base", "enhanced")
	Tier string `mapstructure:"tier" yaml:"tier" default:"nova"`

	// Punctuate enables punctuation in transcriptions
	Punctuate bool `mapstructure:"punctuate" yaml:"punctuate" default:"true"`

	// Diarize enables speaker diarization
	Diarize bool `mapstructure:"diarize" yaml:"diarize" default:"false"`

	// SmartFormat enables smart formatting
	SmartFormat bool `mapstructure:"smart_format" yaml:"smart_format" default:"true"`

	// Multichannel enables multichannel processing
	Multichannel bool `mapstructure:"multichannel" yaml:"multichannel" default:"false"`

	// InterimResults enables interim results in streaming
	InterimResults bool `mapstructure:"interim_results" yaml:"interim_results" default:"true"`

	// Endpointing enables endpointing detection
	Endpointing int `mapstructure:"endpointing" yaml:"endpointing" default:"300"` // milliseconds

	// VADEvents enables VAD events
	VADEvents bool `mapstructure:"vad_events" yaml:"vad_events" default:"false"`

	// UtteranceEndMs specifies utterance end detection in milliseconds
	UtteranceEndMs int `mapstructure:"utterance_end_ms" yaml:"utterance_end_ms" default:"700"`

	// BaseURL for Deepgram API (default: https://api.deepgram.com)
	BaseURL string `mapstructure:"base_url" yaml:"base_url" default:"https://api.deepgram.com"`

	// WebSocketURL for Deepgram WebSocket API (default: wss://api.deepgram.com/v1/listen)
	WebSocketURL string `mapstructure:"websocket_url" yaml:"websocket_url" default:"wss://api.deepgram.com/v1/listen"`

	// Timeout for API calls
	Timeout time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
}

// DefaultDeepgramConfig returns a default Deepgram configuration
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
