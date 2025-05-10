package executor

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga/pkg/agents/base"
	"github.com/lookatitude/beluga/pkg/agents/tools"
	"github.com/lookatitude/beluga/pkg/schema"
)

// AgentExecutor is responsible for running an agent and executing its planned actions.
// It orchestrates the agent's interaction with tools and manages the flow of execution.
type AgentExecutor struct {
	Agent         base.Agent
	Tools         []tools.Tool
	MaxIterations int // Maximum number of iterations to prevent infinite loops
	// Add other fields like memory, callback handlers, etc.
}

// NewAgentExecutor creates a new AgentExecutor.
func NewAgentExecutor(agent base.Agent, agentTools []tools.Tool, maxIterations int) *AgentExecutor {
	return &AgentExecutor{
		Agent:         agent,
		Tools:         agentTools,
		MaxIterations: maxIterations,
	}
}

// Run executes the agent with the given input.
// It follows a plan-execute-observe loop until a final answer is reached or max iterations are exceeded.
func (e *AgentExecutor) Run(ctx context.Context, input map[string]interface{}) (schema.Message, error) {
	var intermediateSteps []schema.Message

	for i := 0; i < e.MaxIterations; i++ {
		actions, err := e.Agent.Plan(input, intermediateSteps)
		if err != nil {
			return nil, fmt.Errorf("error planning agent execution: %w", err)
		}

		// If no actions, assume agent has finished or needs to return a direct response
		if len(actions) == 0 {
			// This part might need more sophisticated handling based on agent's design
			// For now, we assume the last intermediate step or a default message is the output
			if len(intermediateSteps) > 0 {
				return intermediateSteps[len(intermediateSteps)-1], nil
			}
			return schema.NewMessage("No actions planned, agent finished.", schema.AIMessageType), nil
		}

		// For simplicity, this example executes the first action.
		// A more complex executor might handle multiple parallel actions or more sophisticated action selection.
		action := actions[0]

		// Find and execute the tool
		var toolOutput string
		toolFound := false
		for _, tool := range e.Tools {
			if tool.GetName() == action.ToolName {
				toolOutput, err = tool.Execute(ctx, action.ToolInput)
				if err != nil {
					// Handle tool execution error, maybe add as an observation
					intermediateSteps = append(intermediateSteps, schema.NewMessage(fmt.Sprintf("Error executing tool %s: %s", action.ToolName, err.Error()), schema.SystemMessageType))
					continue // Or break, depending on desired error handling
				}
				toolFound = true
				break
			}
		}

		if !toolFound {
			toolOutput = fmt.Sprintf("Tool %s not found.", action.ToolName)
			intermediateSteps = append(intermediateSteps, schema.NewMessage(toolOutput, schema.SystemMessageType))
		} else {
		    // Add tool output as an observation (ToolMessage)
		    toolMessage := schema.NewMessage(toolOutput, schema.ToolMessageType).(*schema.ToolMessage)
		    toolMessage.ToolCallID = action.ToolName // Or a more unique ID if available
		    toolMessage.Name = action.ToolName
		    intermediateSteps = append(intermediateSteps, toolMessage)
		}

		// Check if the agent considers itself done based on the last action/observation
		// This logic is highly dependent on the specific agent's implementation of Plan/Execute.
		// For a simple loop, we might just check if the action log indicates a final answer.
		if action.Log == "Final Answer" { // This is a simplistic check
			return schema.NewMessage(toolOutput, schema.AIMessageType), nil
		}
	}

	return nil, fmt.Errorf("agent exceeded maximum iterations (%d)", e.MaxIterations)
}

