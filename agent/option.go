package agent

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/tool"
)

// Option is a functional option for configuring an agent.
type Option func(*agentConfig)

// Memory is the interface for agent memory persistence. Implementations
// provide the ability to save and load conversation history.
type Memory interface {
	// Save persists messages for the given session.
	Save(ctx context.Context, sessionID string, messages []any) error
	// Load retrieves persisted messages for the given session.
	Load(ctx context.Context, sessionID string) ([]any, error)
}

// agentConfig holds the configuration applied by options.
type agentConfig struct {
	llm           llm.ChatModel
	tools         []tool.Tool
	persona       Persona
	maxIterations int
	timeout       time.Duration
	hooks         Hooks
	handoffs      []Handoff
	memory        Memory
	planner       Planner
	plannerName   string
	children      []Agent
	metadata      map[string]any
}

// defaultConfig returns the default agent configuration.
func defaultConfig() agentConfig {
	return agentConfig{
		maxIterations: 10,
		timeout:       5 * time.Minute,
		plannerName:   "react",
	}
}

// WithLLM sets the language model for the agent.
func WithLLM(model llm.ChatModel) Option {
	return func(c *agentConfig) {
		c.llm = model
	}
}

// WithTools sets the tools available to the agent.
func WithTools(tools []tool.Tool) Option {
	return func(c *agentConfig) {
		c.tools = tools
	}
}

// WithPersona sets the agent's persona.
func WithPersona(p Persona) Option {
	return func(c *agentConfig) {
		c.persona = p
	}
}

// WithMaxIterations sets the maximum number of reasoning loop iterations.
func WithMaxIterations(n int) Option {
	return func(c *agentConfig) {
		if n > 0 {
			c.maxIterations = n
		}
	}
}

// WithTimeout sets the maximum execution duration for the agent.
func WithTimeout(d time.Duration) Option {
	return func(c *agentConfig) {
		if d > 0 {
			c.timeout = d
		}
	}
}

// WithHooks sets the lifecycle hooks for the agent.
func WithHooks(h Hooks) Option {
	return func(c *agentConfig) {
		c.hooks = h
	}
}

// WithHandoffs sets the handoff targets for the agent.
func WithHandoffs(handoffs []Handoff) Option {
	return func(c *agentConfig) {
		c.handoffs = handoffs
	}
}

// WithMemory sets the memory backend for the agent.
func WithMemory(m Memory) Option {
	return func(c *agentConfig) {
		c.memory = m
	}
}

// WithPlanner sets the planner directly (bypasses registry lookup).
func WithPlanner(p Planner) Option {
	return func(c *agentConfig) {
		c.planner = p
	}
}

// WithPlannerName sets the planner by registered name (e.g., "react", "reflexion").
func WithPlannerName(name string) Option {
	return func(c *agentConfig) {
		c.plannerName = name
	}
}

// WithChildren sets child agents for orchestration.
func WithChildren(children []Agent) Option {
	return func(c *agentConfig) {
		c.children = children
	}
}

// WithMetadata sets arbitrary metadata on the agent.
func WithMetadata(meta map[string]any) Option {
	return func(c *agentConfig) {
		c.metadata = meta
	}
}
