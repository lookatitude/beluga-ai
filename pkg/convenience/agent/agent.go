// Package agent provides a simplified API for creating agents with memory and tools.
// It reduces boilerplate by providing a fluent builder pattern for agent configuration.
//
// Note: This package is a work in progress. For production use, please use the
// agents package directly (github.com/lookatitude/beluga-ai/pkg/agents).
//
// Example intended usage (future):
//
//	agent, err := agent.NewBuilder().
//	    WithLLM(llm).
//	    WithBufferMemory(50).
//	    WithTool(calculator).
//	    WithSystemPrompt("You are helpful").
//	    Build(ctx)
//
//	result, err := agent.Run(ctx, "Calculate 2+2")
package agent

// Builder provides a fluent interface for constructing agents.
// This is a placeholder for future implementation.
type Builder struct {
	systemPrompt string
	name         string
	maxTurns     int
	verbose      bool
	agentType    string
}

// NewBuilder creates a new agent builder.
func NewBuilder() *Builder {
	return &Builder{
		name:      "assistant",
		maxTurns:  10,
		agentType: "react",
	}
}

// WithSystemPrompt sets the system prompt for the agent.
func (b *Builder) WithSystemPrompt(prompt string) *Builder {
	b.systemPrompt = prompt
	return b
}

// WithName sets the agent name.
func (b *Builder) WithName(name string) *Builder {
	b.name = name
	return b
}

// WithMaxTurns sets the maximum number of turns.
func (b *Builder) WithMaxTurns(max int) *Builder {
	b.maxTurns = max
	return b
}

// WithVerbose enables verbose output.
func (b *Builder) WithVerbose(verbose bool) *Builder {
	b.verbose = verbose
	return b
}

// WithAgentType sets the agent type (e.g., "react", "tool_calling").
func (b *Builder) WithAgentType(agentType string) *Builder {
	b.agentType = agentType
	return b
}

// GetSystemPrompt returns the configured system prompt.
func (b *Builder) GetSystemPrompt() string {
	return b.systemPrompt
}

// GetName returns the configured agent name.
func (b *Builder) GetName() string {
	return b.name
}

// GetMaxTurns returns the configured max turns.
func (b *Builder) GetMaxTurns() int {
	return b.maxTurns
}

// GetAgentType returns the configured agent type.
func (b *Builder) GetAgentType() string {
	return b.agentType
}
