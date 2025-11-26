// Package iface defines the core interfaces for the voice package.
// It follows the Interface Segregation Principle by providing small, focused interfaces
// that serve specific purposes within the voice system.
package iface

import (
	"context"
)

// STTProvider defines the interface for speech-to-text providers.
// Implementations of this interface will provide access to different
// STT services (e.g., Deepgram, Google, Azure, OpenAI).
//
// STTProvider follows the Interface Segregation Principle (ISP) by providing
// focused methods specific to speech-to-text operations.
type STTProvider interface {
	// Transcribe converts audio data to text.
	// It takes a context for cancellation and deadline propagation, and audio data
	// and returns the transcribed text or an error if the process fails.
	Transcribe(ctx context.Context, audio []byte) (string, error)

	// StartStreaming begins a streaming transcription session.
	// It takes a context and returns a StreamingSession for real-time transcription.
	StartStreaming(ctx context.Context) (StreamingSession, error)
}

// StreamingSession defines the interface for streaming transcription sessions.
type StreamingSession interface {
	// SendAudio sends audio data to the streaming session.
	SendAudio(ctx context.Context, audio []byte) error

	// ReceiveTranscript receives transcribed text from the streaming session.
	// Returns a channel that receives transcript results.
	ReceiveTranscript() <-chan TranscriptResult

	// Close closes the streaming session.
	Close() error
}

// TranscriptResult represents a transcription result from a streaming session.
type TranscriptResult struct {
	Error      error
	Text       string
	Language   string
	Confidence float64
	IsFinal    bool
}
