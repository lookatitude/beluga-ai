package trajectory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrajectoryScore_Fields(t *testing.T) {
	score := &TrajectoryScore{
		Overall: 0.85,
		StepScores: []StepScore{
			{StepIndex: 0, Score: 1.0, Reason: "good"},
			{StepIndex: 1, Score: 0.5, Reason: "partial"},
		},
		Details: map[string]any{
			"precision": 0.9,
			"recall":    0.8,
		},
	}

	assert.Equal(t, 0.85, score.Overall)
	assert.Len(t, score.StepScores, 2)
	assert.Equal(t, 0, score.StepScores[0].StepIndex)
	assert.Equal(t, 1.0, score.StepScores[0].Score)
	assert.Equal(t, "good", score.StepScores[0].Reason)
	assert.Equal(t, 0.9, score.Details["precision"])
}

func TestTrajectoryResult_Fields(t *testing.T) {
	result := TrajectoryResult{
		TrajectoryID: "t1",
		Scores: map[string]*TrajectoryScore{
			"tool_selection":  {Overall: 0.9},
			"step_efficiency": {Overall: 0.75},
		},
	}

	assert.Equal(t, "t1", result.TrajectoryID)
	assert.Len(t, result.Scores, 2)
	assert.Equal(t, 0.9, result.Scores["tool_selection"].Overall)
}

func TestReport_Fields(t *testing.T) {
	report := Report{
		Trajectories: []TrajectoryResult{
			{TrajectoryID: "t1"},
		},
		Aggregate: map[string]float64{
			"tool_selection": 0.85,
		},
	}

	assert.Len(t, report.Trajectories, 1)
	assert.Equal(t, 0.85, report.Aggregate["tool_selection"])
}
