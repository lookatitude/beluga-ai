package trajectory

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecorder_BasicExecution(t *testing.T) {
	rec := NewRecorder(WithAgentID("test-agent"), WithTrajectoryID("traj-1"))
	hooks := rec.Hooks()

	ctx := context.Background()

	// Simulate agent execution.
	require.NoError(t, hooks.OnStart(ctx, "solve the problem"))

	require.NoError(t, hooks.AfterPlan(ctx, []agent.Action{
		{Type: agent.ActionTool, ToolCall: &schema.ToolCall{Name: "search"}},
	}))

	require.NoError(t, hooks.OnToolCall(ctx, agent.ToolCallInfo{
		Name:      "search",
		Arguments: `{"query":"test"}`,
		CallID:    "call-1",
	}))

	require.NoError(t, hooks.OnToolResult(ctx, agent.ToolCallInfo{
		Name:   "search",
		CallID: "call-1",
	}, tool.TextResult("found it")))

	hooks.OnEnd(ctx, "the answer is 42", nil)

	traj := rec.Trajectory()
	require.NotNil(t, traj)
	assert.Equal(t, "test-agent", traj.AgentID)
	assert.Equal(t, "traj-1", traj.ID)
	assert.Equal(t, "solve the problem", traj.Input)
	assert.Equal(t, "the answer is 42", traj.Output)

	// Should have: plan, tool_call, finish
	require.Len(t, traj.Steps, 3)
	assert.Equal(t, StepPlan, traj.Steps[0].Type)
	assert.Equal(t, StepToolCall, traj.Steps[1].Type)
	assert.Equal(t, "search", traj.Steps[1].Action.ToolName)
	assert.Equal(t, `{"query":"test"}`, traj.Steps[1].Action.ToolArgs)
	assert.Equal(t, "found it", traj.Steps[1].Result.Output)
	assert.Equal(t, StepFinish, traj.Steps[2].Type)
}

func TestRecorder_Handoff(t *testing.T) {
	rec := NewRecorder()
	hooks := rec.Hooks()
	ctx := context.Background()

	require.NoError(t, hooks.OnStart(ctx, "input"))
	require.NoError(t, hooks.OnHandoff(ctx, "agent-a", "agent-b"))
	hooks.OnEnd(ctx, "output", nil)

	traj := rec.Trajectory()
	require.NotNil(t, traj)
	require.Len(t, traj.Steps, 2) // handoff, finish
	assert.Equal(t, StepHandoff, traj.Steps[0].Type)
	assert.Equal(t, "agent-b", traj.Steps[0].Action.Target)
}

func TestRecorder_Reset(t *testing.T) {
	rec := NewRecorder(WithAgentID("a"))
	hooks := rec.Hooks()
	ctx := context.Background()

	require.NoError(t, hooks.OnStart(ctx, "input"))
	hooks.OnEnd(ctx, "output", nil)

	assert.NotNil(t, rec.Trajectory())

	rec.Reset()
	assert.Nil(t, rec.Trajectory())
}

func TestRecorder_NilBeforeStart(t *testing.T) {
	rec := NewRecorder()
	assert.Nil(t, rec.Trajectory())
}

func TestRecorder_TrajectoryReturnsCopy(t *testing.T) {
	rec := NewRecorder()
	hooks := rec.Hooks()
	ctx := context.Background()

	require.NoError(t, hooks.OnStart(ctx, "input"))
	hooks.OnEnd(ctx, "output", nil)

	traj1 := rec.Trajectory()
	traj2 := rec.Trajectory()
	require.NotNil(t, traj1)
	require.NotNil(t, traj2)

	// Modifying one copy should not affect the other.
	traj1.Steps = nil
	assert.NotNil(t, traj2.Steps)
}

func TestRecorder_ComposeHooks(t *testing.T) {
	rec := NewRecorder()
	var userHookCalled bool
	userHooks := agent.Hooks{
		OnStart: func(_ context.Context, _ string) error {
			userHookCalled = true
			return nil
		},
	}

	composed := agent.ComposeHooks(rec.Hooks(), userHooks)
	ctx := context.Background()
	require.NoError(t, composed.OnStart(ctx, "input"))

	assert.True(t, userHookCalled)
	// Recorder should also have been called (startTime set).
	rec.mu.Lock()
	started := !rec.startTime.IsZero()
	rec.mu.Unlock()
	assert.True(t, started)
}

func TestFormatActions(t *testing.T) {
	tests := []struct {
		name    string
		actions []agent.Action
		want    string
	}{
		{
			name:    "empty",
			actions: nil,
			want:    "no actions planned",
		},
		{
			name: "single tool",
			actions: []agent.Action{
				{Type: agent.ActionTool, ToolCall: &schema.ToolCall{Name: "search"}},
			},
			want: "tool:search",
		},
		{
			name: "mixed actions",
			actions: []agent.Action{
				{Type: agent.ActionTool, ToolCall: &schema.ToolCall{Name: "search"}},
				{Type: agent.ActionRespond},
				{Type: agent.ActionFinish},
			},
			want: "tool:search; respond; finish",
		},
		{
			name: "tool with nil toolcall",
			actions: []agent.Action{
				{Type: agent.ActionTool},
			},
			want: "tool:(unknown)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatActions(tt.actions)
			assert.Equal(t, tt.want, got)
		})
	}
}
