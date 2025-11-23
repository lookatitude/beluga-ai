package internal

import (
	"sync"
	"time"
)

// WordCountDetector detects interruptions based on word count
type WordCountDetector struct {
	mu           sync.RWMutex
	threshold    int
	currentCount int
}

// NewWordCountDetector creates a new word count detector
func NewWordCountDetector(threshold int) *WordCountDetector {
	return &WordCountDetector{
		threshold:    threshold,
		currentCount: 0,
	}
}

// AddWords adds words to the count
func (wcd *WordCountDetector) AddWords(count int) {
	wcd.mu.Lock()
	defer wcd.mu.Unlock()
	wcd.currentCount += count
}

// CheckThreshold checks if the word count threshold is met
func (wcd *WordCountDetector) CheckThreshold() bool {
	wcd.mu.RLock()
	defer wcd.mu.RUnlock()
	return wcd.currentCount >= wcd.threshold
}

// Reset resets the word count
func (wcd *WordCountDetector) Reset() {
	wcd.mu.Lock()
	defer wcd.mu.Unlock()
	wcd.currentCount = 0
}

// DurationDetector detects interruptions based on duration
type DurationDetector struct {
	mu        sync.RWMutex
	threshold time.Duration
	startTime time.Time
	active    bool
}

// NewDurationDetector creates a new duration detector
func NewDurationDetector(threshold time.Duration) *DurationDetector {
	return &DurationDetector{
		threshold: threshold,
		active:    false,
	}
}

// Start starts the duration measurement
func (dd *DurationDetector) Start() {
	dd.mu.Lock()
	defer dd.mu.Unlock()
	dd.startTime = time.Now()
	dd.active = true
}

// Stop stops the duration measurement
func (dd *DurationDetector) Stop() {
	dd.mu.Lock()
	defer dd.mu.Unlock()
	dd.active = false
}

// CheckThreshold checks if the duration threshold is met
func (dd *DurationDetector) CheckThreshold() bool {
	dd.mu.RLock()
	defer dd.mu.RUnlock()

	if !dd.active {
		return false
	}

	elapsed := time.Since(dd.startTime)
	return elapsed >= dd.threshold
}

// GetElapsed returns the elapsed duration
func (dd *DurationDetector) GetElapsed() time.Duration {
	dd.mu.RLock()
	defer dd.mu.RUnlock()

	if !dd.active {
		return 0
	}

	return time.Since(dd.startTime)
}
