// Package iface defines interfaces for the voice package.
//
// Deprecated: This package has been moved to pkg/voiceutils/iface.
// Please update your imports to use github.com/lookatitude/beluga-ai/pkg/voiceutils/iface.
// This package will be removed in v2.0.
package iface

import (
	"context"
)

// NoiseCancellation defines the interface for noise cancellation providers.
// Implementations of this interface will provide access to different
// noise cancellation algorithms (e.g., Spectral Subtraction, RNNoise, WebRTC).
//
// NoiseCancellation follows the Interface Segregation Principle (ISP) by providing
// focused methods specific to noise cancellation operations.
type NoiseCancellation interface {
	// Process removes noise from audio data.
	// It takes a context and audio data and returns cleaned audio data
	// or an error if the process fails.
	Process(ctx context.Context, audio []byte) ([]byte, error)

	// ProcessStream processes a stream of audio data for noise cancellation.
	// It takes a context and a channel of audio chunks and returns a channel of
	// cleaned audio chunks.
	ProcessStream(ctx context.Context, audioCh <-chan []byte) (<-chan []byte, error)
}
