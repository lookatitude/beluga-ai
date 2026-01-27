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
