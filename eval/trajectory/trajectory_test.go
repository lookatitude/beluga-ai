package trajectory

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepTypes(t *testing.T) {
	assert.Equal(t, StepType("plan"), StepPlan)
	assert.Equal(t, StepType("tool_call"), StepToolCall)
	assert.Equal(t, StepType("respond"), StepRespond)
	assert.Equal(t, StepType("handoff"), StepHandoff)
	assert.Equal(t, StepType("finish"), StepFinish)
}

func TestTrajectory_ActualTools(t *testing.T) {
	tests := []struct {
		name  string
		steps []Step
		want  []string
	}{
		{
			name:  "no steps",
			steps: nil,
			want:  nil,
		},
		{
			name: "no tool calls",
			steps: []Step{
				{Type: StepPlan, Action: StepAction{Message: "plan"}},
				{Type: StepFinish, Action: StepAction{Message: "done"}},
			},
			want: nil,
		},
		{
			name: "unique tools",
			steps: []Step{
				{Type: StepToolCall, Action: StepAction{ToolName: "search"}},
				{Type: StepToolCall, Action: StepAction{ToolName: "calculate"}},
			},
			want: []string{"search", "calculate"},
		},
		{
			name: "deduplicates tools",
			steps: []Step{
				{Type: StepToolCall, Action: StepAction{ToolName: "search"}},
				{Type: StepToolCall, Action: StepAction{ToolName: "search"}},
				{Type: StepToolCall, Action: StepAction{ToolName: "calculate"}},
			},
			want: []string{"search", "calculate"},
		},
		{
			name: "ignores empty tool names",
			steps: []Step{
				{Type: StepToolCall, Action: StepAction{ToolName: ""}},
				{Type: StepToolCall, Action: StepAction{ToolName: "search"}},
			},
			want: []string{"search"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			traj := &Trajectory{Steps: tt.steps}
			got := traj.ActualTools()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadSaveTrajectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "trajectories.json")

	original := []Trajectory{
		{
			ID:             "t1",
			AgentID:        "agent-1",
			Input:          "hello",
			Output:         "world",
			ExpectedOutput: "world",
			ExpectedTools:  []string{"search"},
			Steps: []Step{
				{
					Index:     0,
					Type:      StepToolCall,
					Action:    StepAction{ToolName: "search", ToolArgs: `{"q":"hello"}`},
					Result:    StepResult{Output: "found it"},
					Latency:   100 * time.Millisecond,
					Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				{
					Index:     1,
					Type:      StepFinish,
					Action:    StepAction{Message: "world"},
					Result:    StepResult{Output: "world"},
					Latency:   50 * time.Millisecond,
					Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 100000000, time.UTC),
				},
			},
			TotalLatency: 150 * time.Millisecond,
			Timestamp:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Metadata:     map[string]any{"total_cost": 0.05},
		},
	}

	err := SaveTrajectories(path, original)
	require.NoError(t, err)

	loaded, err := LoadTrajectories(path)
	require.NoError(t, err)
	require.Len(t, loaded, 1)

	assert.Equal(t, original[0].ID, loaded[0].ID)
	assert.Equal(t, original[0].AgentID, loaded[0].AgentID)
	assert.Equal(t, original[0].Input, loaded[0].Input)
	assert.Equal(t, original[0].Output, loaded[0].Output)
	assert.Equal(t, original[0].ExpectedOutput, loaded[0].ExpectedOutput)
	assert.Equal(t, original[0].ExpectedTools, loaded[0].ExpectedTools)
	assert.Len(t, loaded[0].Steps, 2)
	assert.Equal(t, StepToolCall, loaded[0].Steps[0].Type)
	assert.Equal(t, "search", loaded[0].Steps[0].Action.ToolName)
}

func TestLoadTrajectories_FileNotFound(t *testing.T) {
	_, err := LoadTrajectories("/nonexistent/file.json")
	assert.Error(t, err)
}

func TestLoadTrajectories_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o644))

	_, err := LoadTrajectories(path)
	assert.Error(t, err)
}

func TestLoadTrajectories_PathTraversal(t *testing.T) {
	_, err := LoadTrajectories("../../etc/passwd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "..")
}

func TestSaveTrajectories_PathTraversal(t *testing.T) {
	err := SaveTrajectories("../../tmp/evil.json", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "..")
}

func TestStep_Metadata(t *testing.T) {
	step := Step{
		Index: 0,
		Type:  StepToolCall,
		Metadata: map[string]any{
			"input_tokens":  100,
			"output_tokens": 50,
		},
	}
	assert.Equal(t, 100, step.Metadata["input_tokens"])
	assert.Equal(t, 50, step.Metadata["output_tokens"])
}
