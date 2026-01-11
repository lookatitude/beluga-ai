package internal

import (
	"time"
)

// AudioInput represents audio input data for S2S processing.
type AudioInput struct {
	// Audio data (PCM, WAV, etc.)
	Data []byte

	// Audio format information
	Format AudioFormat

	// Metadata
	Timestamp time.Time
	Language  string
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
	// Sample rate in Hz (e.g., 16000, 24000, 48000)
	SampleRate int

	// Number of channels (1 = mono, 2 = stereo)
	Channels int

	// Bit depth (8, 16, 24, 32)
	BitDepth int

	// Encoding format (PCM, WAV, MP3, etc.)
	Encoding string
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
	// Conversation ID
	ConversationID string

	// Session ID
	SessionID string

	// User ID (optional)
	UserID string

	// Conversation history (optional)
	History []ConversationTurn

	// User preferences (optional)
	Preferences map[string]any

	// Agent state (optional, for external agent integration)
	AgentState map[string]any
}

// ConversationTurn represents a single turn in the conversation.
type ConversationTurn struct {
	// Turn ID
	TurnID string

	// Role (user, assistant, system)
	Role string

	// Content (transcript or audio reference)
	Content string

	// Timestamp
	Timestamp time.Time
}
