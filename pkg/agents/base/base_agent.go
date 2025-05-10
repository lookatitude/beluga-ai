package base

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

// BaseAgent provides a foundational struct that can be embedded by concrete agent implementations.
// It handles common agent functionalities like managing configuration, LLM, tools, and memory.
type BaseAgent struct {	config schema.AgentConfig
	llm    llms.LLM
	tools  []tools.Tool
	memory memory.Memory
	// executor is responsible for running the agent's plan. It's not part of the Agent interface directly
	// but is a common dependency for the Execute method.
	// It will be set up by the factory or a specific agent implementation.
	// For now, we acknowledge its conceptual presence here.
}

// NewBaseAgent creates a new BaseAgent instance.
// Note: The executor is not passed here; it's typically set up by a factory or a more specific agent constructor.
func NewBaseAgent(config schema.AgentConfig, llm llms.LLM, tools []tools.Tool, mem memory.Memory) *BaseAgent {
	return &BaseAgent{
		config: config,
		llm:    llm,
		tools:  tools,
		memory: mem,
	}
}

// GetConfig returns the agent's configuration.
func (a *BaseAgent) GetConfig() schema.AgentConfig {
	return a.config
}

// GetLLM returns the LLM instance used by the agent.
func (a *BaseAgent) GetLLM() llms.LLM {
	return a.llm
}

// GetTools returns the list of tools available to the agent.
func (a *BaseAgent) GetTools() []tools.Tool {
	return a.tools
}

// GetMemory returns the memory module used by the agent.
func (a *BaseAgent) GetMemory() memory.Memory {
	return a.memory
}

// InputKeys returns the expected input keys for the agent.
// This is a placeholder and should be overridden by specific agent implementations
// based on their prompting strategy or requirements.
func (a *BaseAgent) InputKeys() []string {
	// Default behavior, can be overridden
	return []string{"input"} // A common default input key
}

// OutputKeys returns the expected output keys from the agent.
// This is a placeholder and should be overridden by specific agent implementations.
func (a *BaseAgent) OutputKeys() []string {
	// Default behavior, can be overridden
	return []string{"output"} // A common default output key
}

// Plan generates a sequence of steps (actions or direct LLM calls) based on the input and intermediate steps.
// This is a placeholder implementation and MUST be overridden by concrete agent types.
func (a *BaseAgent) Plan(ctx context.Context, inputs map[string]interface{}, intermediateSteps []schema.Step) ([]schema.Step, error) {
	// Concrete agent implementations (e.g., ReActAgent, ConversationalAgent) will define their planning logic here.
	// This might involve formatting a prompt with inputs, tools, and history, then calling the LLM.
	return nil, fmt.Errorf("Plan method not implemented in BaseAgent; must be overridden by specific agent type")
}

// Execute runs the planned steps.
// This is a placeholder implementation. Typically, this method would delegate to an AgentExecutor.
// Concrete agent implementations might orchestrate this differently or directly use an executor.
func (a *BaseAgent) Execute(ctx context.Context, steps []schema.Step) (schema.FinalAnswer, error) {
	// This method will typically be implemented by an AgentExecutor that the agent uses.
	// For a BaseAgent, it might be abstract or delegate if an executor is directly associated.
	// For now, returning an error indicating it needs proper implementation or delegation.
	return schema.FinalAnswer{}, fmt.Errorf("Execute method not implemented in BaseAgent or not delegated to an executor; must be handled by specific agent type or its executor")
}

