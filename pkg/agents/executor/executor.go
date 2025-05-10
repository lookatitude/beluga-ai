package executor

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/agents/base"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// Executor defines the interface for an agent executor.
// An executor is responsible for taking a plan (a sequence of steps)
// and running it, interacting with tools and the LLM as needed.
type Executor interface {
	ExecutePlan(ctx context.Context, agent base.Agent, plan []schema.Step) (schema.FinalAnswer, error)
}

// AgentExecutor is a concrete implementation of the Executor interface.
// It iterates through the steps of a plan, executes actions (tool calls or LLM interactions),
// and collects observations to produce a final answer.
type AgentExecutor struct {
	// Configuration for the executor, e.g., max iterations (already in AgentConfig),
	// error handling strategies, etc. For now, we keep it simple.
}

// NewAgentExecutor creates a new AgentExecutor.
func NewAgentExecutor() *AgentExecutor {
	return &AgentExecutor{}
}

// ExecutePlan runs the given plan for the specified agent.
func (e *AgentExecutor) ExecutePlan(ctx context.Context, agent base.Agent, plan []schema.Step) (schema.FinalAnswer, error) {
	var intermediateSteps []schema.Step
	// Removed unused: var currentInputs map[string]interface{}

	agentConfig := agent.GetConfig()
	llm := agent.GetLLM()
	availableTools := make(map[string]tools.Tool)
	for _, t := range agent.GetTools() {
		availableTools[t.GetName()] = t
	}

	for i, step := range plan {
		if agentConfig.MaxIterations > 0 && i >= agentConfig.MaxIterations {
			return schema.FinalAnswer{}, fmt.Errorf("reached max iterations (%d) without a final answer", agentConfig.MaxIterations)
		}

		var observation schema.AgentObservation
		currentStep := schema.Step{Action: step.Action}

		if step.Action.Tool != "" {
			toolToUse, ok := availableTools[step.Action.Tool]
			if !ok {
				return schema.FinalAnswer{}, fmt.Errorf("tool 	%s	 specified in plan not found in agent's tools", step.Action.Tool)
			}

			var toolInputMap map[string]interface{}
			if ti, typeOk := step.Action.ToolInput.(map[string]interface{}); typeOk {
				toolInputMap = ti
			} else {
				// If ToolInput is not nil and not a map, it's an unexpected format for tools that expect maps.
				// If ToolInput is nil, and the tool expects input, this will also be an issue handled by the tool itself or schema validation.
				if step.Action.ToolInput != nil { // Only error if it's non-nil but wrong type
					return schema.FinalAnswer{}, fmt.Errorf("tool 	%s	 received invalid input format: expected map[string]interface{}, got %T: %v", step.Action.Tool, step.Action.ToolInput, step.Action.ToolInput)
				}
				// If step.Action.ToolInput is nil, toolInputMap will remain nil, and the tool must handle it (e.g. if it expects no input or specific optional keys)
			}

			// TODO: Add input validation against toolToUse.GetInputSchema() before execution

			toolOutput, err := toolToUse.Execute(ctx, toolInputMap)
			if err != nil {
				observation.Output = fmt.Sprintf("Error executing tool %s: %v", step.Action.Tool, err)
			} else {
				observation.Output = toolOutput
			}
			observation.ActionLog = step.Action.Log // Carry over the log from the action
		} else {
			// This step is not a tool call, implies it might be a direct LLM call for a thought or final answer generation.
			if step.Observation.Output != "" {
				observation = step.Observation
			} else if step.Action.Log != "" && llm != nil {
				llmResponse, err := llm.Invoke(ctx, step.Action.Log)
				if err != nil {
					observation.Output = fmt.Sprintf("Error from LLM: %v", err)
				} else {
					observation.Output = llmResponse
				}
				observation.ActionLog = step.Action.Log
			} else {
				observation.Output = "No tool specified and no pre-defined observation or LLM prompt found for this step."
			}
		}

		currentStep.Observation = observation
		intermediateSteps = append(intermediateSteps, currentStep)
	}

	if len(intermediateSteps) == 0 {
		return schema.FinalAnswer{Output: "No steps were executed or plan was empty."}, nil
	}

	finalOutput := intermediateSteps[len(intermediateSteps)-1].Observation.Output
	return schema.FinalAnswer{
		Output:            finalOutput,
		IntermediateSteps: intermediateSteps,
	}, nil
}

