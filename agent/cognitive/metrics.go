package cognitive

import (
	"sync"
	"sync/atomic"
	"time"
)

// RoutingMetrics tracks aggregate statistics about dual-process routing
// decisions. It is safe for concurrent use.
type RoutingMetrics struct {
	s1Count         atomic.Int64
	s2Count         atomic.Int64
	escalationCount atomic.Int64

	mu             sync.Mutex
	s1TotalLatency time.Duration
	s2TotalLatency time.Duration
	s1TotalCost    float64
	s2TotalCost    float64
}

// S1Count returns the number of requests handled by System 1.
func (m *RoutingMetrics) S1Count() int64 { return m.s1Count.Load() }

// S2Count returns the number of requests handled by System 2.
func (m *RoutingMetrics) S2Count() int64 { return m.s2Count.Load() }

// EscalationCount returns the number of requests escalated from S1 to S2.
func (m *RoutingMetrics) EscalationCount() int64 { return m.escalationCount.Load() }

// S1TotalLatency returns the total latency for System 1 requests.
func (m *RoutingMetrics) S1TotalLatency() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.s1TotalLatency
}

// S2TotalLatency returns the total latency for System 2 requests.
func (m *RoutingMetrics) S2TotalLatency() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.s2TotalLatency
}

// RecordS1 records a System 1 execution with the given latency and cost.
func (m *RoutingMetrics) RecordS1(latency time.Duration, cost float64) {
	m.s1Count.Add(1)
	m.mu.Lock()
	m.s1TotalLatency += latency
	m.s1TotalCost += cost
	m.mu.Unlock()
}

// RecordS2 records a System 2 execution with the given latency and cost.
func (m *RoutingMetrics) RecordS2(latency time.Duration, cost float64) {
	m.s2Count.Add(1)
	m.mu.Lock()
	m.s2TotalLatency += latency
	m.s2TotalCost += cost
	m.mu.Unlock()
}

// RecordEscalation increments the escalation counter.
func (m *RoutingMetrics) RecordEscalation() {
	m.escalationCount.Add(1)
}

// EscalationRate returns the fraction of total requests that were escalated
// from S1 to S2. Returns 0 if no requests have been processed.
func (m *RoutingMetrics) EscalationRate() float64 {
	total := m.s1Count.Load() + m.s2Count.Load()
	if total == 0 {
		return 0
	}
	return float64(m.escalationCount.Load()) / float64(total)
}

// CostSavings estimates the cost saved by routing some requests through S1
// instead of sending all requests through S2. The hypotheticalS2CostPerRequest
// parameter is the average cost if every request went to S2.
func (m *RoutingMetrics) CostSavings(hypotheticalS2CostPerRequest float64) float64 {
	total := m.s1Count.Load() + m.s2Count.Load()
	if total == 0 {
		return 0
	}
	hypotheticalTotal := float64(total) * hypotheticalS2CostPerRequest
	m.mu.Lock()
	actualTotal := m.s1TotalCost + m.s2TotalCost
	m.mu.Unlock()
	if hypotheticalTotal == 0 {
		return 0
	}
	return hypotheticalTotal - actualTotal
}

// Snapshot returns a copy of the current metrics as a plain struct,
// suitable for serialization or logging.
func (m *RoutingMetrics) Snapshot() MetricsSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return MetricsSnapshot{
		S1Count:         m.s1Count.Load(),
		S2Count:         m.s2Count.Load(),
		EscalationCount: m.escalationCount.Load(),
		S1TotalLatency:  m.s1TotalLatency,
		S2TotalLatency:  m.s2TotalLatency,
		S1TotalCost:     m.s1TotalCost,
		S2TotalCost:     m.s2TotalCost,
	}
}

// MetricsSnapshot is a point-in-time copy of routing metrics.
type MetricsSnapshot struct {
	S1Count         int64
	S2Count         int64
	EscalationCount int64
	S1TotalLatency  time.Duration
	S2TotalLatency  time.Duration
	S1TotalCost     float64
	S2TotalCost     float64
}
