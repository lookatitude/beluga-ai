package internal

import (
	"sync"
	"sync/atomic"
	"time"
)

// SessionTimeout manages automatic session timeout on inactivity.
type SessionTimeout struct {
	lastActivity time.Time
	timer        *time.Timer
	onTimeout    func()
	timeout      time.Duration
	mu           sync.RWMutex
	active       bool
	timerID      int64 // Used to track timer generations to prevent race conditions
}

// NewSessionTimeout creates a new session timeout manager.
func NewSessionTimeout(timeout time.Duration, onTimeout func()) *SessionTimeout {
	return &SessionTimeout{
		timeout:      timeout,
		lastActivity: time.Now(),
		onTimeout:    onTimeout,
		active:       false,
	}
}

// Start starts the timeout monitoring.
func (st *SessionTimeout) Start() {
	st.mu.Lock()
	defer st.mu.Unlock()

	if st.active {
		return
	}

	st.active = true
	st.lastActivity = time.Now()
	st.resetTimer()
}

// Stop stops the timeout monitoring.
func (st *SessionTimeout) Stop() {
	st.mu.Lock()
	defer st.mu.Unlock()

	if !st.active {
		return
	}

	st.active = false
	// Invalidate any pending timers FIRST by incrementing the ID
	// This ensures any callbacks that are about to execute will see the new ID
	atomic.AddInt64(&st.timerID, 1)
	if st.timer != nil {
		st.timer.Stop()
		st.timer = nil
	}
}

// UpdateActivity updates the last activity time and resets the timer.
func (st *SessionTimeout) UpdateActivity() {
	st.mu.Lock()
	defer st.mu.Unlock()

	if !st.active {
		return
	}

	st.lastActivity = time.Now()
	st.resetTimer()
}

// resetTimer resets the timeout timer.
// This function must be called with st.mu already locked.
func (st *SessionTimeout) resetTimer() {
	// CRITICAL: Increment timer ID FIRST, before stopping the old timer.
	// This ensures that any callback from the old timer that's about to execute
	// will see the new timerID and return early.
	// Use atomic.StoreInt64 to ensure memory visibility across goroutines
	newID := atomic.AddInt64(&st.timerID, 1)
	// Use atomic.LoadInt64 to ensure we read the latest value (memory barrier)
	currentID := atomic.LoadInt64(&st.timerID)
	_ = newID // Ensure the increment happens

	if st.timer != nil {
		// Stop the old timer - this prevents it from firing if it hasn't already.
		// If it has already fired, the callback will check timerID and return.
		st.timer.Stop()
	}

	// Create new timer with the current ID
	st.timer = time.AfterFunc(st.timeout, func() {
		// Check 1: Fast path - is this timer still valid? (atomic read, no lock)
		// Use LoadInt64 which provides acquire semantics (memory barrier)
		if atomic.LoadInt64(&st.timerID) != currentID {
			return // This timer was replaced, don't execute
		}

		// Check 2: Acquire lock to check state atomically
		// The lock acquisition also provides a memory barrier
		st.mu.RLock()
		onTimeout := st.onTimeout
		active := st.active
		// Re-check timer ID while holding lock (double-check pattern)
		// This ensures we see the latest timerID value
		timerStillValid := atomic.LoadInt64(&st.timerID) == currentID
		st.mu.RUnlock()

		// Check 3: Final validation - timer must still be valid and active
		if !timerStillValid || !active {
			return // Timer was replaced or session is no longer active
		}

		// All checks passed, safe to execute callback
		if onTimeout != nil {
			onTimeout()
		}
	})
}

// GetLastActivity returns the last activity time.
func (st *SessionTimeout) GetLastActivity() time.Time {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.lastActivity
}
