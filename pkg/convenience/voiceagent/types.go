package voiceagent

import (
	"context"

	sttiface "github.com/lookatitude/beluga-ai/pkg/stt/iface"
	ttsiface "github.com/lookatitude/beluga-ai/pkg/tts/iface"
	vadiface "github.com/lookatitude/beluga-ai/pkg/vad/iface"
)

// VoiceAgent defines the simplified interface for voice-enabled agents.
// It provides a streamlined API for creating voice conversations.
type VoiceAgent interface {
	// StartSession creates and starts a new voice session.
	// The session handles the full conversation lifecycle.
	StartSession(ctx context.Context) (Session, error)

	// ProcessAudio transcribes audio, processes it with the agent, and generates audio response.
	// This is a convenience method for one-shot audio processing.
	ProcessAudio(ctx context.Context, audio []byte) ([]byte, error)

	// ProcessText processes text input and generates audio response.
	// This bypasses STT and goes directly to the agent.
	ProcessText(ctx context.Context, text string) (string, error)

	// GetSTT returns the STT provider if configured.
	GetSTT() sttiface.STTProvider

	// GetTTS returns the TTS provider if configured.
	GetTTS() ttsiface.TTSProvider

	// GetVAD returns the VAD provider if configured.
	GetVAD() vadiface.VADProvider

	// Shutdown gracefully stops the voice agent and releases resources.
	Shutdown() error
}

// Session defines the interface for a voice conversation session.
// A session maintains state across multiple audio interactions.
type Session interface {
	// ID returns the unique session identifier.
	ID() string

	// Start begins the session and starts processing audio.
	Start(ctx context.Context) error

	// Stop ends the session and releases resources.
	Stop() error

	// SendAudio sends audio data to the session for processing.
	SendAudio(ctx context.Context, audio []byte) error

	// ReceiveAudio returns a channel for receiving synthesized audio responses.
	ReceiveAudio() <-chan []byte

	// GetTranscript returns the accumulated transcript of the session.
	GetTranscript() string

	// IsActive returns whether the session is currently active.
	IsActive() bool
}

// TranscriptCallback is called when a transcript is received.
type TranscriptCallback func(text string, isFinal bool)

// ResponseCallback is called when a response is generated.
type ResponseCallback func(text string)

// ErrorCallback is called when an error occurs.
type ErrorCallback func(error)
