// Package executor provides agent execution implementations.
// It handles the execution loop, tool calling, and result processing.
package executor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// AgentExecutor implements the Executor interface.
// It manages the execution loop for agents, handling planning, tool execution, and result collection.
type AgentExecutor struct {
	maxIterations           int
	returnIntermediateSteps bool
	handleParsingErrors     bool
	// TODO: Add metrics interface when needed
}

// NewAgentExecutor creates a new agent executor with default settings.
func NewAgentExecutor(opts ...ExecutorOption) *AgentExecutor {
	executor := &AgentExecutor{
		maxIterations:           15,
		returnIntermediateSteps: false,
		handleParsingErrors:     true,
	}

	// Apply options
	for _, opt := range opts {
		opt(executor)
	}

	return executor
}

// ExecutorOption represents a functional option for configuring the executor.
type ExecutorOption func(*AgentExecutor)

// WithMaxIterations sets the maximum number of execution iterations.
func WithMaxIterations(max int) ExecutorOption {
	return func(e *AgentExecutor) {
		e.maxIterations = max
	}
}

// WithReturnIntermediateSteps configures whether to return intermediate steps.
func WithReturnIntermediateSteps(returnSteps bool) ExecutorOption {
	return func(e *AgentExecutor) {
		e.returnIntermediateSteps = returnSteps
	}
}

// WithHandleParsingErrors configures error handling behavior.
func WithHandleParsingErrors(handle bool) ExecutorOption {
	return func(e *AgentExecutor) {
		e.handleParsingErrors = handle
	}
}

// ExecutePlan executes the given plan for the specified agent.
// It iterates through the plan steps, executing tools and collecting results.
func (e *AgentExecutor) ExecutePlan(ctx context.Context, agent iface.Agent, plan []schema.Step) (schema.FinalAnswer, error) {
	start := time.Now()

	if len(plan) == 0 {
		return schema.FinalAnswer{Output: "No steps to execute"}, nil
	}

	var intermediateSteps []iface.IntermediateStep

	// Execute each step in the plan
	for i, step := range plan {
		if e.maxIterations > 0 && i >= e.maxIterations {
			if agent.GetMetrics() != nil {
				agent.GetMetrics().RecordExecutorRun(ctx, "agent_executor", time.Since(start), i, false)
			}
			return schema.FinalAnswer{}, fmt.Errorf("execution failed for agent %s: maximum iterations (%d) exceeded", agent.GetConfig().Name, e.maxIterations)
		}

		// Execute the step
		observation, err := e.executeStep(ctx, agent, step)
		if err != nil {
			if agent.GetMetrics() != nil {
				agent.GetMetrics().RecordExecutorRun(ctx, "agent_executor", time.Since(start), i, false)
			}
			return schema.FinalAnswer{}, fmt.Errorf("execution failed for agent %s at step %d: %w", agent.GetConfig().Name, i, err)
		}

		// Record intermediate step
		intermediateStep := iface.IntermediateStep{
			Action:      iface.AgentAction(step.Action),
			Observation: observation,
		}
		intermediateSteps = append(intermediateSteps, intermediateStep)
	}

	// Create final answer
	finalOutput := ""
	if len(intermediateSteps) > 0 {
		finalOutput = intermediateSteps[len(intermediateSteps)-1].Observation
	}

	result := schema.FinalAnswer{
		Output: finalOutput,
	}

	if e.returnIntermediateSteps {
		result.IntermediateSteps = convertToSchemaSteps(intermediateSteps)
	}

	if agent.GetMetrics() != nil {
		agent.GetMetrics().RecordExecutorRun(ctx, "agent_executor", time.Since(start), len(plan), true)
	}

	return result, nil
}

// executeStep executes a single step in the plan.
func (e *AgentExecutor) executeStep(ctx context.Context, agent iface.Agent, step schema.Step) (string, error) {
	// If step has a tool, execute it
	if step.Action.Tool != "" {
		return e.executeTool(ctx, agent, step.Action)
	}

	// If step has pre-defined observation, return it
	if step.Observation.Output != "" {
		return step.Observation.Output, nil
	}

	// If step has log content, treat it as direct LLM call
	if step.Action.Log != "" {
		llm := agent.GetLLM()
		if llm == nil {
			return "", errors.New("step requires LLM but agent does not have one")
		}

		result, err := llm.Invoke(ctx, step.Action.Log)
		if err != nil {
			return "", fmt.Errorf("LLM call failed: %w", err)
		}
		return fmt.Sprintf("%v", result), nil
	}

	return "No action or observation defined for this step", nil
}

// executeTool executes a tool with the given action.
func (e *AgentExecutor) executeTool(ctx context.Context, agent iface.Agent, action schema.AgentAction) (string, error) {
	var selectedTool tools.Tool

	// Find the tool by name
	for _, tool := range agent.GetTools() {
		if tool.Name() == action.Tool {
			selectedTool = tool
			break
		}
	}

	if selectedTool == nil {
		return "", fmt.Errorf("tool '%s' not found in agent's available tools", action.Tool)
	}

	// Record tool call start
	toolStart := time.Now()

	// Execute the tool
	result, err := selectedTool.Execute(ctx, action.ToolInput)

	if agent.GetMetrics() != nil {
		agent.GetMetrics().RecordToolCall(ctx, action.Tool, time.Since(toolStart), err == nil)
	}

	if err != nil {
		return "", fmt.Errorf("tool execution failed: %w", err)
	}

	// Convert result to string
	resultStr := fmt.Sprintf("%v", result)
	return resultStr, nil
}

// convertToSchemaSteps converts internal intermediate steps to schema steps.
func convertToSchemaSteps(intermediateSteps []iface.IntermediateStep) []schema.Step {
	steps := make([]schema.Step, len(intermediateSteps))
	for i, step := range intermediateSteps {
		steps[i] = schema.Step{
			Action: schema.AgentAction(step.Action),
			Observation: schema.AgentObservation{
				Output: step.Observation,
			},
		}
	}
	return steps
}

// Ensure AgentExecutor implements the Executor interface.
var _ iface.Executor = (*AgentExecutor)(nil)
