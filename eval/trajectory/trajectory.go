package trajectory

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/core"
)

// StepType identifies the kind of step in an agent trajectory.
type StepType string

const (
	// StepPlan indicates a planning step where the agent decides what to do.
	StepPlan StepType = "plan"
	// StepToolCall indicates a tool invocation step.
	StepToolCall StepType = "tool_call"
	// StepRespond indicates a response step where the agent produces output.
	StepRespond StepType = "respond"
	// StepHandoff indicates a handoff to another agent.
	StepHandoff StepType = "handoff"
	// StepFinish indicates the agent has finished execution.
	StepFinish StepType = "finish"
)

// StepAction describes what the agent did in a step.
type StepAction struct {
	// ToolName is the name of the tool invoked (for StepToolCall).
	ToolName string `json:"tool_name,omitempty"`
	// ToolArgs is the JSON-encoded arguments passed to the tool.
	ToolArgs string `json:"tool_args,omitempty"`
	// Message is the text content of the step (for StepRespond/StepFinish).
	Message string `json:"message,omitempty"`
	// Target is the target agent name (for StepHandoff).
	Target string `json:"target,omitempty"`
}

// StepResult describes the outcome of a step.
type StepResult struct {
	// Output is the result content of the step.
	Output string `json:"output,omitempty"`
	// Error is the error message if the step failed.
	Error string `json:"error,omitempty"`
}

// Step represents a single step in an agent trajectory.
type Step struct {
	// Index is the zero-based position of this step in the trajectory.
	Index int `json:"index"`
	// Type identifies the kind of step.
	Type StepType `json:"type"`
	// Action describes what was done.
	Action StepAction `json:"action"`
	// Result describes the outcome.
	Result StepResult `json:"result"`
	// Latency is how long this step took.
	Latency time.Duration `json:"latency"`
	// Timestamp is when this step started.
	Timestamp time.Time `json:"timestamp"`
	// Metadata holds step-specific data (e.g., token counts).
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Trajectory represents a complete agent execution trace.
type Trajectory struct {
	// ID uniquely identifies this trajectory.
	ID string `json:"id"`
	// AgentID identifies the agent that produced this trajectory.
	AgentID string `json:"agent_id"`
	// Input is the original user input.
	Input string `json:"input"`
	// Output is the final agent output.
	Output string `json:"output"`
	// ExpectedOutput is the ground-truth or reference answer.
	ExpectedOutput string `json:"expected_output,omitempty"`
	// ExpectedTools lists the tool names that should have been used.
	ExpectedTools []string `json:"expected_tools,omitempty"`
	// Steps is the ordered sequence of steps the agent took.
	Steps []Step `json:"steps"`
	// TotalLatency is the end-to-end duration.
	TotalLatency time.Duration `json:"total_latency"`
	// Timestamp is when the trajectory started.
	Timestamp time.Time `json:"timestamp"`
	// Metadata holds trajectory-level data (e.g., total tokens).
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ActualTools returns the deduplicated list of tool names used in this trajectory.
func (t *Trajectory) ActualTools() []string {
	seen := make(map[string]bool)
	var tools []string
	for _, s := range t.Steps {
		if s.Type == StepToolCall && s.Action.ToolName != "" {
			if !seen[s.Action.ToolName] {
				seen[s.Action.ToolName] = true
				tools = append(tools, s.Action.ToolName)
			}
		}
	}
	return tools
}

// LoadTrajectories reads trajectories from a JSON file at the given path.
// The path is cleaned and validated to prevent path traversal.
func LoadTrajectories(path string) ([]Trajectory, error) {
	clean := filepath.Clean(path)
	if strings.Contains(clean, "..") {
		return nil, core.Errorf(core.ErrInvalidInput, "trajectory: invalid path containing '..'")
	}

	data, err := os.ReadFile(clean)
	if err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, "trajectory: read file: %w", err)
	}

	var trajectories []Trajectory
	if err := json.Unmarshal(data, &trajectories); err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, "trajectory: unmarshal: %w", err)
	}

	return trajectories, nil
}

// SaveTrajectories writes trajectories to a JSON file at the given path.
// The path is cleaned and validated to prevent path traversal.
func SaveTrajectories(path string, trajectories []Trajectory) error {
	clean := filepath.Clean(path)
	if strings.Contains(clean, "..") {
		return core.Errorf(core.ErrInvalidInput, "trajectory: invalid path containing '..'")
	}

	data, err := json.MarshalIndent(trajectories, "", "  ")
	if err != nil {
		return core.Errorf(core.ErrInvalidInput, "trajectory: marshal: %w", err)
	}

	return os.WriteFile(clean, data, 0o600)
}
