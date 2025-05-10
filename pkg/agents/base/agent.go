package base

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

// Agent defines the interface for an AI agent.
// Agents are responsible for planning and executing tasks to achieve a goal.
type Agent interface {	// Plan generates a sequence of steps (actions or direct LLM calls) based on the input and intermediate steps.
	Plan(ctx context.Context, inputs map[string]interface{}, intermediateSteps []schema.Step) ([]schema.Step, error)

	// Execute runs the planned steps. This is typically delegated to an AgentExecutor.
	// It should return the final answer or an error if execution fails.
	Execute(ctx context.Context, steps []schema.Step) (schema.FinalAnswer, error)

	// GetTools returns the list of tools available to the agent.
	GetTools() []tools.Tool

	// GetMemory returns the memory module used by the agent.
	GetMemory() memory.Memory

	// GetConfig returns the agent's configuration.
	GetConfig() schema.AgentConfig

	// GetLLM returns the LLM instance used by the agent.
	GetLLM() llms.LLM

	// InputKeys returns the expected input keys for the agent.
	InputKeys() []string

	// OutputKeys returns the expected output keys from the agent.
	OutputKeys() []string
}

