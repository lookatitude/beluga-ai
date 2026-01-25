package internal

import (
	"time"
)

// AudioInput represents audio input data for S2S processing.
type AudioInput struct {
	Timestamp time.Time
	Format    AudioFormat
	Language  string
	Data      []byte
	Quality   AudioQuality
}

// AudioOutput represents audio output data from S2S processing.
type AudioOutput struct {
	// Audio data (PCM, WAV, etc.)
	Data []byte

	// Audio format information
	Format AudioFormat

	// Metadata
	Timestamp            time.Time
	Provider             string
	VoiceCharacteristics VoiceCharacteristics
	Latency              time.Duration
}

// AudioFormat represents audio format information.
type AudioFormat struct {
	Encoding   string
	SampleRate int
	Channels   int
	BitDepth   int
}

// AudioQuality represents audio quality metadata.
type AudioQuality struct {
	// Signal-to-noise ratio (dB)
	SNR float64

	// Whether the audio is clear
	IsClear bool

	// Background noise level (0.0 to 1.0)
	NoiseLevel float64
}

// VoiceCharacteristics represents voice characteristics in the output.
type VoiceCharacteristics struct {
	// Voice ID or name
	VoiceID string

	// Language code (e.g., "en-US")
	Language string

	// Gender (optional)
	Gender string

	// Speaking rate (optional)
	SpeakingRate float64
}

// ConversationContext represents conversation context for S2S processing.
type ConversationContext struct {
	Preferences    map[string]any
	AgentState     map[string]any
	ConversationID string
	SessionID      string
	UserID         string
	History        []ConversationTurn
}

// ConversationTurn represents a single turn in the conversation.
type ConversationTurn struct {
	Timestamp time.Time
	TurnID    string
	Role      string
	Content   string
}
