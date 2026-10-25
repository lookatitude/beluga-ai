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
	efficiency := stepEfficiency(totalSteps, pq.maxExpectedSteps)

	// Redundancy: detect repeated tool calls with same name and args.
	stepScores, redundantCount, toolCallCount := scoreToolRedundancy(t.Steps)

	// Redundancy score: 1.0 means no redundancy.
	redundancyScore := 1.0
	if toolCallCount > 0 {
		redundancyScore = 1.0 - float64(redundantCount)/float64(toolCallCount)
	}

	// Goal achievement: did the trajectory end with a finish step and produce output?
	goalAchieved := hasFinishStep(t.Steps)
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

// stepEfficiency scores how close totalSteps is to maxExpected: 1.0 at or
// below maxExpected, decreasing linearly to 0.0 at 2*maxExpected and beyond.
func stepEfficiency(totalSteps, maxExpected int) float64 {
	if totalSteps <= maxExpected {
		return 1.0
	}
	overshoot := float64(totalSteps-maxExpected) / float64(maxExpected)
	efficiency := 1.0 - overshoot
	if efficiency < 0 {
		return 0
	}
	return efficiency
}

// scoreToolRedundancy scores every tool-call step: 1.0 for the first call with
// a given tool name and args, 0.0 for repeats. It also returns the number of
// repeated calls and the total number of tool calls.
func scoreToolRedundancy(steps []trajectory.Step) (stepScores []trajectory.StepScore, redundantCount, toolCallCount int) {
	type toolSig struct {
		name string
		args string
	}
	seen := make(map[toolSig]int)
	for i := range steps {
		step := &steps[i]
		if step.Type != trajectory.StepToolCall {
			continue
		}
		toolCallCount++
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
	return stepScores, redundantCount, toolCallCount
}

// hasFinishStep reports whether any step in steps is a finish step.
func hasFinishStep(steps []trajectory.Step) bool {
	for i := range steps {
		if steps[i].Type == trajectory.StepFinish {
			return true
		}
	}
	return false
}
