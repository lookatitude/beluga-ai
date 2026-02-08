package agent

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// BaseAgent is the default agent implementation. It can be used directly or
// embedded in custom agent types. BaseAgent wires together a planner, executor,
// tools, handoffs, and hooks into a working agent.
type BaseAgent struct {
	id       string
	config   agentConfig
	executor *Executor
}

// New creates a new BaseAgent with the given ID and options.
func New(id string, opts ...Option) *BaseAgent {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	a := &BaseAgent{
		id:     id,
		config: cfg,
	}

	return a
}

// ID returns the agent's unique identifier.
func (a *BaseAgent) ID() string { return a.id }

// Persona returns the agent's persona.
func (a *BaseAgent) Persona() Persona { return a.config.persona }

// Tools returns the tools available to the agent, including handoff tools.
func (a *BaseAgent) Tools() []tool.Tool {
	tools := make([]tool.Tool, 0, len(a.config.tools)+len(a.config.handoffs))
	tools = append(tools, a.config.tools...)
	if len(a.config.handoffs) > 0 {
		tools = append(tools, HandoffsToTools(a.config.handoffs)...)
	}
	return tools
}

// Children returns child agents.
func (a *BaseAgent) Children() []Agent { return a.config.children }

// Invoke executes the agent synchronously and returns the final text result.
func (a *BaseAgent) Invoke(ctx context.Context, input string, opts ...Option) (string, error) {
	// Apply runtime options
	cfg := a.config
	for _, opt := range opts {
		opt(&cfg)
	}

	var result strings.Builder
	var lastErr error

	for event, err := range a.stream(ctx, input, cfg) {
		if err != nil {
			lastErr = err
			break
		}
		switch event.Type {
		case EventText:
			result.WriteString(event.Text)
		case EventError:
			lastErr = fmt.Errorf("agent error: %s", event.Text)
		}
	}

	if lastErr != nil {
		return result.String(), lastErr
	}
	return result.String(), nil
}

// Stream executes the agent and returns an iterator of events.
func (a *BaseAgent) Stream(ctx context.Context, input string, opts ...Option) iter.Seq2[Event, error] {
	cfg := a.config
	for _, opt := range opts {
		opt(&cfg)
	}
	return a.stream(ctx, input, cfg)
}

// stream is the internal streaming implementation.
func (a *BaseAgent) stream(ctx context.Context, input string, cfg agentConfig) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		// Resolve planner
		planner, err := a.resolvePlanner(cfg)
		if err != nil {
			yield(Event{Type: EventError, AgentID: a.id}, err)
			return
		}

		// Build messages
		messages := a.buildInitialMessages(input, cfg)

		// Build tools list including handoffs
		tools := make([]tool.Tool, 0, len(cfg.tools)+len(cfg.handoffs))
		tools = append(tools, cfg.tools...)
		if len(cfg.handoffs) > 0 {
			tools = append(tools, HandoffsToTools(cfg.handoffs)...)
		}

		// Create executor
		executor := NewExecutor(
			WithExecutorPlanner(planner),
			WithExecutorMaxIterations(cfg.maxIterations),
			WithExecutorTimeout(cfg.timeout),
			WithExecutorHooks(cfg.hooks),
		)

		// Run the executor
		for event, err := range executor.Run(ctx, input, a.id, tools, messages, cfg.hooks) {
			if !yield(event, err) {
				return
			}
			if err != nil {
				return
			}
		}
	}
}

// resolvePlanner returns the planner to use, resolving from config.
func (a *BaseAgent) resolvePlanner(cfg agentConfig) (Planner, error) {
	if cfg.planner != nil {
		return cfg.planner, nil
	}
	if cfg.llm == nil {
		return nil, fmt.Errorf("agent %q: no LLM configured", a.id)
	}
	return NewPlanner(cfg.plannerName, PlannerConfig{
		LLM: cfg.llm,
	})
}

// buildInitialMessages constructs the initial message list from persona and input.
func (a *BaseAgent) buildInitialMessages(input string, cfg agentConfig) []schema.Message {
	var messages []schema.Message

	// Add persona as system message
	if !cfg.persona.IsEmpty() {
		if sysMsg := cfg.persona.ToSystemMessage(); sysMsg != nil {
			messages = append(messages, sysMsg)
		}
	}

	// Add user input
	messages = append(messages, schema.NewHumanMessage(input))

	return messages
}

// Config returns the agent's configuration (for testing and extension).
func (a *BaseAgent) Config() agentConfig { return a.config }
