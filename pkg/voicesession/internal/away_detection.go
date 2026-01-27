package internal

import (
	"sync"
	"time"
)

// AwayDetection manages user away state detection.
type AwayDetection struct {
	lastActivity      time.Time
	onAwayStateChange func(isAway bool)
	awayThreshold     time.Duration
	mu                sync.RWMutex
	isAway            bool
}

// NewAwayDetection creates a new away detection manager.
func NewAwayDetection(awayThreshold time.Duration, onAwayStateChange func(isAway bool)) *AwayDetection {
	return &AwayDetection{
		awayThreshold:     awayThreshold,
		lastActivity:      time.Now(),
		isAway:            false,
		onAwayStateChange: onAwayStateChange,
	}
}

// UpdateActivity updates the last activity time and checks away status.
func (ad *AwayDetection) UpdateActivity() {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	ad.lastActivity = time.Now()

	// Check if we should transition from away to active
	if ad.isAway {
		ad.isAway = false
		if ad.onAwayStateChange != nil {
			ad.onAwayStateChange(false)
		}
	}
}

// CheckAwayStatus checks if the user should be marked as away.
func (ad *AwayDetection) CheckAwayStatus() bool {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	timeSinceActivity := time.Since(ad.lastActivity)
	shouldBeAway := timeSinceActivity >= ad.awayThreshold

	if shouldBeAway && !ad.isAway {
		ad.isAway = true
		if ad.onAwayStateChange != nil {
			ad.onAwayStateChange(true)
		}
	}

	return ad.isAway
}

// IsAway returns whether the user is currently away.
func (ad *AwayDetection) IsAway() bool {
	ad.mu.RLock()
	defer ad.mu.RUnlock()
	return ad.isAway
}

// StartMonitoring starts periodic away status checking.
func (ad *AwayDetection) StartMonitoring(checkInterval time.Duration) {
	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		for range ticker.C {
			ad.CheckAwayStatus()
		}
	}()
}
