package degradation

import (
	"context"
	"sync"
	"time"
)

// SecurityEventType identifies the kind of security event recorded.
type SecurityEventType string

const (
	// EventGuardBlocked indicates a guard rejected content.
	EventGuardBlocked SecurityEventType = "guard_blocked"

	// EventInjectionDetected indicates a prompt injection attempt was detected.
	EventInjectionDetected SecurityEventType = "injection_detected"

	// EventPIILeakage indicates PII was found in output content.
	EventPIILeakage SecurityEventType = "pii_leakage"

	// EventToolAbuse indicates suspicious tool usage patterns.
	EventToolAbuse SecurityEventType = "tool_abuse"

	// EventRateLimitHit indicates excessive request volume from an agent.
	EventRateLimitHit SecurityEventType = "rate_limit_hit"

	// EventCustom indicates a user-defined security event type.
	EventCustom SecurityEventType = "custom"
)

// SecurityEvent represents a single security-relevant occurrence recorded by
// the monitor.
type SecurityEvent struct {
	// Type identifies the kind of security event.
	Type SecurityEventType

	// Severity is the weight of this event, in the range [0.0, 1.0].
	// Higher values indicate more serious anomalies.
	Severity float64

	// Timestamp is when the event occurred.
	Timestamp time.Time

	// Source identifies which component generated the event (e.g., guard name).
	Source string

	// Metadata carries event-specific key-value pairs.
	Metadata map[string]any
}

// defaultWindowSize is the default sliding window for severity computation.
const defaultWindowSize = 5 * time.Minute

// SecurityMonitor tracks security anomaly signals within a sliding time
// window and computes an aggregate severity score. It is safe for concurrent
// use.
type SecurityMonitor struct {
	mu         sync.RWMutex
	events     []SecurityEvent
	windowSize time.Duration
	nowFunc    func() time.Time // for testing
}

// monitorOptions holds configuration for SecurityMonitor.
type monitorOptions struct {
	windowSize time.Duration
	nowFunc    func() time.Time
}

// MonitorOption configures a SecurityMonitor.
type MonitorOption func(*monitorOptions)

// WithWindowSize sets the sliding window duration for severity computation.
// Events older than the window are discarded. Defaults to 5 minutes.
func WithWindowSize(d time.Duration) MonitorOption {
	return func(o *monitorOptions) {
		if d > 0 {
			o.windowSize = d
		}
	}
}

// withNowFunc overrides the time source for testing. This is unexported
// because it is only useful in tests.
func withNowFunc(fn func() time.Time) MonitorOption {
	return func(o *monitorOptions) {
		o.nowFunc = fn
	}
}

// NewSecurityMonitor creates a SecurityMonitor with the given options.
func NewSecurityMonitor(opts ...MonitorOption) *SecurityMonitor {
	o := monitorOptions{
		windowSize: defaultWindowSize,
		nowFunc:    time.Now,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &SecurityMonitor{
		windowSize: o.windowSize,
		nowFunc:    o.nowFunc,
	}
}

// RecordEvent adds a security event to the monitor. The event's Severity is
// clamped to [0.0, 1.0]. Context is accepted for future tracing integration.
func (m *SecurityMonitor) RecordEvent(_ context.Context, event SecurityEvent) {
	if event.Severity < 0 {
		event.Severity = 0
	}
	if event.Severity > 1 {
		event.Severity = 1
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = m.nowFunc()
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	m.pruneExpiredLocked()
}

// CurrentSeverity returns the aggregate severity score across all events in
// the current window. The score is the sum of individual event severities,
// with newer events weighted more heavily via linear decay. The result is
// clamped to [0.0, 1.0].
func (m *SecurityMonitor) CurrentSeverity() float64 {
	m.mu.Lock()
	m.pruneExpiredLocked()
	m.mu.Unlock()

	m.mu.RLock()
	defer m.mu.RUnlock()

	now := m.nowFunc()
	cutoff := now.Add(-m.windowSize)
	var total float64

	for _, e := range m.events {
		if e.Timestamp.Before(cutoff) {
			continue
		}
		age := now.Sub(e.Timestamp)
		// Linear decay: newer events have weight closer to 1.0.
		weight := 1.0 - (float64(age) / float64(m.windowSize))
		if weight < 0 {
			weight = 0
		}
		total += e.Severity * weight
	}

	if total > 1.0 {
		total = 1.0
	}
	return total
}

// EventCount returns the number of events currently in the window. Expired
// events are pruned on read so a burst of events followed by a quiet period
// does not retain unbounded memory.
func (m *SecurityMonitor) EventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneExpiredLocked()
	return len(m.events)
}

// Reset clears all recorded events.
func (m *SecurityMonitor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = nil
}

// pruneExpiredLocked removes events older than the window. Caller must hold
// m.mu for writing.
func (m *SecurityMonitor) pruneExpiredLocked() {
	cutoff := m.nowFunc().Add(-m.windowSize)
	n := 0
	for _, e := range m.events {
		if !e.Timestamp.Before(cutoff) {
			m.events[n] = e
			n++
		}
	}
	// Clear references to allow GC of metadata maps.
	for i := n; i < len(m.events); i++ {
		m.events[i] = SecurityEvent{}
	}
	m.events = m.events[:n]
}
