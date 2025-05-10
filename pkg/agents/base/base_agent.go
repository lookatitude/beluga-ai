package base

import (
	"fmt"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

// Agent defines the interface for an AI agent.
// It outlines the core functionalities that any agent should implement.
type Agent interface {
	// Plan generates a sequence of actions based on the input and current state.
	Plan(input map[string]interface{}, intermediateSteps []schema.Message) ([]tools.ToolAgentAction, error)

	// Execute performs the actions determined by the Plan method.
	// It interacts with tools and manages the agent's state.
	Execute(actions []tools.ToolAgentAction) (schema.Message, error)

	// GetInputKeys returns the expected input keys for the agent.
	GetInputKeys() []string

	// GetOutputKeys returns the expected output keys from the agent.
	GetOutputKeys() []string

	// GetTools returns the list of tools available to the agent.
	GetTools() []tools.Tool

	// GetName returns the name of the agent.
	GetName() string
}

// BaseAgent provides a foundational structure for agents.
// It can be embedded in specific agent implementations to provide common functionality.
type BaseAgent struct {
	Name        string
	Tools       []tools.Tool
	InputKeys   []string
	OutputKeys  []string
}

// NewBaseAgent creates a new BaseAgent.
func NewBaseAgent(name string, agentTools []tools.Tool, inputKeys []string, outputKeys []string) *BaseAgent {
	return &BaseAgent{
		Name:       name,
		Tools:      agentTools,
		InputKeys:  inputKeys,
		OutputKeys: outputKeys,
	}
}

// GetTools returns the tools available to the agent.
func (a *BaseAgent) GetTools() []tools.Tool {
	return a.Tools
}

// GetName returns the name of the agent.
func (a *BaseAgent) GetName() string {
	return a.Name
}

// GetInputKeys returns the expected input keys for the agent.
func (a *BaseAgent) GetInputKeys() []string {
	return a.InputKeys
}

// GetOutputKeys returns the expected output keys from the agent.
func (a *BaseAgent) GetOutputKeys() []string {
	return a.OutputKeys
}




// Execute performs the actions determined by the Plan method.
// This is a placeholder and should be implemented by specific agent types
// that embed BaseAgent or by BaseAgent itself if it has a default execution logic.
func (a *BaseAgent) Execute(actions []tools.ToolAgentAction) (schema.Message, error) {
	// Placeholder implementation. Specific agents should override this.
	return schema.NewMessage("BaseAgent Execute method not implemented", schema.SystemMessageType), nil
}




// Plan generates a sequence of actions based on the input and current state.
// This is a placeholder and should be implemented by specific agent types
// that embed BaseAgent or by BaseAgent itself if it has a default planning logic.
func (a *BaseAgent) Plan(input map[string]interface{}, intermediateSteps []schema.Message) ([]tools.ToolAgentAction, error) {
	// Placeholder implementation. Specific agents should override this.
	return nil, fmt.Errorf("BaseAgent Plan method not implemented")
}

