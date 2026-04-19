package metrics

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/eval/trajectory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanningQuality_EmptyTrajectory(t *testing.T) {
	pq := NewPlanningQuality()
	traj := trajectory.Trajectory{}

	score, err := pq.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)
	assert.Equal(t, 0.0, score.Overall)
}

func TestPlanningQuality_EfficientTrajectory(t *testing.T) {
	pq := NewPlanningQuality(WithMaxExpectedSteps(5))
	traj := trajectory.Trajectory{
		Steps: []trajectory.Step{
			{Index: 0, Type: trajectory.StepPlan},
			{Index: 1, Type: trajectory.StepToolCall, Action: trajectory.StepAction{ToolName: "search", ToolArgs: `{"q":"test"}`}, Result: trajectory.StepResult{Output: "result"}},
			{Index: 2, Type: trajectory.StepFinish, Action: trajectory.StepAction{Message: "done"}},
		},
	}

	score, err := pq.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)

	// 3 steps <= 5 max, so efficiency = 1.0
	// No redundancy: score = 1.0
	// Has finish: goal = 1.0
	// Overall = 0.4*1.0 + 0.3*1.0 + 0.3*1.0 = 1.0
	assert.Equal(t, 1.0, score.Overall)
	assert.Equal(t, true, score.Details["goal_achieved"])
}

func TestPlanningQuality_OvershootSteps(t *testing.T) {
	pq := NewPlanningQuality(WithMaxExpectedSteps(2))
	steps := make([]trajectory.Step, 4)
	for i := range steps {
		steps[i] = trajectory.Step{
			Index:  i,
			Type:   trajectory.StepToolCall,
			Action: trajectory.StepAction{ToolName: "tool" + string(rune('a'+i)), ToolArgs: "{}"},
		}
	}

	traj := trajectory.Trajectory{Steps: steps}

	score, err := pq.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)

	// 4 steps, max 2: efficiency = 1.0 - (4-2)/2 = 0.0
	assert.Equal(t, 0.0, score.Details["efficiency"])
	// No goal achieved (no finish step).
	assert.Equal(t, false, score.Details["goal_achieved"])
}

func TestPlanningQuality_Redundancy(t *testing.T) {
	pq := NewPlanningQuality(WithMaxExpectedSteps(10))
	traj := trajectory.Trajectory{
		Steps: []trajectory.Step{
			{Index: 0, Type: trajectory.StepToolCall, Action: trajectory.StepAction{ToolName: "search", ToolArgs: `{"q":"test"}`}},
			{Index: 1, Type: trajectory.StepToolCall, Action: trajectory.StepAction{ToolName: "search", ToolArgs: `{"q":"test"}`}},
			{Index: 2, Type: trajectory.StepToolCall, Action: trajectory.StepAction{ToolName: "search", ToolArgs: `{"q":"test"}`}},
			{Index: 3, Type: trajectory.StepFinish, Action: trajectory.StepAction{Message: "done"}},
		},
	}

	score, err := pq.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)

	// 2 out of 3 tool calls are redundant -> redundancy_score = 1.0 - 2/3 = 0.333
	assert.InDelta(t, 1.0/3.0, score.Details["redundancy_score"], 0.01)
	assert.Equal(t, 2, score.Details["redundant_count"])

	// Step scores: first is unique (1.0), second and third are redundant (0.0).
	require.Len(t, score.StepScores, 3)
	assert.Equal(t, 1.0, score.StepScores[0].Score)
	assert.Equal(t, 0.0, score.StepScores[1].Score)
	assert.Equal(t, 0.0, score.StepScores[2].Score)
}

func TestPlanningQuality_DefaultMaxSteps(t *testing.T) {
	pq := NewPlanningQuality()
	assert.Equal(t, 10, pq.maxExpectedSteps)
}

func TestPlanningQuality_Name(t *testing.T) {
	pq := NewPlanningQuality()
	assert.Equal(t, "planning_quality", pq.Name())
}

func TestPlanningQuality_InvalidMaxSteps(t *testing.T) {
	pq := NewPlanningQuality(WithMaxExpectedSteps(-1))
	assert.Equal(t, 10, pq.maxExpectedSteps) // Should keep default.

	pq = NewPlanningQuality(WithMaxExpectedSteps(0))
	assert.Equal(t, 10, pq.maxExpectedSteps)
}
