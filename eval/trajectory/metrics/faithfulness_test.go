package metrics

import (
	"context"
	"fmt"
	"testing"

	"github.com/lookatitude/beluga-ai/eval/trajectory"
	"github.com/lookatitude/beluga-ai/internal/testutil/mockllm"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrajectoryFaithfulness_NoModel(t *testing.T) {
	tf := NewTrajectoryFaithfulness()
	_, err := tf.ScoreTrajectory(context.Background(), trajectory.Trajectory{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no LLM model configured")
}

func TestTrajectoryFaithfulness_HighScore(t *testing.T) {
	mock := newMockChatModel(mockllm.WithResponse(schema.NewAIMessage("0.95")))

	tf := NewTrajectoryFaithfulness(WithModel(mock))
	traj := trajectory.Trajectory{
		Input:          "What is 2+2?",
		Output:         "4",
		ExpectedOutput: "4",
		Steps: []trajectory.Step{
			{Index: 0, Type: trajectory.StepToolCall, Action: trajectory.StepAction{ToolName: "calculate", ToolArgs: `{"expr":"2+2"}`}, Result: trajectory.StepResult{Output: "4"}},
			{Index: 1, Type: trajectory.StepFinish, Action: trajectory.StepAction{Message: "4"}},
		},
	}

	score, err := tf.ScoreTrajectory(context.Background(), traj)
	require.NoError(t, err)
	assert.Equal(t, 0.95, score.Overall)
}

func TestTrajectoryFaithfulness_LLMError(t *testing.T) {
	mock := newMockChatModel(mockllm.WithError(fmt.Errorf("api failure")))

	tf := NewTrajectoryFaithfulness(WithModel(mock))
	_, err := tf.ScoreTrajectory(context.Background(), trajectory.Trajectory{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "llm generate")
}

func TestTrajectoryFaithfulness_UnparsableResponse(t *testing.T) {
	mock := newMockChatModel(mockllm.WithResponse(schema.NewAIMessage("not a number")))

	tf := NewTrajectoryFaithfulness(WithModel(mock))
	_, err := tf.ScoreTrajectory(context.Background(), trajectory.Trajectory{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}

func TestTrajectoryFaithfulness_ClampScore(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     float64
	}{
		{"above 1", "1.5", 1.0},
		{"below 0", "-0.5", 0.0},
		{"normal", "0.7", 0.7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := newMockChatModel(mockllm.WithResponse(schema.NewAIMessage(tt.response)))
			tf := NewTrajectoryFaithfulness(WithModel(mock))

			score, err := tf.ScoreTrajectory(context.Background(), trajectory.Trajectory{})
			require.NoError(t, err)
			assert.Equal(t, tt.want, score.Overall)
		})
	}
}

func TestTrajectoryFaithfulness_Name(t *testing.T) {
	tf := NewTrajectoryFaithfulness()
	assert.Equal(t, "trajectory_faithfulness", tf.Name())
}

func TestFormatTrajectorySteps(t *testing.T) {
	steps := []trajectory.Step{
		{Index: 0, Type: trajectory.StepPlan, Action: trajectory.StepAction{Message: "plan to search"}},
		{Index: 1, Type: trajectory.StepToolCall, Action: trajectory.StepAction{ToolName: "search", ToolArgs: `{"q":"test"}`}, Result: trajectory.StepResult{Output: "found"}},
		{Index: 2, Type: trajectory.StepRespond, Action: trajectory.StepAction{Message: "here is the answer"}},
		{Index: 3, Type: trajectory.StepHandoff, Action: trajectory.StepAction{Target: "agent-b"}},
		{Index: 4, Type: trajectory.StepFinish, Action: trajectory.StepAction{Message: "done"}},
	}

	result := formatTrajectorySteps(steps)
	assert.Contains(t, result, "Plan: plan to search")
	assert.Contains(t, result, `Called tool "search"`)
	assert.Contains(t, result, "Result: found")
	assert.Contains(t, result, "Respond: here is the answer")
	assert.Contains(t, result, "Handoff to agent-b")
	assert.Contains(t, result, "Finish: done")
}

func TestFormatTrajectorySteps_Empty(t *testing.T) {
	assert.Equal(t, "(no steps)", formatTrajectorySteps(nil))
}

func TestFormatTrajectorySteps_Error(t *testing.T) {
	steps := []trajectory.Step{
		{Index: 0, Type: trajectory.StepToolCall, Action: trajectory.StepAction{ToolName: "search"}, Result: trajectory.StepResult{Error: "timeout"}},
	}
	result := formatTrajectorySteps(steps)
	assert.Contains(t, result, "Error: timeout")
}
