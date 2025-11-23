package internal

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// TurnDetectionIntegration manages turn detector integration
type TurnDetectionIntegration struct {
	detector iface.TurnDetector
	mu       sync.RWMutex
}

// NewTurnDetectionIntegration creates a new turn detection integration
func NewTurnDetectionIntegration(detector iface.TurnDetector) *TurnDetectionIntegration {
	return &TurnDetectionIntegration{
		detector: detector,
	}
}

// DetectTurn detects a turn in audio using the turn detector
func (tdi *TurnDetectionIntegration) DetectTurn(ctx context.Context, audio []byte) (bool, error) {
	tdi.mu.RLock()
	detector := tdi.detector
	tdi.mu.RUnlock()

	if detector == nil {
		return false, fmt.Errorf("turn detector not set")
	}

	return detector.DetectTurn(ctx, audio)
}

// DetectTurnStream detects turns in a stream of audio
// Note: TurnDetector doesn't have streaming - process chunks individually
func (tdi *TurnDetectionIntegration) DetectTurnStream(ctx context.Context, audioCh <-chan []byte) (<-chan bool, error) {
	tdi.mu.RLock()
	detector := tdi.detector
	tdi.mu.RUnlock()

	if detector == nil {
		return nil, fmt.Errorf("turn detector not set")
	}

	resultCh := make(chan bool, 10)
	go func() {
		defer close(resultCh)
		for audio := range audioCh {
			detected, err := detector.DetectTurn(ctx, audio)
			if err != nil {
				return
			}
			select {
			case <-ctx.Done():
				return
			case resultCh <- detected:
			}
		}
	}()
	return resultCh, nil
}
