package factory

import (
	"fmt"

	"github.com/lookatitude/beluga/pkg/agents/base"
	"github.com/lookatitude/beluga/pkg/agents/tools"
	// Potentially import specific agent implementations here if the factory creates them directly
)

// AgentFactory is responsible for creating instances of agents.
// This allows for decoupling agent creation logic from the rest of the application.
type AgentFactory struct {
	// Configuration for the factory, e.g., default tools, model configurations, etc.
	// For simplicity, this example is kept minimal.
}

// NewAgentFactory creates a new AgentFactory.
func NewAgentFactory() *AgentFactory {
	return &AgentFactory{}
}

// CreateAgent creates an agent of a specific type with the given configuration.
// This is a simplified example. A real factory might take more specific configuration
// or have different methods for different agent types.
func (f *AgentFactory) CreateAgent(agentType string, name string, agentTools []tools.Tool, inputKeys []string, outputKeys []string) (base.Agent, error) {
	switch agentType {
	case "simple": // Example agent type
		// Here you would instantiate a specific agent implementation
		// For now, we return a BaseAgent as a placeholder.
		// In a real scenario, you might have: return specificagent.NewSimpleAgent(...), error
		return base.NewBaseAgent(name, agentTools, inputKeys, outputKeys), nil
	// Add cases for other agent types
	// case "react":
	// return reactagent.NewReActAgent(name, agentTools, inputKeys, outputKeys, llmInstance), nil
	default:
		return nil, fmt.Errorf("unknown agent type: %s", agentType)
	}
}

// GetDefaultTools could be a helper method to provide a default set of tools for agents.
func (f *AgentFactory) GetDefaultTools() []tools.Tool {
	// Example: return a list of common tools
	return []tools.Tool{}
}

