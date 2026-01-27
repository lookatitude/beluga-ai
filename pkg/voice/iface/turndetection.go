// Package iface defines interfaces for the voice package.
//
// Deprecated: This package has been moved to pkg/voiceutils/iface.
// Please update your imports to use github.com/lookatitude/beluga-ai/pkg/voiceutils/iface.
// This package will be removed in v2.0.
package iface

import (
	"context"
	"time"
)

// TurnDetector defines the interface for turn detection providers.
// Implementations of this interface will provide access to different
// turn detection algorithms (e.g., ONNX-based, Heuristic-based).
//
// TurnDetector follows the Interface Segregation Principle (ISP) by providing
// focused methods specific to turn detection operations.
type TurnDetector interface {
	// DetectTurn analyzes audio data to detect the end of a user's turn.
	// It takes a context and audio data and returns true if a turn end is detected,
	// false otherwise, or an error if the process fails.
	DetectTurn(ctx context.Context, audio []byte) (bool, error)

	// DetectTurnWithSilence analyzes audio data and silence duration to detect turn end.
	// It takes a context, audio data, and silence duration and returns true if a turn end
	// is detected, false otherwise, or an error if the process fails.
	DetectTurnWithSilence(ctx context.Context, audio []byte, silenceDuration time.Duration) (bool, error)
}
