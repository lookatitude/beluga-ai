package metrics

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/eval/trajectory"
)

func init() {
	trajectory.Register("planning_quality", func(cfg map[string]any) (trajectory.TrajectoryMetric, error) {
		var opts []PlanningQualityOption
		if v, ok := cfg["max_expected_steps"]; ok {
			if n, ok := v.(int); ok && n > 0 {
				opts = append(opts, WithMaxExpectedSteps(n))
			}
		}
		return NewPlanningQuality(opts...), nil
	})
}

// Compile-time interface check.
var _ trajectory.TrajectoryMetric = (*PlanningQuality)(nil)

// PlanningQualityOption configures a PlanningQuality metric.
type PlanningQualityOption func(*PlanningQuality)

// WithMaxExpectedSteps sets the maximum number of steps expected for
// efficient task completion. Steps beyond this count reduce the score.
func WithMaxExpectedSteps(n int) PlanningQualityOption {
	return func(pq *PlanningQuality) {
		if n > 0 {
			pq.maxExpectedSteps = n
		}
	}
}

// PlanningQuality evaluates the quality of an agent's planning by measuring
// step efficiency, redundancy, and goal achievement.
type PlanningQuality struct {
	maxExpectedSteps int
}

// NewPlanningQuality creates a new PlanningQuality metric with the given options.
func NewPlanningQuality(opts ...PlanningQualityOption) *PlanningQuality {
	pq := &PlanningQuality{
		maxExpectedSteps: 10,
	}
	for _, opt := range opts {
		opt(pq)
	}
	return pq
}

// Name returns "planning_quality".
func (pq *PlanningQuality) Name() string { return "planning_quality" }

// ScoreTrajectory evaluates planning quality based on step efficiency,
// redundancy detection, and goal achievement.
func (pq *PlanningQuality) ScoreTrajectory(_ context.Context, t trajectory.Trajectory) (*trajectory.TrajectoryScore, error) {
	totalSteps := len(t.Steps)
	if totalSteps == 0 {
		return &trajectory.TrajectoryScore{
			Overall: 0,
			Details: map[string]any{
				"efficiency":    0.0,
				"redundancy":    0.0,
				"goal_achieved": false,
				"total_steps":   0,
			},
		}, nil
	}

	// Step efficiency: how close to optimal step count.
	efficiency := 1.0
	if totalSteps > pq.maxExpectedSteps {
		// Linearly decrease from 1.0 to 0.0 as steps go from max to 2*max.
		overshoot := float64(totalSteps-pq.maxExpectedSteps) / float64(pq.maxExpectedSteps)
		efficiency = 1.0 - overshoot
		if efficiency < 0 {
			efficiency = 0
		}
	}

	// Redundancy: detect repeated tool calls with same name and args.
	type toolSig struct {
		name string
		args string
	}
	seen := make(map[toolSig]int)
	redundantCount := 0
	var stepScores []trajectory.StepScore

	for _, step := range t.Steps {
		if step.Type == trajectory.StepToolCall {
			sig := toolSig{name: step.Action.ToolName, args: step.Action.ToolArgs}
			seen[sig]++
			if seen[sig] > 1 {
				redundantCount++
				stepScores = append(stepScores, trajectory.StepScore{
					StepIndex: step.Index,
					Score:     0.0,
					Reason:    "redundant tool call",
				})
			} else {
				stepScores = append(stepScores, trajectory.StepScore{
					StepIndex: step.Index,
					Score:     1.0,
					Reason:    "unique tool call",
				})
			}
		}
	}

	// Redundancy score: 1.0 means no redundancy.
	toolCallCount := 0
	for _, step := range t.Steps {
		if step.Type == trajectory.StepToolCall {
			toolCallCount++
		}
	}
	redundancyScore := 1.0
	if toolCallCount > 0 {
		redundancyScore = 1.0 - float64(redundantCount)/float64(toolCallCount)
	}

	// Goal achievement: did the trajectory end with a finish step and produce output?
	goalAchieved := false
	for _, step := range t.Steps {
		if step.Type == trajectory.StepFinish {
			goalAchieved = true
			break
		}
	}
	goalScore := 0.0
	if goalAchieved {
		goalScore = 1.0
	}

	// Overall: weighted average (efficiency 40%, redundancy 30%, goal 30%).
	overall := 0.4*efficiency + 0.3*redundancyScore + 0.3*goalScore

	return &trajectory.TrajectoryScore{
		Overall:    overall,
		StepScores: stepScores,
		Details: map[string]any{
			"efficiency":         efficiency,
			"redundancy_score":   redundancyScore,
			"redundant_count":    redundantCount,
			"goal_achieved":      goalAchieved,
			"goal_score":         goalScore,
			"total_steps":        totalSteps,
			"max_expected_steps": pq.maxExpectedSteps,
		},
	}, nil
}
