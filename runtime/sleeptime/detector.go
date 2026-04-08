package sleeptime

import (
	"sync"
	"time"
)

// IdleDetector determines whether a session is currently idle (no active user
// interaction). Implementations must be safe for concurrent use.
type IdleDetector interface {
	// IsIdle reports whether the session is currently idle.
	IsIdle() bool

	// OnActivity records that user activity has occurred, resetting the
	// inactivity timer.
	OnActivity()
}

// InactivityDetector is an IdleDetector that considers a session idle after
// a configurable period of inactivity. Each call to OnActivity resets the
// inactivity timer.
type InactivityDetector struct {
	mu       sync.Mutex
	timeout  time.Duration
	lastSeen time.Time
	nowFn    func() time.Time // for testing
}

// Compile-time check.
var _ IdleDetector = (*InactivityDetector)(nil)

// NewInactivityDetector creates an InactivityDetector that considers the
// session idle after the given timeout has elapsed since the last activity.
// The minimum timeout is 1 second; smaller values are clamped.
func NewInactivityDetector(timeout time.Duration) *InactivityDetector {
	if timeout < time.Second {
		timeout = time.Second
	}
	return &InactivityDetector{
		timeout:  timeout,
		lastSeen: time.Now(),
		nowFn:    time.Now,
	}
}

// IsIdle reports whether the configured inactivity timeout has elapsed since
// the last recorded activity.
func (d *InactivityDetector) IsIdle() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.nowFn().Sub(d.lastSeen) >= d.timeout
}

// OnActivity records a user activity event, resetting the inactivity timer.
func (d *InactivityDetector) OnActivity() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastSeen = d.nowFn()
}
