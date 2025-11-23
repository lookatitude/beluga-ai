package iface

import (
	"context"
)

// VADProvider defines the interface for voice activity detection providers.
// Implementations of this interface will provide access to different
// VAD algorithms (e.g., Silero, Energy-based, WebRTC).
//
// VADProvider follows the Interface Segregation Principle (ISP) by providing
// focused methods specific to voice activity detection operations.
type VADProvider interface {
	// Process analyzes audio data to detect voice activity.
	// It takes a context and audio data and returns true if voice activity is detected,
	// false otherwise, or an error if the process fails.
	Process(ctx context.Context, audio []byte) (bool, error)

	// ProcessStream processes a stream of audio data for voice activity detection.
	// It takes a context and a channel of audio chunks and returns a channel of
	// voice activity detection results.
	ProcessStream(ctx context.Context, audioCh <-chan []byte) (<-chan VADResult, error)
}

// VADResult represents a voice activity detection result.
type VADResult struct {
	HasVoice   bool
	Confidence float64
	Error      error
}
