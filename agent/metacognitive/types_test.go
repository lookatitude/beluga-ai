package metacognitive

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSelfModel(t *testing.T) {
	m := NewSelfModel("agent-1")
	require.NotNil(t, m)
	assert.Equal(t, "agent-1", m.AgentID)
	assert.Empty(t, m.Heuristics)
	assert.NotNil(t, m.Capabilities)
	assert.False(t, m.UpdatedAt.IsZero())
}

func TestHeuristic_Fields(t *testing.T) {
	h := Heuristic{
		ID:         "h_123",
		Content:    "Avoid: excessive retries",
		Source:     "failure",
		TaskType:   "search",
		Utility:    0.8,
		UsageCount: 3,
		CreatedAt:  time.Now(),
	}
	assert.Equal(t, "h_123", h.ID)
	assert.Equal(t, "failure", h.Source)
	assert.Equal(t, 0.8, h.Utility)
	assert.Equal(t, 3, h.UsageCount)
}

func TestCapabilityScore_Fields(t *testing.T) {
	cs := CapabilityScore{
		TaskType:    "coding",
		SuccessRate: 0.85,
		SampleCount: 20,
		AvgLatency:  2 * time.Second,
		LastUpdated: time.Now(),
	}
	assert.Equal(t, "coding", cs.TaskType)
	assert.InDelta(t, 0.85, cs.SuccessRate, 0.001)
	assert.Equal(t, 20, cs.SampleCount)
	assert.Equal(t, 2*time.Second, cs.AvgLatency)
}

func TestMonitoringSignals_Zero(t *testing.T) {
	var s MonitoringSignals
	assert.Empty(t, s.TaskInput)
	assert.False(t, s.Success)
	assert.Nil(t, s.ToolCalls)
	assert.Nil(t, s.Errors)
}

func TestHooks_NilFields(t *testing.T) {
	// All fields should be nil by default.
	var h Hooks
	assert.Nil(t, h.OnHeuristicExtracted)
	assert.Nil(t, h.OnCapabilityUpdated)
	assert.Nil(t, h.OnSelfModelLoaded)
}
