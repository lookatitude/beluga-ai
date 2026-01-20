package planexecute

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	llms "github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockPlanExecuteAgent(t *testing.T) {
	mockLLM := llms.NewAdvancedMockChatModel("test-model")
	tools := []tools.Tool{} // Empty tools for testing

	t.Run("default_behavior", func(t *testing.T) {
		mock, err := NewMockPlanExecuteAgent("test-agent", mockLLM, tools)
		require.NoError(t, err)

		// Test Plan method
		action, finish, err := mock.Plan(context.Background(), nil, map[string]any{"input": "test input"})
		assert.NoError(t, err)
		assert.Equal(t, iface.AgentAction{}, action) // No action for default finish
		assert.Contains(t, finish.ReturnValues, "output")
		assert.Equal(t, 1, mock.GetCallCount())
	})

	t.Run("error_injection", func(t *testing.T) {
		expectedErr := errors.New("mock error")
		mock, err := NewMockPlanExecuteAgent("test-agent", mockLLM, tools,
			WithMockPlanExecuteError(true, expectedErr))
		require.NoError(t, err)

		// Test Plan method returns error
		action, finish, err := mock.Plan(context.Background(), nil, map[string]any{"input": "test input"})
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, iface.AgentAction{}, action)
		assert.Equal(t, iface.AgentFinish{}, finish)
		assert.Equal(t, 1, mock.GetCallCount())
	})

	t.Run("delay_simulation", func(t *testing.T) {
		delay := 50 * time.Millisecond
		mock, err := NewMockPlanExecuteAgent("test-agent", mockLLM, tools,
			WithMockPlanExecuteDelay(delay))
		require.NoError(t, err)

		start := time.Now()
		_, _, err = mock.Plan(context.Background(), nil, map[string]any{"input": "test input"})
		duration := time.Since(start)

		assert.NoError(t, err)
		assert.True(t, duration >= delay, "Delay should be respected")
		assert.Equal(t, 1, mock.GetCallCount())
	})

	t.Run("predefined_plan_steps", func(t *testing.T) {
		steps := []PlanStep{
			{StepNumber: 1, Action: "test_action_1", Tool: "test_tool_1", Input: "input_1", Reasoning: "reasoning_1"},
			{StepNumber: 2, Action: "test_action_2", Tool: "test_tool_2", Input: "input_2", Reasoning: "reasoning_2"},
		}
		mock, err := NewMockPlanExecuteAgent("test-agent", mockLLM, tools,
			WithMockPlanSteps(steps))
		require.NoError(t, err)

		action, finish, err := mock.Plan(context.Background(), nil, map[string]any{"input": "test input"})
		assert.NoError(t, err)
		assert.Equal(t, iface.AgentFinish{}, finish)
		assert.Equal(t, "ExecutePlan", action.Tool)
		assert.Contains(t, action.ToolInput, "plan")
		assert.Equal(t, 1, mock.GetCallCount())

		// Verify plan steps
		returnedSteps := mock.GetPlanSteps()
		assert.Len(t, returnedSteps, 2)
		assert.Equal(t, steps[0].Action, returnedSteps[0].Action)
		assert.Equal(t, steps[1].Tool, returnedSteps[1].Tool)
	})

	t.Run("predefined_execution_results", func(t *testing.T) {
		results := []map[string]any{
			{"step_1": "result_1"},
			{"step_2": "result_2"},
		}
		mock, err := NewMockPlanExecuteAgent("test-agent", mockLLM, tools,
			WithMockExecutionResults(results))
		require.NoError(t, err)

		plan := &ExecutionPlan{
			Goal:       "test goal",
			Steps:      []PlanStep{{StepNumber: 1}, {StepNumber: 2}},
			TotalSteps: 2,
		}

		executionResults, err := mock.ExecutePlan(context.Background(), plan)
		assert.NoError(t, err)
		assert.Contains(t, executionResults, "step_1")
		assert.Contains(t, executionResults, "step_2")
		assert.Equal(t, 2, executionResults["total_steps"])
		assert.Equal(t, 1, mock.GetCallCount())

		// Verify execution results
		returnedResults := mock.GetExecutionResults()
		assert.Len(t, returnedResults, 2)
		assert.Equal(t, results[0], returnedResults[0])
	})

	t.Run("call_count_tracking", func(t *testing.T) {
		mock, err := NewMockPlanExecuteAgent("test-agent", mockLLM, tools)
		require.NoError(t, err)

		// Initial call count should be 0
		assert.Equal(t, 0, mock.GetCallCount())

		// Call Plan method
		mock.Plan(context.Background(), nil, map[string]any{"input": "test"})
		assert.Equal(t, 1, mock.GetCallCount())

		// Call ExecutePlan method
		plan := &ExecutionPlan{Steps: []PlanStep{{StepNumber: 1}}}
		mock.ExecutePlan(context.Background(), plan)
		assert.Equal(t, 2, mock.GetCallCount())
	})

	t.Run("context_cancellation_with_delay", func(t *testing.T) {
		delay := 100 * time.Millisecond
		mock, err := NewMockPlanExecuteAgent("test-agent", mockLLM, tools,
			WithMockPlanExecuteDelay(delay))
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, _, err = mock.Plan(ctx, nil, map[string]any{"input": "test input"})
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}