// Package agents defines interfaces and implementations for AI agents.
package base
import (
	"context"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

// AgentAction represents an action to be taken by the agent.
type AgentAction struct {
	Tool      string // The name of the tool to use
	ToolInput any    // The input to the tool (can be string, map[string]any, etc.) - Changed to any
	Log       string // Additional logging information (e.g., the LLM's thought process)
}

// AgentFinish represents the final response from the agent.
type AgentFinish struct {
	ReturnValues map[string]any // Final output values
	Log          string         // Additional logging information
}

// Agent is the interface for agents.
type Agent interface {
	// InputVariables returns the expected input keys for the agent.
	InputVariables() []string

	// OutputVariables returns the keys for the agent's final output.
	OutputVariables() []string

	// GetTools returns the tools available to the agent.
	GetTools() []tools.Tool

	// Plan decides the next action or finish state based on intermediate steps and inputs.
	Plan(ctx context.Context, intermediateSteps []struct {
		Action      AgentAction
		Observation string
	}, inputs map[string]any) (AgentAction, AgentFinish, error)

	// TODO: Add methods for streaming?
}

// Ensure Agent implements the Runnable interface (conceptual check)
// var _ core.Runnable = (Agent)(nil) // This won't compile directly, needs adapter or different interface design
