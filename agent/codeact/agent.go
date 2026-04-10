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
	base          *agent.BaseAgent
	executor      CodeExecutor
	language      string
	timeout       time.Duration
	hooks         CodeActHooks
	planner       agent.Planner
	llm           llm.ChatModel
	maxIterations int
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
	planner        agent.Planner
	maxIterations  int
}

func defaultAgentOptions() agentOptions {
	return agentOptions{
		language:      "python",
		timeout:       30 * time.Second,
		maxIterations: 10,
	}
}

// WithPlanner sets a custom planner for the CodeActAgent. If unset, a
// CodeActPlanner is constructed from the configured LLM.
func WithPlanner(p agent.Planner) AgentOption {
	return func(o *agentOptions) {
		o.planner = p
	}
}

// WithMaxIterations sets the maximum number of plan-act-observe iterations.
func WithMaxIterations(n int) AgentOption {
	return func(o *agentOptions) {
		if n > 0 {
			o.maxIterations = n
		}
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
	var planner agent.Planner = o.planner
	if planner == nil && o.llm != nil {
		planner = NewCodeActPlanner(o.llm, plannerOpts...)
	}
	if planner != nil {
		baseOpts = append(baseOpts, agent.WithPlanner(planner))
	}
	if o.llm != nil {
		baseOpts = append(baseOpts, agent.WithLLM(o.llm))
	}
	baseOpts = append(baseOpts, o.agentOpts...)

	base := agent.New(id, baseOpts...)

	return &CodeActAgent{
		base:          base,
		executor:      executor,
		language:      o.language,
		timeout:       o.timeout,
		hooks:         o.hooks,
		planner:       planner,
		llm:           o.llm,
		maxIterations: o.maxIterations,
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

// Stream executes the agent and returns an iterator of events. It drives a
// plan-act-observe loop directly: when the planner emits an ActionCode action,
// the code block is executed via the configured CodeExecutor, a
// code_exec/code_result event pair is emitted, and the observation is fed
// back into the planner for the next iteration. All other action types are
// delegated to the BaseAgent for backward compatibility.
func (a *CodeActAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		// If we don't have a planner, fall back to the base agent behaviour.
		if a.planner == nil {
			for event, err := range a.base.Stream(ctx, input, opts...) {
				if !yield(event, err) {
					return
				}
				if err != nil {
					return
				}
			}
			return
		}

		// Build initial messages from persona + input.
		var messages []schema.Message
		if !a.base.Persona().IsEmpty() {
			if sysMsg := a.base.Persona().ToSystemMessage(); sysMsg != nil {
				messages = append(messages, sysMsg)
			}
		}
		messages = append(messages, schema.NewHumanMessage(input))

		state := agent.PlannerState{
			Input:    input,
			Messages: messages,
			Tools:    a.base.Tools(),
			Metadata: make(map[string]any),
		}

		for i := 0; i < a.maxIterations; i++ {
			if err := ctx.Err(); err != nil {
				yield(agent.Event{Type: agent.EventError, AgentID: a.ID()}, err)
				return
			}
			state.Iteration = i

			var actions []agent.Action
			var err error
			if i == 0 {
				actions, err = a.planner.Plan(ctx, state)
			} else {
				actions, err = a.planner.Replan(ctx, state)
			}
			if err != nil {
				yield(agent.Event{Type: agent.EventError, AgentID: a.ID()}, err)
				return
			}

			done := false
			for _, action := range actions {
				if action.Type == ActionCode {
					obs, stop := a.handleCodeAction(ctx, action, yield)
					state.Observations = append(state.Observations, obs)
					if stop {
						return
					}
					continue
				}

				switch action.Type {
				case agent.ActionFinish, agent.ActionRespond:
					if action.Message != "" {
						if !yield(agent.Event{Type: agent.EventText, AgentID: a.ID(), Text: action.Message}, nil) {
							return
						}
					}
					if action.Type == agent.ActionFinish {
						yield(agent.Event{Type: agent.EventDone, AgentID: a.ID(), Text: action.Message}, nil)
						done = true
					}
				default:
					// Unsupported action type in the codeact loop.
					yield(agent.Event{Type: agent.EventError, AgentID: a.ID(), Text: fmt.Sprintf("unsupported action type: %s", action.Type)}, fmt.Errorf("unsupported action type: %s", action.Type))
					return
				}
				if done {
					return
				}
			}
		}

		yield(agent.Event{Type: agent.EventError, AgentID: a.ID(), Text: "maximum iterations exceeded"}, fmt.Errorf("codeact agent: reached maximum iterations (%d)", a.maxIterations))
	}
}

// handleCodeAction executes a code action via the CodeExecutor and emits the
// code_exec and code_result events. It returns an Observation to be fed back
// into the planner and a boolean indicating whether the caller should stop
// iterating (e.g., because the consumer stopped the stream).
func (a *CodeActAgent) handleCodeAction(ctx context.Context, action agent.Action, yield func(agent.Event, error) bool) (agent.Observation, bool) {
	code, _ := action.Metadata["code"].(string)
	lang, _ := action.Metadata["language"].(string)
	if lang == "" {
		lang = a.language
	}

	if !yield(agent.Event{
		Type:    EventCodeExec,
		AgentID: a.ID(),
		Text:    code,
		Metadata: map[string]any{
			"language": lang,
		},
	}, nil) {
		return agent.Observation{Action: action}, true
	}

	start := time.Now()
	result, execErr := a.ExecuteCode(ctx, CodeAction{
		Language: lang,
		Code:     code,
		Timeout:  a.timeout,
	})
	latency := time.Since(start)

	toolResult := codeResultToToolResult(result)

	if !yield(agent.Event{
		Type:       EventCodeResult,
		AgentID:    a.ID(),
		ToolResult: toolResult,
		Metadata: map[string]any{
			"exit_code": result.ExitCode,
			"duration":  result.Duration,
		},
	}, nil) {
		return agent.Observation{Action: action, Result: toolResult, Error: execErr, Latency: latency}, true
	}

	return agent.Observation{
		Action:  action,
		Result:  toolResult,
		Error:   execErr,
		Latency: latency,
	}, false
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
