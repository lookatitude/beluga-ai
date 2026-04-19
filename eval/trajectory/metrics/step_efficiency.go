package metrics

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/eval/trajectory"
)

func init() {
	trajectory.Register("step_efficiency", func(_ map[string]any) (trajectory.TrajectoryMetric, error) {
		return NewStepEfficiency(), nil
	})
}

// Compile-time interface check.
var _ trajectory.TrajectoryMetric = (*StepEfficiency)(nil)

// StepEfficiency measures the ratio of productive steps to total steps.
// A step is considered productive if it is a tool call with output, a
// respond step, a handoff, or a finish step. Plan steps and failed tool
// calls are considered non-productive overhead.
type StepEfficiency struct{}

// NewStepEfficiency creates a new StepEfficiency metric.
func NewStepEfficiency() *StepEfficiency {
	return &StepEfficiency{}
}

// Name returns "step_efficiency".
func (se *StepEfficiency) Name() string { return "step_efficiency" }

// ScoreTrajectory computes the ratio of productive steps to total steps.
func (se *StepEfficiency) ScoreTrajectory(_ context.Context, t trajectory.Trajectory) (*trajectory.TrajectoryScore, error) {
	totalSteps := len(t.Steps)
	if totalSteps == 0 {
		return &trajectory.TrajectoryScore{
			Overall: 0,
			Details: map[string]any{
				"productive_steps": 0,
				"total_steps":      0,
			},
		}, nil
	}

	productiveCount := 0
	var stepScores []trajectory.StepScore

	for _, step := range t.Steps {
		productive := isProductive(step)
		score := 0.0
		reason := "non-productive"
		if productive {
			score = 1.0
			reason = "productive"
			productiveCount++
		}
		stepScores = append(stepScores, trajectory.StepScore{
			StepIndex: step.Index,
			Score:     score,
			Reason:    reason,
		})
	}

	overall := float64(productiveCount) / float64(totalSteps)

	return &trajectory.TrajectoryScore{
		Overall:    overall,
		StepScores: stepScores,
		Details: map[string]any{
			"productive_steps": productiveCount,
			"total_steps":      totalSteps,
		},
	}, nil
}

// isProductive determines whether a step contributed to task completion.
func isProductive(step trajectory.Step) bool {
	switch step.Type {
	case trajectory.StepToolCall:
		// Productive if the tool call produced output without error.
		return step.Result.Output != "" && step.Result.Error == ""
	case trajectory.StepRespond, trajectory.StepHandoff, trajectory.StepFinish:
		return true
	case trajectory.StepPlan:
		// Planning is overhead, not directly productive.
		return false
	default:
		return false
	}
}
