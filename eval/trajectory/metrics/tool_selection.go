package metrics

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/eval/trajectory"
)

func init() {
	trajectory.Register("tool_selection", func(_ map[string]any) (trajectory.TrajectoryMetric, error) {
		return NewToolSelection(), nil
	})
}

// Compile-time interface check.
var _ trajectory.TrajectoryMetric = (*ToolSelection)(nil)

// ToolSelection evaluates whether the agent used the expected tools.
// It computes precision, recall, and F1 score comparing actual tool usage
// against ExpectedTools in the trajectory.
type ToolSelection struct{}

// NewToolSelection creates a new ToolSelection metric.
func NewToolSelection() *ToolSelection {
	return &ToolSelection{}
}

// Name returns "tool_selection".
func (ts *ToolSelection) Name() string { return "tool_selection" }

// ScoreTrajectory computes the F1 score of tool usage. Per-step scores
// indicate whether each tool call was expected (1.0) or unexpected (0.0).
func (ts *ToolSelection) ScoreTrajectory(_ context.Context, t trajectory.Trajectory) (*trajectory.TrajectoryScore, error) {
	expected := toSet(t.ExpectedTools)
	actual := t.ActualTools()
	actualSet := toSet(actual)

	// Compute precision, recall, F1.
	var truePositives int
	for _, name := range actual {
		if expected[name] {
			truePositives++
		}
	}

	var precision, recall, f1 float64
	if len(actualSet) > 0 {
		precision = float64(truePositives) / float64(len(actualSet))
	}
	if len(expected) > 0 {
		recall = float64(truePositives) / float64(len(expected))
	}
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}

	// If no expected tools and no actual tools, consider it perfect.
	if len(expected) == 0 && len(actualSet) == 0 {
		f1 = 1.0
		precision = 1.0
		recall = 1.0
	}

	// Per-step scores for tool_call steps.
	var stepScores []trajectory.StepScore
	for _, step := range t.Steps {
		if step.Type == trajectory.StepToolCall {
			score := 0.0
			reason := "unexpected tool"
			if expected[step.Action.ToolName] {
				score = 1.0
				reason = "expected tool"
			}
			stepScores = append(stepScores, trajectory.StepScore{
				StepIndex: step.Index,
				Score:     score,
				Reason:    reason,
			})
		}
	}

	return &trajectory.TrajectoryScore{
		Overall:    f1,
		StepScores: stepScores,
		Details: map[string]any{
			"precision":      precision,
			"recall":         recall,
			"f1":             f1,
			"expected_tools": t.ExpectedTools,
			"actual_tools":   actual,
			"true_positives": truePositives,
		},
	}, nil
}

// toSet converts a string slice to a set.
func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}
