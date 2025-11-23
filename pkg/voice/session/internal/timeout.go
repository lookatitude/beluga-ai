package internal

import (
	"sync"
	"time"
)

// SessionTimeout manages automatic session timeout on inactivity
type SessionTimeout struct {
	mu           sync.RWMutex
	timeout      time.Duration
	lastActivity time.Time
	timer        *time.Timer
	onTimeout    func()
	active       bool
}

// NewSessionTimeout creates a new session timeout manager
func NewSessionTimeout(timeout time.Duration, onTimeout func()) *SessionTimeout {
	return &SessionTimeout{
		timeout:      timeout,
		lastActivity: time.Now(),
		onTimeout:    onTimeout,
		active:       false,
	}
}

// Start starts the timeout monitoring
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

// Stop stops the timeout monitoring
func (st *SessionTimeout) Stop() {
	st.mu.Lock()
	defer st.mu.Unlock()

	if !st.active {
		return
	}

	st.active = false
	if st.timer != nil {
		st.timer.Stop()
		st.timer = nil
	}
}

// UpdateActivity updates the last activity time and resets the timer
func (st *SessionTimeout) UpdateActivity() {
	st.mu.Lock()
	defer st.mu.Unlock()

	if !st.active {
		return
	}

	st.lastActivity = time.Now()
	st.resetTimer()
}

// resetTimer resets the timeout timer
func (st *SessionTimeout) resetTimer() {
	if st.timer != nil {
		st.timer.Stop()
	}

	st.timer = time.AfterFunc(st.timeout, func() {
		st.mu.RLock()
		onTimeout := st.onTimeout
		active := st.active
		st.mu.RUnlock()

		if active && onTimeout != nil {
			onTimeout()
		}
	})
}

// GetLastActivity returns the last activity time
func (st *SessionTimeout) GetLastActivity() time.Time {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.lastActivity
}
