package metrics

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/lookatitude/beluga-ai/eval/trajectory"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

const trajectoryFaithfulnessPrompt = `You are an evaluation judge assessing whether an agent's trajectory faithfully follows a logical path to answer the given input.

Input: %s

Expected Output: %s

Agent's Final Output: %s

Agent's Trajectory (steps taken):
%s

Evaluate the trajectory's faithfulness on a scale from 0.0 to 1.0:
- 1.0: Every step logically follows from the previous, tools are used appropriately, and the final output correctly addresses the input.
- 0.5: Some steps are logical but others are unnecessary or misguided; the output partially addresses the input.
- 0.0: Steps are illogical, tools are misused, or the output fails to address the input.

Respond with ONLY a single decimal number between 0.0 and 1.0.`

// Compile-time interface check.
var _ trajectory.TrajectoryMetric = (*TrajectoryFaithfulness)(nil)

// TrajectoryFaithfulnessOption configures a TrajectoryFaithfulness metric.
type TrajectoryFaithfulnessOption func(*TrajectoryFaithfulness)

// WithModel sets the LLM judge model for faithfulness evaluation.
func WithModel(model llm.ChatModel) TrajectoryFaithfulnessOption {
	return func(tf *TrajectoryFaithfulness) {
		tf.llm = model
	}
}

// TrajectoryFaithfulness evaluates whether an agent's trajectory faithfully
// follows a logical path to accomplish its task. It uses an LLM as a judge.
type TrajectoryFaithfulness struct {
	llm llm.ChatModel
}

// NewTrajectoryFaithfulness creates a new TrajectoryFaithfulness metric.
// A ChatModel must be provided via WithModel for the metric to function.
func NewTrajectoryFaithfulness(opts ...TrajectoryFaithfulnessOption) *TrajectoryFaithfulness {
	tf := &TrajectoryFaithfulness{}
	for _, opt := range opts {
		opt(tf)
	}
	return tf
}

// Name returns "trajectory_faithfulness".
func (tf *TrajectoryFaithfulness) Name() string { return "trajectory_faithfulness" }

// ScoreTrajectory evaluates the trajectory's faithfulness using an LLM judge.
// Returns an error if no LLM model was configured.
func (tf *TrajectoryFaithfulness) ScoreTrajectory(ctx context.Context, t trajectory.Trajectory) (*trajectory.TrajectoryScore, error) {
	if tf.llm == nil {
		return nil, fmt.Errorf("trajectory_faithfulness: no LLM model configured")
	}

	stepsText := formatTrajectorySteps(t.Steps)
	prompt := fmt.Sprintf(trajectoryFaithfulnessPrompt, t.Input, t.ExpectedOutput, t.Output, stepsText)

	resp, err := tf.llm.Generate(ctx, []schema.Message{
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return nil, fmt.Errorf("trajectory_faithfulness: llm generate: %w", err)
	}

	score, err := parseScoreResponse(resp.Text())
	if err != nil {
		return nil, fmt.Errorf("trajectory_faithfulness: %w", err)
	}

	return &trajectory.TrajectoryScore{
		Overall: score,
		Details: map[string]any{
			"judge_model": tf.llm.ModelID(),
		},
	}, nil
}

// formatTrajectorySteps formats steps into a readable text for the LLM prompt.
func formatTrajectorySteps(steps []trajectory.Step) string {
	if len(steps) == 0 {
		return "(no steps)"
	}
	var b strings.Builder
	for _, s := range steps {
		fmt.Fprintf(&b, "Step %d [%s]: ", s.Index, s.Type)
		switch s.Type {
		case trajectory.StepToolCall:
			fmt.Fprintf(&b, "Called tool %q with args: %s", s.Action.ToolName, s.Action.ToolArgs)
			if s.Result.Output != "" {
				fmt.Fprintf(&b, " -> Result: %s", s.Result.Output)
			}
			if s.Result.Error != "" {
				fmt.Fprintf(&b, " -> Error: %s", s.Result.Error)
			}
		case trajectory.StepPlan:
			fmt.Fprintf(&b, "Plan: %s", s.Action.Message)
		case trajectory.StepRespond:
			fmt.Fprintf(&b, "Respond: %s", s.Action.Message)
		case trajectory.StepHandoff:
			fmt.Fprintf(&b, "Handoff to %s", s.Action.Target)
		case trajectory.StepFinish:
			fmt.Fprintf(&b, "Finish: %s", s.Action.Message)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// parseScoreResponse extracts a float64 score from an LLM response string.
func parseScoreResponse(text string) (float64, error) {
	text = strings.TrimSpace(text)
	score, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse score from response %q: %w", text, err)
	}
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return score, nil
}
