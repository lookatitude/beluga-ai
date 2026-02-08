package agent

import (
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

func TestActionType_Constants(t *testing.T) {
	tests := []struct {
		action ActionType
		want   string
	}{
		{ActionTool, "tool"},
		{ActionRespond, "respond"},
		{ActionFinish, "finish"},
		{ActionHandoff, "handoff"},
	}
	for _, tt := range tests {
		if string(tt.action) != tt.want {
			t.Errorf("ActionType %v = %q, want %q", tt.action, string(tt.action), tt.want)
		}
	}
}

func TestPlannerState_Fields(t *testing.T) {
	state := PlannerState{
		Input:     "test input",
		Messages:  []schema.Message{schema.NewHumanMessage("hello")},
		Tools:     []tool.Tool{&simpleTool{toolName: "calc"}},
		Iteration: 3,
		Metadata:  map[string]any{"key": "val"},
	}

	if state.Input != "test input" {
		t.Errorf("Input = %q, want %q", state.Input, "test input")
	}
	if len(state.Messages) != 1 {
		t.Errorf("Messages len = %d, want 1", len(state.Messages))
	}
	if len(state.Tools) != 1 {
		t.Errorf("Tools len = %d, want 1", len(state.Tools))
	}
	if state.Iteration != 3 {
		t.Errorf("Iteration = %d, want 3", state.Iteration)
	}
	if state.Metadata["key"] != "val" {
		t.Errorf("Metadata[key] = %v, want %q", state.Metadata["key"], "val")
	}
}

func TestAction_Fields(t *testing.T) {
	tc := &schema.ToolCall{
		ID:        "call-1",
		Name:      "search",
		Arguments: `{"q":"test"}`,
	}
	action := Action{
		Type:     ActionTool,
		ToolCall: tc,
		Message:  "searching...",
		Metadata: map[string]any{"retries": 0},
	}

	if action.Type != ActionTool {
		t.Errorf("Type = %q, want %q", action.Type, ActionTool)
	}
	if action.ToolCall.Name != "search" {
		t.Errorf("ToolCall.Name = %q, want %q", action.ToolCall.Name, "search")
	}
	if action.Message != "searching..." {
		t.Errorf("Message = %q, want %q", action.Message, "searching...")
	}
}

func TestObservation_Fields(t *testing.T) {
	obs := Observation{
		Action: Action{Type: ActionTool, Message: "test"},
		Result: tool.TextResult("result text"),
		Error:  nil,
		Latency: 100 * time.Millisecond,
	}

	if obs.Action.Type != ActionTool {
		t.Errorf("Action.Type = %q, want %q", obs.Action.Type, ActionTool)
	}
	if obs.Result == nil {
		t.Fatal("Result should not be nil")
	}
	if obs.Error != nil {
		t.Errorf("Error = %v, want nil", obs.Error)
	}
	if obs.Latency != 100*time.Millisecond {
		t.Errorf("Latency = %v, want %v", obs.Latency, 100*time.Millisecond)
	}
}

func TestPlannerInterface_MockSatisfies(t *testing.T) {
	var _ Planner = (*mockPlanner)(nil)
}
