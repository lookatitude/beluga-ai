package speculative

import (
	"sync/atomic"
	"time"
)

// Metrics tracks speculative execution statistics using atomic operations
// for thread safety.
type Metrics struct {
	predictions  atomic.Int64
	hits         atomic.Int64
	misses       atomic.Int64
	totalSpeedup atomic.Int64 // stored as nanoseconds
	wastedTokens atomic.Int64
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// RecordHit records a successful prediction.
func (m *Metrics) RecordHit(speedup time.Duration) {
	m.predictions.Add(1)
	m.hits.Add(1)
	m.totalSpeedup.Add(int64(speedup))
}

// RecordMiss records a failed prediction.
func (m *Metrics) RecordMiss(wastedTokens int) {
	m.predictions.Add(1)
	m.misses.Add(1)
	m.wastedTokens.Add(int64(wastedTokens))
}

// Predictions returns the total number of predictions attempted.
func (m *Metrics) Predictions() int64 {
	return m.predictions.Load()
}

// Hits returns the number of successful predictions.
func (m *Metrics) Hits() int64 {
	return m.hits.Load()
}

// Misses returns the number of failed predictions.
func (m *Metrics) Misses() int64 {
	return m.misses.Load()
}

// TotalSpeedup returns the cumulative time saved by successful predictions.
func (m *Metrics) TotalSpeedup() time.Duration {
	return time.Duration(m.totalSpeedup.Load())
}

// WastedTokens returns the total tokens wasted on failed predictions.
func (m *Metrics) WastedTokens() int64 {
	return m.wastedTokens.Load()
}

// HitRate returns the ratio of successful predictions to total predictions.
// Returns 0 if no predictions have been made.
func (m *Metrics) HitRate() float64 {
	total := m.predictions.Load()
	if total == 0 {
		return 0
	}
	return float64(m.hits.Load()) / float64(total)
}

// Snapshot returns a point-in-time copy of all metrics.
func (m *Metrics) Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		Predictions:  m.predictions.Load(),
		Hits:         m.hits.Load(),
		Misses:       m.misses.Load(),
		TotalSpeedup: time.Duration(m.totalSpeedup.Load()),
		WastedTokens: m.wastedTokens.Load(),
	}
}

// MetricsSnapshot is an immutable point-in-time copy of metrics.
type MetricsSnapshot struct {
	// Predictions is the total number of predictions attempted.
	Predictions int64
	// Hits is the number of successful predictions.
	Hits int64
	// Misses is the number of failed predictions.
	Misses int64
	// TotalSpeedup is the cumulative time saved.
	TotalSpeedup time.Duration
	// WastedTokens is the total tokens wasted on failed predictions.
	WastedTokens int64
}

// HitRate returns the ratio of hits to total predictions.
func (s MetricsSnapshot) HitRate() float64 {
	if s.Predictions == 0 {
		return 0
	}
	return float64(s.Hits) / float64(s.Predictions)
}
