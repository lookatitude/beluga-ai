package teambuilder

import (
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
)

// PoolEntry associates an agent with its declared capabilities and
// runtime performance metrics.
type PoolEntry struct {
	// Agent is the registered agent.
	Agent agent.Agent
	// Capabilities lists the skills/domains this agent can handle.
	Capabilities []string
	// Metrics tracks runtime performance for this agent.
	Metrics *AgentMetrics
}

// AgentMetrics tracks runtime performance statistics for an agent.
// All fields are updated atomically or under a mutex by the pool.
type AgentMetrics struct {
	mu           sync.RWMutex
	invocations  int
	successes    int
	failures     int
	totalLatency time.Duration
	lastUsed     time.Time
}

// NewAgentMetrics creates a zero-valued AgentMetrics.
func NewAgentMetrics() *AgentMetrics {
	return &AgentMetrics{}
}

// RecordSuccess records a successful invocation with the given latency.
func (m *AgentMetrics) RecordSuccess(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invocations++
	m.successes++
	m.totalLatency += latency
	m.lastUsed = time.Now()
}

// RecordFailure records a failed invocation with the given latency.
func (m *AgentMetrics) RecordFailure(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.invocations++
	m.failures++
	m.totalLatency += latency
	m.lastUsed = time.Now()
}

// Snapshot returns a point-in-time copy of the metrics.
func (m *AgentMetrics) Snapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	snap := MetricsSnapshot{
		Invocations:  m.invocations,
		Successes:    m.successes,
		Failures:     m.failures,
		TotalLatency: m.totalLatency,
		LastUsed:     m.lastUsed,
	}
	if m.invocations > 0 {
		snap.AvgLatency = m.totalLatency / time.Duration(m.invocations)
	}
	return snap
}

// MetricsSnapshot is an immutable point-in-time copy of AgentMetrics.
type MetricsSnapshot struct {
	Invocations  int
	Successes    int
	Failures     int
	TotalLatency time.Duration
	AvgLatency   time.Duration
	LastUsed     time.Time
}

// SuccessRate returns the fraction of successful invocations (0.0 to 1.0).
// Returns 0 if there have been no invocations.
func (s MetricsSnapshot) SuccessRate() float64 {
	if s.Invocations == 0 {
		return 0
	}
	return float64(s.Successes) / float64(s.Invocations)
}
