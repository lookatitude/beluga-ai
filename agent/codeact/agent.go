package codeact

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// EventCodeExec is the event type for code execution events.
const EventCodeExec agent.EventType = "code_exec"

// EventCodeResult is the event type for code execution result events.
const EventCodeResult agent.EventType = "code_result"

// CodeActAgent wraps a BaseAgent with a CodeExecutor. It intercepts ActionCode
// actions from the planner and routes them to the executor instead of the
// standard tool execution path.
type CodeActAgent struct {
	base     *agent.BaseAgent
	executor CodeExecutor
	language string
	timeout  time.Duration
	hooks    CodeActHooks
}

// Compile-time interface check.
var _ agent.Agent = (*CodeActAgent)(nil)

// AgentOption configures a CodeActAgent.
type AgentOption func(*agentOptions)

type agentOptions struct {
	executor       CodeExecutor
	language       string
	timeout        time.Duration
	hooks          CodeActHooks
	agentOpts      []agent.Option
	llm            llm.ChatModel
	allowedImports []string
}

func defaultAgentOptions() agentOptions {
	return agentOptions{
		language: "python",
		timeout:  30 * time.Second,
	}
}

// WithExecutor sets the code executor for the agent.
func WithExecutor(e CodeExecutor) AgentOption {
	return func(o *agentOptions) {
		o.executor = e
	}
}

// WithLanguage sets the preferred programming language.
func WithLanguage(lang string) AgentOption {
	return func(o *agentOptions) {
		o.language = lang
	}
}

// WithExecTimeout sets the default timeout for code execution.
func WithExecTimeout(d time.Duration) AgentOption {
	return func(o *agentOptions) {
		if d > 0 {
			o.timeout = d
		}
	}
}

// WithCodeActHooks sets lifecycle hooks for code execution.
func WithCodeActHooks(h CodeActHooks) AgentOption {
	return func(o *agentOptions) {
		o.hooks = h
	}
}

// WithAgentLLM sets the LLM for the underlying agent.
func WithAgentLLM(model llm.ChatModel) AgentOption {
	return func(o *agentOptions) {
		o.llm = model
	}
}

// WithAgentOption passes through an option to the underlying BaseAgent.
func WithAgentOption(opt agent.Option) AgentOption {
	return func(o *agentOptions) {
		o.agentOpts = append(o.agentOpts, opt)
	}
}

// WithAllowedCodeImports sets allowed imports for the planner.
func WithAllowedCodeImports(imports []string) AgentOption {
	return func(o *agentOptions) {
		o.allowedImports = imports
	}
}

// NewCodeActAgent creates a new CodeActAgent with the given ID and options.
func NewCodeActAgent(id string, opts ...AgentOption) *CodeActAgent {
	o := defaultAgentOptions()
	for _, opt := range opts {
		opt(&o)
	}

	// Default to NoopExecutor if none provided
	executor := o.executor
	if executor == nil {
		executor = NewNoopExecutor()
	}

	// Build planner options
	var plannerOpts []PlannerOption
	plannerOpts = append(plannerOpts, WithPlannerLanguage(o.language))
	if len(o.allowedImports) > 0 {
		plannerOpts = append(plannerOpts, WithAllowedImports(o.allowedImports))
	}

	// Build base agent options
	baseOpts := make([]agent.Option, 0, len(o.agentOpts)+3)
	if o.llm != nil {
		planner := NewCodeActPlanner(o.llm, plannerOpts...)
		baseOpts = append(baseOpts, agent.WithPlanner(planner))
		baseOpts = append(baseOpts, agent.WithLLM(o.llm))
	}
	baseOpts = append(baseOpts, o.agentOpts...)

	base := agent.New(id, baseOpts...)

	return &CodeActAgent{
		base:     base,
		executor: executor,
		language: o.language,
		timeout:  o.timeout,
		hooks:    o.hooks,
	}
}

// ID returns the agent's unique identifier.
func (a *CodeActAgent) ID() string { return a.base.ID() }

// Persona returns the agent's persona.
func (a *CodeActAgent) Persona() agent.Persona { return a.base.Persona() }

// Tools returns the tools available to the agent.
func (a *CodeActAgent) Tools() []tool.Tool { return a.base.Tools() }

// Children returns child agents.
func (a *CodeActAgent) Children() []agent.Agent { return a.base.Children() }

// Invoke executes the agent synchronously and returns the final text result.
func (a *CodeActAgent) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	var result strings.Builder
	var lastErr error

	for event, err := range a.Stream(ctx, input, opts...) {
		if err != nil {
			lastErr = err
			break
		}
		switch event.Type {
		case agent.EventText, agent.EventDone:
			result.WriteString(event.Text)
		case agent.EventError:
			lastErr = fmt.Errorf("agent error: %s", event.Text)
		}
	}

	if lastErr != nil {
		return result.String(), lastErr
	}
	return result.String(), nil
}

// Stream executes the agent and returns an iterator of events.
// It intercepts ActionCode events from the base agent's stream, executes the
// code via the CodeExecutor, and injects the results back.
func (a *CodeActAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		for event, err := range a.base.Stream(ctx, input, opts...) {
			if err != nil {
				yield(event, err)
				return
			}

			// Intercept tool result events that carry code execution metadata
			// The executor in the base agent treats unknown action types by
			// returning an error observation. We hook into BeforeAct via the
			// agent hooks to intercept ActionCode.
			if !yield(event, nil) {
				return
			}
		}
	}
}

// ExecuteCode runs a code action through the executor with hooks.
func (a *CodeActAgent) ExecuteCode(ctx context.Context, action CodeAction) (CodeResult, error) {
	if action.Language == "" {
		action.Language = a.language
	}
	if action.Timeout == 0 {
		action.Timeout = a.timeout
	}

	// BeforeExec hook
	if a.hooks.BeforeExec != nil {
		if err := a.hooks.BeforeExec(ctx, action); err != nil {
			return CodeResult{}, err
		}
	}

	result, err := a.executor.Execute(ctx, action)

	// OnError hook
	if err != nil && a.hooks.OnError != nil {
		err = a.hooks.OnError(ctx, action, err)
	}

	if err != nil {
		return result, err
	}

	// AfterExec hook
	if a.hooks.AfterExec != nil {
		if hookErr := a.hooks.AfterExec(ctx, action, result); hookErr != nil {
			return result, hookErr
		}
	}

	return result, nil
}

// codeResultToToolResult converts a CodeResult into a tool.Result for the
// agent executor's observation flow.
func codeResultToToolResult(result CodeResult) *tool.Result {
	var text string
	if result.Success() {
		text = result.Output
	} else {
		text = fmt.Sprintf("Exit code: %d\nStdout: %s\nStderr: %s", result.ExitCode, result.Output, result.Error)
	}
	return &tool.Result{
		Content: []schema.ContentPart{
			schema.TextPart{Text: text},
		},
	}
}
