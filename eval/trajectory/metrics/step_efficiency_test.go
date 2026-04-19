package metrics

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/eval/trajectory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepEfficiency_EmptyTrajectory(t *testing.T) {
	se := NewStepEfficiency()
	score, err := se.ScoreTrajectory(context.Background(), trajectory.Trajectory{})
	require.NoError(t, err)
	assert.Equal(t, 0.0, score.Overall)
}

func TestStepEfficiency_AllProductive(t *testing.T) {
	se := NewStepEfficiency()
	traj := trajectory.Trajectory{
		Steps: []trajectory.Step{
			{Index: 0, Type: trajectory.StepToolCall, Result: trajectory.StepResult{Output: "result"}},
			{Index: 1, Type: trajectory.StepRespond},
			{Index: 2, Type: trajectory.StepFinish},
		},
	}

	score, err := se.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)
	assert.Equal(t, 1.0, score.Overall)
	assert.Equal(t, 3, score.Details["productive_steps"])
}

func TestStepEfficiency_MixedSteps(t *testing.T) {
	se := NewStepEfficiency()
	traj := trajectory.Trajectory{
		Steps: []trajectory.Step{
			{Index: 0, Type: trajectory.StepPlan},                                                  // Not productive
			{Index: 1, Type: trajectory.StepToolCall, Result: trajectory.StepResult{Output: ""}},   // No output -> not productive
			{Index: 2, Type: trajectory.StepToolCall, Result: trajectory.StepResult{Output: "ok"}}, // Productive
			{Index: 3, Type: trajectory.StepFinish},                                                // Productive
		},
	}

	score, err := se.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)
	assert.Equal(t, 0.5, score.Overall) // 2/4
	assert.Equal(t, 2, score.Details["productive_steps"])
	assert.Equal(t, 4, score.Details["total_steps"])
}

func TestStepEfficiency_FailedToolCall(t *testing.T) {
	se := NewStepEfficiency()
	traj := trajectory.Trajectory{
		Steps: []trajectory.Step{
			{Index: 0, Type: trajectory.StepToolCall, Result: trajectory.StepResult{Output: "result", Error: "failed"}},
			{Index: 1, Type: trajectory.StepFinish},
		},
	}

	score, err := se.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)
	assert.Equal(t, 0.5, score.Overall) // Only finish is productive; tool with error is not.

	require.Len(t, score.StepScores, 2)
	assert.Equal(t, 0.0, score.StepScores[0].Score)
	assert.Equal(t, "non-productive", score.StepScores[0].Reason)
	assert.Equal(t, 1.0, score.StepScores[1].Score)
	assert.Equal(t, "productive", score.StepScores[1].Reason)
}

func TestStepEfficiency_HandoffProductive(t *testing.T) {
	se := NewStepEfficiency()
	traj := trajectory.Trajectory{
		Steps: []trajectory.Step{
			{Index: 0, Type: trajectory.StepHandoff},
		},
	}

	score, err := se.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)
	assert.Equal(t, 1.0, score.Overall)
}

func TestStepEfficiency_Name(t *testing.T) {
	se := NewStepEfficiency()
	assert.Equal(t, "step_efficiency", se.Name())
}
