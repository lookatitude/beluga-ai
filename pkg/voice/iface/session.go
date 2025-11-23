package iface

import (
	"context"
)

// VoiceSession defines the interface for voice interaction sessions.
// It manages the complete lifecycle of a voice interaction between a user and an AI agent.
//
// VoiceSession follows the Interface Segregation Principle (ISP) by providing
// focused methods specific to voice session management.
type VoiceSession interface {
	// Start starts the voice session.
	// It takes a context and returns an error if the session cannot be started.
	Start(ctx context.Context) error

	// Stop stops the voice session gracefully.
	// It takes a context and returns an error if the session cannot be stopped.
	Stop(ctx context.Context) error

	// Say converts text to speech and plays it.
	// It takes a context and text and returns a SayHandle for controlling playback,
	// or an error if the operation fails.
	Say(ctx context.Context, text string) (SayHandle, error)

	// SayWithOptions converts text to speech with options and plays it.
	// It takes a context, text, and options and returns a SayHandle for controlling playback,
	// or an error if the operation fails.
	SayWithOptions(ctx context.Context, text string, options SayOptions) (SayHandle, error)

	// ProcessAudio processes incoming audio data.
	// It takes a context and audio data and returns an error if processing fails.
	ProcessAudio(ctx context.Context, audio []byte) error

	// GetState returns the current session state.
	GetState() SessionState

	// OnStateChanged sets a callback function that is called when the session state changes.
	OnStateChanged(callback func(SessionState))

	// GetSessionID returns the session identifier.
	GetSessionID() string
}

// SessionState represents the state of a voice session.
type SessionState string

const (
	SessionStateInitial    SessionState = "initial"
	SessionStateListening  SessionState = "listening"
	SessionStateProcessing SessionState = "processing"
	SessionStateSpeaking   SessionState = "speaking"
	SessionStateAway       SessionState = "away"
	SessionStateEnded      SessionState = "ended"
)

// SayOptions represents options for the Say operation.
type SayOptions struct {
	AllowInterruptions bool
	Voice              string
	Speed              float64 // 0.5-2.0, default: 1.0
	Volume             float64 // 0.0-1.0, default: 1.0
}

// SayHandle provides control over a Say operation.
type SayHandle interface {
	// WaitForPlayout waits for audio to finish playing.
	WaitForPlayout(ctx context.Context) error

	// Cancel cancels the Say operation.
	Cancel() error
}
