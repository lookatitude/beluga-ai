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
// The registry manages agent type registration and creation following the factory pattern.
//
// Returns:
//   - *AgentRegistry: A new agent registry instance
//
// Example:
//
//	registry := agents.NewAgentRegistry()
//	registry.Register("base", baseAgentCreator)
//
// Example usage can be found in examples/agents/basic/main.go
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		creators: make(map[string]AgentCreatorFunc),
	}
}

// Register registers a new agent type with the registry.
// This method is thread-safe and allows extending the framework with custom agent types.
//
// Parameters:
//   - agentType: Unique identifier for the agent type (e.g., "base", "react")
//   - creator: Function that creates agent instances of this type
//
// Example:
//
//	registry.Register("custom", func(ctx context.Context, name string, llm any, tools []tools.Tool, config agents.Config) (iface.CompositeAgent, error) {
//	    return NewCustomAgent(name, llm, tools, config)
//	})
//
// Example usage can be found in examples/agents/basic/main.go
func (r *AgentRegistry) Register(agentType string, creator AgentCreatorFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.creators[agentType] = creator
}

// Create creates a new agent instance using the registered agent type.
// This method is thread-safe and returns an error if the agent type is not registered.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - agentType: Type of agent to create (must be registered)
//   - name: Unique name for the agent instance
//   - llm: Language model instance (LLM or ChatModel)
//   - agentTools: Slice of tools available to the agent
//   - config: Agent configuration
//
// Returns:
//   - iface.CompositeAgent: A new agent instance
//   - error: ErrCodeInitialization if agent type is not registered, or agent creation errors
//
// Example:
//
//	agent, err := registry.Create(ctx, "base", "my-agent", llm, tools, config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Example usage can be found in examples/agents/basic/main.go
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
// This method is thread-safe and returns an empty slice if no types are registered.
//
// Returns:
//   - []string: Slice of registered agent type names
//
// Example:
//
//	types := registry.ListAgentTypes()
//	fmt.Printf("Available agent types: %v\n", types)
//
// Example usage can be found in examples/agents/basic/main.go
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
// This is a convenience function for registering with the global registry.
//
// Parameters:
//   - agentType: Unique identifier for the agent type
//   - creator: Function that creates agent instances of this type
//
// Example:
//
//	agents.RegisterAgentType("custom", customAgentCreator)
//
// Example usage can be found in examples/agents/basic/main.go
func RegisterAgentType(agentType string, creator AgentCreatorFunc) {
	globalAgentRegistry.Register(agentType, creator)
}

// CreateAgent creates an agent using the global registry.
// This is a convenience function for creating agents with the global registry.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - agentType: Type of agent to create (must be registered)
//   - name: Unique name for the agent instance
//   - llm: Language model instance
//   - agentTools: Slice of tools available to the agent
//   - config: Agent configuration
//
// Returns:
//   - iface.CompositeAgent: A new agent instance
//   - error: Agent type not found or creation errors
//
// Example:
//
//	agent, err := agents.CreateAgent(ctx, "base", "my-agent", llm, tools, config)
//
// Example usage can be found in examples/agents/basic/main.go
func CreateAgent(ctx context.Context, agentType, name string, llm any, agentTools []tools.Tool, config Config) (iface.CompositeAgent, error) {
	return globalAgentRegistry.Create(ctx, agentType, name, llm, agentTools, config)
}

// ListAvailableAgentTypes returns all available agent types from the global registry.
// This is a convenience function for listing types from the global registry.
//
// Returns:
//   - []string: Slice of available agent type names
//
// Example:
//
//	types := agents.ListAvailableAgentTypes()
//	fmt.Printf("Available types: %v\n", types)
//
// Example usage can be found in examples/agents/basic/main.go
func ListAvailableAgentTypes() []string {
	return globalAgentRegistry.ListAgentTypes()
}

// GetGlobalAgentRegistry returns the global registry instance for advanced usage.
// Deprecated: Use GetRegistry() instead for consistency.
func GetGlobalAgentRegistry() *AgentRegistry {
	return globalAgentRegistry
}

// GetRegistry returns the global registry instance.
// This follows the standard pattern used across all Beluga AI packages.
//
// Example:
//
//	registry := agents.GetRegistry()
//	agentTypes := registry.ListAgentTypes()
//	agent, err := registry.CreateAgent(ctx, "my-agent", "base", llm, tools, config)
func GetRegistry() *AgentRegistry {
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
