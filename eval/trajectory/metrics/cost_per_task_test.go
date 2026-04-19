package metrics

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/eval/trajectory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCostPerTask_TrajectoryLevelCost(t *testing.T) {
	tests := []struct {
		name      string
		totalCost float64
		maxBudget float64
		wantScore float64
	}{
		{
			name:      "zero cost",
			totalCost: 0.0,
			maxBudget: 1.0,
			wantScore: 1.0,
		},
		{
			name:      "half budget",
			totalCost: 0.5,
			maxBudget: 1.0,
			wantScore: 0.5,
		},
		{
			name:      "full budget",
			totalCost: 1.0,
			maxBudget: 1.0,
			wantScore: 0.0,
		},
		{
			name:      "over budget clamped",
			totalCost: 2.0,
			maxBudget: 1.0,
			wantScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCostPerTask(WithMaxBudget(tt.maxBudget))
			traj := trajectory.Trajectory{
				Metadata: map[string]any{"total_cost": tt.totalCost},
			}

			score, err := c.ScoreTrajectory(context.Background(), traj)
			require.NoError(t, err)
			assert.InDelta(t, tt.wantScore, score.Overall, 0.001)
			assert.Equal(t, tt.totalCost, score.Details["total_cost"])
			assert.Equal(t, tt.maxBudget, score.Details["max_budget"])
		})
	}
}

func TestCostPerTask_StepLevelCost(t *testing.T) {
	c := NewCostPerTask(WithMaxBudget(1.0))
	traj := trajectory.Trajectory{
		Steps: []trajectory.Step{
			{Index: 0, Metadata: map[string]any{"step_cost": 0.1}},
			{Index: 1, Metadata: map[string]any{"step_cost": 0.2}},
			{Index: 2, Metadata: map[string]any{}}, // No cost.
		},
	}

	score, err := c.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)
	// total = 0.3, score = 1.0 - 0.3/1.0 = 0.7
	assert.InDelta(t, 0.7, score.Overall, 0.001)
}

func TestCostPerTask_NoCostData(t *testing.T) {
	c := NewCostPerTask()
	traj := trajectory.Trajectory{
		Steps: []trajectory.Step{
			{Index: 0},
			{Index: 1},
		},
	}

	score, err := c.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)
	// total = 0, score = 1.0
	assert.Equal(t, 1.0, score.Overall)
}

func TestCostPerTask_InvalidCostType(t *testing.T) {
	c := NewCostPerTask()
	traj := trajectory.Trajectory{
		Metadata: map[string]any{"total_cost": "not a number"},
	}

	_, err := c.ScoreTrajectory(context.Background(), traj)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid total_cost")
}

func TestCostPerTask_InvalidStepCost(t *testing.T) {
	c := NewCostPerTask()
	traj := trajectory.Trajectory{
		Steps: []trajectory.Step{
			{Index: 0, Metadata: map[string]any{"step_cost": "bad"}},
		},
	}

	_, err := c.ScoreTrajectory(context.Background(), traj)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "step_cost")
}

func TestCostPerTask_DefaultBudget(t *testing.T) {
	c := NewCostPerTask()
	assert.Equal(t, 1.0, c.maxBudget)
}

func TestCostPerTask_InvalidBudget(t *testing.T) {
	c := NewCostPerTask(WithMaxBudget(-1))
	assert.Equal(t, 1.0, c.maxBudget) // Should keep default.

	c = NewCostPerTask(WithMaxBudget(0))
	assert.Equal(t, 1.0, c.maxBudget)
}

func TestCostPerTask_Name(t *testing.T) {
	c := NewCostPerTask()
	assert.Equal(t, "cost_per_task", c.Name())
}

func TestCostPerTask_IntCost(t *testing.T) {
	c := NewCostPerTask(WithMaxBudget(10.0))
	traj := trajectory.Trajectory{
		Metadata: map[string]any{"total_cost": 5},
	}

	score, err := c.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)
	assert.InDelta(t, 0.5, score.Overall, 0.001)
}
