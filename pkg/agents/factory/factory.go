package factory

import (
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/agents/base"
	"github.com/lookatitude/beluga-ai/pkg/llms"
	"github.com/lookatitude/beluga-ai/pkg/memory"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	"github.com/lookatitude/beluga-ai/pkg/config" // Assuming global config access or specific provider configs
)

// AgentFactory defines the interface for creating agent instances.	ype AgentFactory interface {
	CreateAgent(ctx context.Context, agentConfig schema.AgentConfig) (base.Agent, error)
}

// ConcreteAgentFactory is a concrete implementation of AgentFactory.
// It requires factories for LLMs, Memory, and a ToolRegistry to assemble agents.	ype ConcreteAgentFactory struct {
	llmFactory    llms.LLMFactory // To get LLM instances based on config
	memoryFactory memory.Factory  // To get Memory instances based on config
	toolRegistry  tools.Registry  // To get Tool instances by name
	// We might also need access to global LLMProviderConfigs and MemoryProviderConfigs
	// if the AgentConfig only refers to them by name.
	// For simplicity, let's assume the LLMFactory and MemoryFactory handle this lookup internally
	// or the AgentConfig itself contains enough detail for them.
	// Let's also assume a global map of LLMProviderConfig and MemoryConfig for now, passed via a config provider.
	cfgProvider config.Provider // To access global LLM and Memory configurations by name
}

// NewConcreteAgentFactory creates a new ConcreteAgentFactory.
func NewConcreteAgentFactory(llmFact llms.LLMFactory, memFact memory.Factory, toolReg tools.Registry, cfgProvider config.Provider) *ConcreteAgentFactory {
	return &ConcreteAgentFactory{
		llmFactory:    llmFact,
		memoryFactory: memFact,
		toolRegistry:  toolReg,
		cfgProvider:   cfgProvider,
	}
}

// CreateAgent creates a new agent instance based on the provided AgentConfig.
func (f *ConcreteAgentFactory) CreateAgent(ctx context.Context, agentConfig schema.AgentConfig) (base.Agent, error) {
	// 1. Get LLM instance
	// The AgentConfig should specify which LLMProviderConfig to use by name.
	llmProviderCfg, err := f.cfgProvider.GetLLMProviderConfig(agentConfig.LLMProviderConfigName)
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM provider config 	%s	: %w", agentConfig.LLMProviderConfigName, err)
	}
	llmInstance, err := f.llmFactory.CreateLLM(ctx, llmProviderCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM instance for agent 	%s	: %w", agentConfig.Name, err)
	}

	// 2. Get Tool instances
	var agentTools []tools.Tool
	for _, toolName := range agentConfig.ToolNames {
		toolInstance, err := f.toolRegistry.GetTool(toolName)
		if err != nil {
			return nil, fmt.Errorf("failed to get tool 	%s	 for agent 	%s	: %w", toolName, agentConfig.Name, err)
		}
		agentTools = append(agentTools, toolInstance)
	}

	// 3. Get Memory instance
	// The AgentConfig should specify MemoryType and potentially a MemoryConfigName.
	// The MemoryFactory will handle creating the correct memory type.
	// For now, let's assume MemoryConfigName refers to a global memory configuration if needed (e.g., for VectorStoreMemory).
	var memoryInstance memory.Memory
	if agentConfig.MemoryType != "" {
		// We need a way to pass memory-specific config to the memory factory.
		// This might involve fetching a MemoryProviderConfig similar to LLMProviderConfig.
		// For now, let's assume the MemoryFactory can handle it with just the type and a potential name.
		// This part needs to be more robust based on how MemoryFactory is designed.
		// Let's assume a simplified MemoryFactory.CreateMemory(type, configName) for now.
		// Or, the AgentConfig itself could embed the memory configuration details.

		// Placeholder: This needs a proper MemoryConfig retrieval and passing mechanism.
		// For instance, if MemoryType is "vector", MemoryConfigName might point to a VectorStoreConfig.
		memCfg := memory.Config{ // This is a placeholder for actual memory config structure
			Type: agentConfig.MemoryType,
			Name: agentConfig.MemoryConfigName, // This name might be used by the factory to fetch detailed config
		}
		memoryInstance, err = f.memoryFactory.CreateMemory(ctx, memCfg) // Assuming MemoryFactory.CreateMemory exists
		if err != nil {
			return nil, fmt.Errorf("failed to create memory instance (	%s	) for agent 	%s	: %w", agentConfig.MemoryType, agentConfig.Name, err)
		}
	} else {
		// Default to no memory or a basic buffer memory if not specified
		// For now, let's assume nil memory if not specified, or the agent handles it.
		// Or, create a default BufferMemory.
		// This depends on the desired default behavior.
		// Let's assume for now that if MemoryType is empty, no specific memory is configured via factory.
		// The BaseAgent constructor might initialize a default if `nil` is passed.
		// For clarity, let's require MemoryType to be specified if memory is desired.
		// If agentConfig.MemoryType is empty, memoryInstance will be nil.
	}

	// 4. Create the Agent instance (currently BaseAgent)
	// In the future, we might have different agent types (ReAct, Conversational, etc.)
	// and the factory would decide which concrete agent struct to instantiate based on AgentConfig.Type or similar.
	// For now, we create a BaseAgent.
	// The BaseAgent itself doesn't directly use an executor in its constructor.
	// The executor is typically used *by* the agent or an orchestrator *with* the agent.
	newAgent := base.NewBaseAgent(agentConfig, llmInstance, agentTools, memoryInstance)

	// TODO: Potentially set up an executor for the agent here if the factory is responsible for fully initializing it.
	// However, the Agent interface's Execute method might be what the executor calls.
	// For now, the BaseAgent's Execute method is a placeholder.

	return newAgent, nil
}

