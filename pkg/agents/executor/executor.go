package executor

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/agents/base"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

// Executor defines the interface for an agent executor.
// An executor is responsible for taking a plan (a sequence of steps)
// and running it, interacting with tools and the LLM as needed.	ype Executor interface {
	ExecutePlan(ctx context.Context, agent base.Agent, plan []schema.Step) (schema.FinalAnswer, error)
}

// AgentExecutor is a concrete implementation of the Executor interface.
// It iterates through the steps of a plan, executes actions (tool calls or LLM interactions),
// and collects observations to produce a final answer.	ype AgentExecutor struct {
	// Configuration for the executor, e.g., max iterations (already in AgentConfig),
	// error handling strategies, etc. For now, we keep it simple.
	// We can add specific executor configurations if needed later.
}

// NewAgentExecutor creates a new AgentExecutor.
func NewAgentExecutor() *AgentExecutor {
	return &AgentExecutor{}
}

// ExecutePlan runs the given plan for the specified agent.
func (e *AgentExecutor) ExecutePlan(ctx context.Context, agent base.Agent, plan []schema.Step) (schema.FinalAnswer, error) {
	var intermediateSteps []schema.Step
	var currentInputs map[string]interface{} // This might need to be initialized from agent.InputKeys() or passed in.
	// For simplicity, we assume the plan itself contains all necessary context or the agent handles it.

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

		// Is this an action step (tool call)?
		if step.Action.Tool != "" {
			toolToUse, ok := availableTools[step.Action.Tool]
			if !ok {
				observation.Output = fmt.Sprintf("Error: Tool 	%s	 not found.", step.Action.Tool)
				currentStep.Observation = observation
				intermediateSteps = append(intermediateSteps, currentStep)
				// Potentially, the agent's Plan method should re-plan based on this observation.
				// For a simple executor, we might just continue or error out.
				// For now, we will let the agent re-plan in the next iteration if this executor is called within a loop by the agent itself.
				// This executor is more about executing a *given* plan.
				// If the plan is fixed, this is a fatal error for the plan execution.
				return schema.FinalAnswer{}, fmt.Errorf("tool '%s' specified in plan not found in agent's tools", step.Action.Tool)
			}

			// TODO: Add input validation against tool.GetInputSchema()
			toolInputStr := ""
			if strInput, ok := step.Action.ToolInput.(string); ok {
				toolInputStr = strInput
			} else {
				// Handle more complex inputs if necessary, e.g., by serializing map[string]interface{} to JSON
				toolInputStr = fmt.Sprintf("%v", step.Action.ToolInput)
			}

			toolOutput, err := toolToUse.Execute(ctx, toolInputStr) // Assuming tool input is string for now
			if err != nil {
				observation.Output = fmt.Sprintf("Error executing tool %s: %v", step.Action.Tool, err)
			} else {
				observation.Output = toolOutput
			}
			observation.ActionLog = step.Action.Log // Carry over the log from the action
		} else {
			// This step is not a tool call, implies it might be a direct LLM call for a thought or final answer generation.
			// The current `schema.Step` doesn't explicitly differentiate this well from a tool action with an empty tool name.
			// For now, we assume if Tool is empty, the Action.Log might contain a prompt or the Observation.Output is expected to be set by the planner.
			// This part needs refinement based on how different agent types (ReAct, PlanAndExecute) structure their plans.
			// If the plan step already has an observation, use it. Otherwise, this step might be ill-defined for this executor.
			if step.Observation.Output != "" {
				observation = step.Observation
			} else if step.Action.Log != "" && llm != nil {
				// Assuming the Action.Log is a prompt for the LLM if no tool is specified.
				// This is a simplistic interpretation; a more robust system would have explicit step types.
				llmResponse, err := llm.Invoke(ctx, step.Action.Log) // Using Invoke for simplicity
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

		// Check if this observation should be considered the final answer.
		// This logic is highly dependent on the agent type and its prompting.
		// For a generic executor, we might look for a specific signal or assume the last step's observation is final.
		// A more robust agent would have its Plan method determine if the goal is met.
		// For now, we'll assume the plan is structured such that the last step's observation is the basis for the final answer.
	}

	if len(intermediateSteps) == 0 {
		return schema.FinalAnswer{Output: "No steps were executed or plan was empty."}, nil
	}

	// The final answer is typically the output of the last observation in the executed plan.
	// More sophisticated agents might parse this or use an LLM to synthesize a final answer.
	finalOutput := intermediateSteps[len(intermediateSteps)-1].Observation.Output
	return schema.FinalAnswer{
		Output:            finalOutput,
		IntermediateSteps: intermediateSteps,
	}, nil
}

