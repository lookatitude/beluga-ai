package metrics

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/eval/trajectory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolSelection_PerfectMatch(t *testing.T) {
	ts := NewToolSelection()
	traj := trajectory.Trajectory{
		ExpectedTools: []string{"search", "calculate"},
		Steps: []trajectory.Step{
			{Index: 0, Type: trajectory.StepToolCall, Action: trajectory.StepAction{ToolName: "search"}},
			{Index: 1, Type: trajectory.StepToolCall, Action: trajectory.StepAction{ToolName: "calculate"}},
		},
	}

	score, err := ts.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)
	assert.Equal(t, 1.0, score.Overall)
	assert.Equal(t, 1.0, score.Details["precision"])
	assert.Equal(t, 1.0, score.Details["recall"])
}

func TestToolSelection_NoExpectedNoActual(t *testing.T) {
	ts := NewToolSelection()
	traj := trajectory.Trajectory{
		Steps: []trajectory.Step{
			{Index: 0, Type: trajectory.StepPlan},
			{Index: 1, Type: trajectory.StepFinish},
		},
	}

	score, err := ts.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)
	assert.Equal(t, 1.0, score.Overall)
}

func TestToolSelection_MissingTools(t *testing.T) {
	ts := NewToolSelection()
	traj := trajectory.Trajectory{
		ExpectedTools: []string{"search", "calculate", "summarize"},
		Steps: []trajectory.Step{
			{Index: 0, Type: trajectory.StepToolCall, Action: trajectory.StepAction{ToolName: "search"}},
		},
	}

	score, err := ts.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)
	// precision = 1/1 = 1.0, recall = 1/3, f1 = 2*1.0*(1/3)/(1.0+(1/3)) = 0.5
	assert.InDelta(t, 0.5, score.Overall, 0.01)
}

func TestToolSelection_ExtraTools(t *testing.T) {
	ts := NewToolSelection()
	traj := trajectory.Trajectory{
		ExpectedTools: []string{"search"},
		Steps: []trajectory.Step{
			{Index: 0, Type: trajectory.StepToolCall, Action: trajectory.StepAction{ToolName: "search"}},
			{Index: 1, Type: trajectory.StepToolCall, Action: trajectory.StepAction{ToolName: "irrelevant"}},
			{Index: 2, Type: trajectory.StepToolCall, Action: trajectory.StepAction{ToolName: "also_irrelevant"}},
		},
	}

	score, err := ts.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)
	// precision = 1/3, recall = 1/1 = 1.0, f1 = 2*(1/3)*1.0/((1/3)+1.0) = 0.5
	assert.InDelta(t, 0.5, score.Overall, 0.01)
}

func TestToolSelection_NoOverlap(t *testing.T) {
	ts := NewToolSelection()
	traj := trajectory.Trajectory{
		ExpectedTools: []string{"search"},
		Steps: []trajectory.Step{
			{Index: 0, Type: trajectory.StepToolCall, Action: trajectory.StepAction{ToolName: "wrong_tool"}},
		},
	}

	score, err := ts.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)
	assert.Equal(t, 0.0, score.Overall)
}

func TestToolSelection_StepScores(t *testing.T) {
	ts := NewToolSelection()
	traj := trajectory.Trajectory{
		ExpectedTools: []string{"search"},
		Steps: []trajectory.Step{
			{Index: 0, Type: trajectory.StepToolCall, Action: trajectory.StepAction{ToolName: "search"}},
			{Index: 1, Type: trajectory.StepToolCall, Action: trajectory.StepAction{ToolName: "wrong"}},
			{Index: 2, Type: trajectory.StepPlan},
		},
	}

	score, err := ts.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)

	require.Len(t, score.StepScores, 2) // Only tool_call steps.
	assert.Equal(t, 1.0, score.StepScores[0].Score)
	assert.Equal(t, "expected tool", score.StepScores[0].Reason)
	assert.Equal(t, 0.0, score.StepScores[1].Score)
	assert.Equal(t, "unexpected tool", score.StepScores[1].Reason)
}

func TestToolSelection_Name(t *testing.T) {
	ts := NewToolSelection()
	assert.Equal(t, "tool_selection", ts.Name())
}
