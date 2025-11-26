// Package agents provides a standardized registry pattern for agent creation.
// This follows the Beluga AI Framework design patterns with consistent factory interfaces.
package agents

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
	llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
)

// AgentCreatorFunc defines the function signature for creating agents.
type AgentCreatorFunc func(ctx context.Context, name string, llm any, agentTools []tools.Tool, config Config) (iface.CompositeAgent, error)

// AgentRegistry is the global registry for creating agent instances.
// It maintains a registry of available agent types and their creation functions.
type AgentRegistry struct {
	creators map[string]AgentCreatorFunc
	mu       sync.RWMutex
}

// NewAgentRegistry creates a new AgentRegistry instance.
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		creators: make(map[string]AgentCreatorFunc),
	}
}

// Register registers a new agent type with the registry.
func (r *AgentRegistry) Register(agentType string, creator AgentCreatorFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.creators[agentType] = creator
}

// Create creates a new agent instance using the registered agent type.
func (r *AgentRegistry) Create(ctx context.Context, agentType, name string, llm any, agentTools []tools.Tool, config Config) (iface.CompositeAgent, error) {
	r.mu.RLock()
	creator, exists := r.creators[agentType]
	r.mu.RUnlock()

	if !exists {
		return nil, NewAgentError(
			"create_agent",
			name,
			ErrCodeInitialization,
			fmt.Errorf("agent type '%s' not registered", agentType),
		)
	}
	return creator(ctx, name, llm, agentTools, config)
}

// ListAgentTypes returns a list of all registered agent type names.
func (r *AgentRegistry) ListAgentTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.creators))
	for name := range r.creators {
		names = append(names, name)
	}
	return names
}

// Global registry instance for easy access.
var globalAgentRegistry = NewAgentRegistry()

// RegisterAgentType registers an agent type with the global registry.
func RegisterAgentType(agentType string, creator AgentCreatorFunc) {
	globalAgentRegistry.Register(agentType, creator)
}

// CreateAgent creates an agent using the global registry.
func CreateAgent(ctx context.Context, agentType, name string, llm any, agentTools []tools.Tool, config Config) (iface.CompositeAgent, error) {
	return globalAgentRegistry.Create(ctx, agentType, name, llm, agentTools, config)
}

// ListAvailableAgentTypes returns all available agent types from the global registry.
func ListAvailableAgentTypes() []string {
	return globalAgentRegistry.ListAgentTypes()
}

// GetGlobalAgentRegistry returns the global registry instance for advanced usage.
func GetGlobalAgentRegistry() *AgentRegistry {
	return globalAgentRegistry
}

// Built-in agent type constants.
const (
	AgentTypeBase  = "base"
	AgentTypeReAct = "react"
)

// init registers the built-in agent types.
func init() {
	// Register built-in agent types
	RegisterAgentType(AgentTypeBase, createBaseAgent)
	RegisterAgentType(AgentTypeReAct, createReActAgent)
}

// Built-in agent creators.
func createBaseAgent(ctx context.Context, name string, llm any, agentTools []tools.Tool, config Config) (iface.CompositeAgent, error) {
	baseLLM, ok := llm.(llmsiface.LLM)
	if !ok {
		return nil, NewAgentError(
			"create_base_agent",
			name,
			ErrCodeInitialization,
			fmt.Errorf("base agent requires LLM interface, got %T", llm),
		)
	}

	factory := NewAgentFactory(&config)
	return factory.CreateBaseAgent(ctx, name, baseLLM, agentTools)
}

func createReActAgent(ctx context.Context, name string, llm any, agentTools []tools.Tool, config Config) (iface.CompositeAgent, error) {
	chatModel, ok := llm.(llmsiface.ChatModel)
	if !ok {
		return nil, NewAgentError(
			"create_react_agent",
			name,
			ErrCodeInitialization,
			fmt.Errorf("ReAct agent requires ChatModel interface, got %T", llm),
		)
	}

	factory := NewAgentFactory(&config)
	// ReAct agent needs a prompt, use default or from config
	var prompt any
	// TODO: Extract prompt from config or use default
	return factory.CreateReActAgent(ctx, name, chatModel, agentTools, prompt)
}
