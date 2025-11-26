package internal

import (
	"sync"
	"time"
)

// InterruptionConfig configures interruption detection.
type InterruptionConfig struct {
	WordCountThreshold int
	DurationThreshold  time.Duration
	Enabled            bool
}

// DefaultInterruptionConfig returns default interruption configuration.
func DefaultInterruptionConfig() *InterruptionConfig {
	return &InterruptionConfig{
		WordCountThreshold: 3,
		DurationThreshold:  500 * time.Millisecond,
		Enabled:            true,
	}
}

// InterruptionDetector detects user interruptions.
type InterruptionDetector struct {
	config   *InterruptionConfig
	mu       sync.RWMutex
	detected bool
}

// NewInterruptionDetector creates a new interruption detector.
func NewInterruptionDetector(config *InterruptionConfig) *InterruptionDetector {
	if config == nil {
		config = DefaultInterruptionConfig()
	}

	return &InterruptionDetector{
		config:   config,
		detected: false,
	}
}

// CheckInterruption checks if an interruption should be triggered based on word count and duration.
func (id *InterruptionDetector) CheckInterruption(wordCount int, duration time.Duration) bool {
	if !id.config.Enabled {
		return false
	}

	id.mu.Lock()
	defer id.mu.Unlock()

	// Check if thresholds are met
	wordThresholdMet := wordCount >= id.config.WordCountThreshold
	durationThresholdMet := duration >= id.config.DurationThreshold

	// Interruption detected if both thresholds are met
	id.detected = wordThresholdMet && durationThresholdMet
	return id.detected
}

// IsInterrupted returns whether an interruption is currently detected.
func (id *InterruptionDetector) IsInterrupted() bool {
	id.mu.RLock()
	defer id.mu.RUnlock()
	return id.detected
}

// Reset resets the interruption detection state.
func (id *InterruptionDetector) Reset() {
	id.mu.Lock()
	defer id.mu.Unlock()
	id.detected = false
}
