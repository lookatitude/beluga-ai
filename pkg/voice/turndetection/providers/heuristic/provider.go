package heuristic

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
	turndetectioniface "github.com/lookatitude/beluga-ai/pkg/voice/turndetection/iface"
)

// HeuristicProvider implements the TurnDetector interface for Heuristic-based Turn Detection
type HeuristicProvider struct {
	config *HeuristicConfig
	mu     sync.RWMutex
}

// NewHeuristicProvider creates a new Heuristic Turn Detection provider
func NewHeuristicProvider(config *turndetection.Config) (turndetectioniface.TurnDetector, error) {
	if config == nil {
		return nil, turndetection.NewTurnDetectionError("NewHeuristicProvider", turndetection.ErrCodeInvalidConfig,
			fmt.Errorf("config cannot be nil"))
	}

	// Convert base config to Heuristic config
	heuristicConfig := &HeuristicConfig{
		Config: config,
	}

	// Set defaults if not provided
	if heuristicConfig.SentenceEndMarkers == "" {
		heuristicConfig.SentenceEndMarkers = ".!?"
	}
	if len(heuristicConfig.QuestionMarkers) == 0 {
		heuristicConfig.QuestionMarkers = []string{"what", "where", "when", "why", "how", "who", "which"}
	}
	if heuristicConfig.MinSilenceDuration == 0 {
		heuristicConfig.MinSilenceDuration = 500 * time.Millisecond
	}
	if heuristicConfig.MinTurnLength == 0 {
		heuristicConfig.MinTurnLength = 10
	}
	if heuristicConfig.MaxTurnLength == 0 {
		heuristicConfig.MaxTurnLength = 5000
	}

	return &HeuristicProvider{
		config: heuristicConfig,
	}, nil
}

// DetectTurn implements the TurnDetector interface
// Note: Heuristic provider works with transcripts, but the interface uses audio
// For now, we'll use a simple placeholder that always returns false
// In a real implementation, this would need transcript data from STT
func (p *HeuristicProvider) DetectTurn(ctx context.Context, audio []byte) (bool, error) {
	// Heuristic detection typically works with transcripts, not raw audio
	// For now, return false (no turn detected) as a placeholder
	// In a real implementation, this would:
	// 1. Get transcript from STT (would need integration)
	// 2. Apply heuristic rules
	// 3. Return turn detection result

	// Placeholder: Simple check based on audio length
	// This is not ideal but maintains interface compliance
	if len(audio) == 0 {
		return false, nil
	}

	// For heuristic detection, we typically need transcript data
	// This is a limitation of the current interface design
	// In practice, turn detection would be called with transcript data
	return false, nil
}

// DetectTurnWithSilence implements the TurnDetector interface
func (p *HeuristicProvider) DetectTurnWithSilence(ctx context.Context, audio []byte, silenceDuration time.Duration) (bool, error) {
	// Check if silence duration exceeds minimum threshold
	if silenceDuration >= p.config.MinSilenceDuration {
		return true, nil
	}

	// Otherwise, use regular detection
	return p.DetectTurn(ctx, audio)
}

// detectTurnFromTranscript performs heuristic turn detection on a transcript
// This is a helper method that would be used when transcript data is available
func (p *HeuristicProvider) detectTurnFromTranscript(transcript string, isFinal bool) bool {
	// Check minimum turn length
	if len(transcript) < p.config.MinTurnLength {
		return false
	}

	// Check maximum turn length
	if len(transcript) > p.config.MaxTurnLength {
		return true // Turn is too long, likely complete
	}

	// Check for sentence ending markers
	trimmed := strings.TrimSpace(transcript)
	if len(trimmed) > 0 {
		lastChar := trimmed[len(trimmed)-1]
		if strings.ContainsRune(p.config.SentenceEndMarkers, rune(lastChar)) {
			return true
		}
	}

	// Check for question markers (case-insensitive)
	lowerTranscript := strings.ToLower(transcript)
	for _, marker := range p.config.QuestionMarkers {
		if strings.Contains(lowerTranscript, marker+"?") {
			return true
		}
	}

	// If transcript is final and has reasonable length, consider it a turn
	if isFinal && len(transcript) >= p.config.MinTurnLength {
		return true
	}

	return false
}
