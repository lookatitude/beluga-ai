package metacognitive

import (
	"time"
)

// SelfModel holds persistent self-knowledge for an agent, including
// accumulated heuristics and capability scores across task types.
type SelfModel struct {
	// AgentID identifies the agent this model belongs to.
	AgentID string

	// Heuristics are learned rules derived from execution experience.
	Heuristics []Heuristic

	// Capabilities maps task types to their performance scores.
	Capabilities map[string]*CapabilityScore

	// UpdatedAt is the timestamp of the last model update.
	UpdatedAt time.Time
}

// NewSelfModel creates a new SelfModel for the given agent ID.
func NewSelfModel(agentID string) *SelfModel {
	return &SelfModel{
		AgentID:      agentID,
		Heuristics:   nil,
		Capabilities: make(map[string]*CapabilityScore),
		UpdatedAt:    time.Now(),
	}
}

// Heuristic is a learned rule extracted from agent execution experience.
// Heuristics can be derived from failures ("avoid X") or successes ("prefer X").
type Heuristic struct {
	// ID uniquely identifies this heuristic.
	ID string

	// Content is the human-readable heuristic text.
	Content string

	// Source indicates how this heuristic was derived: "failure" or "success".
	Source string

	// TaskType categorizes the task that produced this heuristic.
	TaskType string

	// Utility is the estimated usefulness of this heuristic (0.0 to 1.0).
	Utility float64

	// UsageCount tracks how many times this heuristic has been retrieved.
	UsageCount int

	// CreatedAt is when this heuristic was first extracted.
	CreatedAt time.Time
}

// CapabilityScore tracks an agent's performance on a specific task type
// using exponential moving average for smooth updates.
type CapabilityScore struct {
	// TaskType identifies the task category.
	TaskType string

	// SuccessRate is the EMA of success outcomes (0.0 to 1.0).
	SuccessRate float64

	// SampleCount is the total number of observations.
	SampleCount int

	// AvgLatency is the EMA of execution duration.
	AvgLatency time.Duration

	// LastUpdated is when this score was last modified.
	LastUpdated time.Time
}

// MonitoringSignals captures execution telemetry from a single agent turn.
type MonitoringSignals struct {
	// TaskInput is the original user input for this turn.
	TaskInput string

	// TaskType categorizes the task (may be empty if unclassified).
	TaskType string

	// Outcome is the agent's final response text.
	Outcome string

	// Success indicates whether the turn completed without errors.
	Success bool

	// ToolCalls lists the names of tools invoked during the turn.
	ToolCalls []string

	// IterationCount is the number of plan-act loop iterations.
	IterationCount int

	// TotalLatency is the wall-clock duration of the turn.
	TotalLatency time.Duration

	// Errors collects error messages encountered during the turn.
	Errors []string
}

// Hooks provides optional callbacks for observing metacognitive events.
// All fields are optional; nil hooks are skipped.
type Hooks struct {
	// OnHeuristicExtracted is called when a new heuristic is derived.
	OnHeuristicExtracted func(h Heuristic)

	// OnCapabilityUpdated is called when a capability score changes.
	OnCapabilityUpdated func(taskType string, score CapabilityScore)

	// OnSelfModelLoaded is called when a self-model is loaded from the store.
	OnSelfModelLoaded func(model *SelfModel)
}
