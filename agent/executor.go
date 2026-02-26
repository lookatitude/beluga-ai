package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// Executor runs the Plan→Act→Observe reasoning loop. It is planner-agnostic:
// the same loop works for ReAct, Reflexion, or any other planner strategy.
type Executor struct {
	planner       Planner
	maxIterations int
	timeout       time.Duration
	hooks         Hooks
	toolRegistry  *tool.Registry
}

// ExecutorOption configures an Executor.
type ExecutorOption func(*Executor)

// WithExecutorPlanner sets the planner for the executor.
func WithExecutorPlanner(p Planner) ExecutorOption {
	return func(e *Executor) {
		e.planner = p
	}
}

// WithExecutorMaxIterations sets the maximum iterations for the executor.
func WithExecutorMaxIterations(n int) ExecutorOption {
	return func(e *Executor) {
		if n > 0 {
			e.maxIterations = n
		}
	}
}

// WithExecutorTimeout sets the timeout for the executor.
func WithExecutorTimeout(d time.Duration) ExecutorOption {
	return func(e *Executor) {
		if d > 0 {
			e.timeout = d
		}
	}
}

// WithExecutorHooks sets the hooks for the executor.
func WithExecutorHooks(h Hooks) ExecutorOption {
	return func(e *Executor) {
		e.hooks = h
	}
}

// NewExecutor creates a new Executor with the given options.
func NewExecutor(opts ...ExecutorOption) *Executor {
	e := &Executor{
		maxIterations: 10,
		timeout:       5 * time.Minute,
		toolRegistry:  tool.NewRegistry(),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// runContext holds the mutable state for a single execution of the reasoning loop.
type runContext struct {
	agentID     string
	hooks       Hooks
	reg         *tool.Registry
	state       PlannerState
	yield       func(Event, error) bool
	finalResult string
	finalErr    error
}

// yieldError emits an error event and records it as the final error.
func (rc *runContext) yieldError(err error) {
	rc.finalErr = err
	rc.yield(Event{Type: EventError, AgentID: rc.agentID}, err)
}

// Run executes the reasoning loop, yielding events as they occur.
// The loop: Plan → Execute actions → Observe results → Replan → repeat.
func (e *Executor) Run(ctx context.Context, input string, agentID string, tools []tool.Tool, messages []schema.Message, hooks Hooks) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		ctx, cancel := e.applyTimeout(ctx)
		if cancel != nil {
			defer cancel()
		}

		rc := e.newRunContext(agentID, tools, hooks, yield, input, messages)

		if hooks.OnStart != nil {
			if err := hooks.OnStart(ctx, input); err != nil {
				rc.yieldError(err)
				return
			}
		}

		defer func() {
			if hooks.OnEnd != nil {
				hooks.OnEnd(ctx, rc.finalResult, rc.finalErr)
			}
		}()

		e.runLoop(ctx, rc, agentID)
	}
}

// applyTimeout wraps ctx with a timeout if configured. Returns the context
// and a cancel func (nil if no timeout was applied).
func (e *Executor) applyTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if e.timeout > 0 {
		return context.WithTimeout(ctx, e.timeout)
	}
	return ctx, nil
}

// newRunContext initialises a runContext for a single execution.
func (e *Executor) newRunContext(agentID string, tools []tool.Tool, hooks Hooks, yield func(Event, error) bool, input string, messages []schema.Message) *runContext {
	reg := tool.NewRegistry()
	for _, t := range tools {
		_ = reg.Add(t)
	}
	return &runContext{
		agentID: agentID,
		hooks:   hooks,
		reg:     reg,
		yield:   yield,
		state: PlannerState{
			Input:    input,
			Messages: messages,
			Tools:    tools,
			Metadata: make(map[string]any),
		},
	}
}

// runLoop drives the plan-act-observe iterations.
func (e *Executor) runLoop(ctx context.Context, rc *runContext, agentID string) {
	for i := 0; i < e.maxIterations; i++ {
		if err := ctx.Err(); err != nil {
			rc.yieldError(fmt.Errorf("agent execution cancelled: %w", err))
			return
		}

		rc.state.Iteration = i

		done, shouldReturn := e.runIteration(ctx, rc, i)
		if shouldReturn {
			return
		}
		if done {
			rc.yield(Event{Type: EventDone, AgentID: agentID, Text: rc.finalResult}, nil)
			return
		}

		rc.state.Messages = e.buildMessages(rc.state)
	}

	rc.yieldError(fmt.Errorf("agent reached maximum iterations (%d)", e.maxIterations))
}

// runIteration executes a single plan-act-observe cycle. It returns (done, shouldReturn).
// done=true means the agent finished or handed off. shouldReturn=true means an error
// was yielded and the caller should return immediately.
func (e *Executor) runIteration(ctx context.Context, rc *runContext, i int) (bool, bool) {
	if rc.callHookBeforePlan(ctx) {
		return false, true
	}

	actions, err := e.planActions(ctx, rc.state, i)
	if err != nil {
		if rc.hooks.OnError != nil {
			err = rc.hooks.OnError(ctx, err)
		}
		if err != nil {
			rc.yieldError(err)
			return false, true
		}
	}

	if rc.callHookAfterPlan(ctx, actions) {
		return false, true
	}

	done, shouldReturn := e.executeActions(ctx, rc, actions)
	if shouldReturn {
		return false, true
	}
	if done {
		return true, false
	}

	if rc.callHookOnIteration(ctx, i) {
		return false, true
	}

	return false, false
}

// callHookBeforePlan invokes the BeforePlan hook if set. Returns true on error.
func (rc *runContext) callHookBeforePlan(ctx context.Context) bool {
	if rc.hooks.BeforePlan == nil {
		return false
	}
	if err := rc.hooks.BeforePlan(ctx, rc.state); err != nil {
		rc.yieldError(err)
		return true
	}
	return false
}

// callHookAfterPlan invokes the AfterPlan hook if set. Returns true on error.
func (rc *runContext) callHookAfterPlan(ctx context.Context, actions []Action) bool {
	if rc.hooks.AfterPlan == nil {
		return false
	}
	if err := rc.hooks.AfterPlan(ctx, actions); err != nil {
		rc.yieldError(err)
		return true
	}
	return false
}

// callHookOnIteration invokes the OnIteration hook if set. Returns true on error.
func (rc *runContext) callHookOnIteration(ctx context.Context, i int) bool {
	if rc.hooks.OnIteration == nil {
		return false
	}
	if err := rc.hooks.OnIteration(ctx, i); err != nil {
		rc.yieldError(err)
		return true
	}
	return false
}

// planActions calls Plan or Replan depending on the iteration.
func (e *Executor) planActions(ctx context.Context, state PlannerState, iteration int) ([]Action, error) {
	if iteration == 0 {
		return e.planner.Plan(ctx, state)
	}
	return e.planner.Replan(ctx, state)
}

// executeActions runs each action in sequence, returning (done, shouldReturn).
func (e *Executor) executeActions(ctx context.Context, rc *runContext, actions []Action) (bool, bool) {
	for _, action := range actions {
		done, shouldReturn := e.executeSingleAction(ctx, rc, action)
		if shouldReturn || done {
			return done, shouldReturn
		}
	}
	return false, false
}

// executeSingleAction runs one action with before/after hooks. Returns (done, shouldReturn).
func (e *Executor) executeSingleAction(ctx context.Context, rc *runContext, action Action) (bool, bool) {
	if rc.hooks.BeforeAct != nil {
		if err := rc.hooks.BeforeAct(ctx, action); err != nil {
			rc.yieldError(err)
			return false, true
		}
	}

	obs := e.executeAction(ctx, rc.agentID, action, rc.reg, rc.hooks, rc.yield)

	if rc.hooks.AfterAct != nil {
		if err := rc.hooks.AfterAct(ctx, action, obs); err != nil {
			rc.yieldError(err)
			return false, true
		}
	}

	rc.state.Observations = append(rc.state.Observations, obs)

	if action.Type == ActionFinish {
		rc.finalResult = action.Message
		return true, false
	}
	if action.Type == ActionHandoff {
		return true, false
	}
	return false, false
}

// executeAction handles a single action and returns an observation.
func (e *Executor) executeAction(
	ctx context.Context,
	agentID string,
	action Action,
	reg *tool.Registry,
	hooks Hooks,
	yield func(Event, error) bool,
) Observation {
	start := time.Now()

	switch action.Type {
	case ActionTool:
		return e.executeToolAction(ctx, agentID, action, reg, hooks, yield, start)

	case ActionRespond, ActionFinish:
		yield(Event{
			Type:    EventText,
			AgentID: agentID,
			Text:    action.Message,
		}, nil)
		return Observation{Action: action, Latency: time.Since(start)}

	case ActionHandoff:
		yield(Event{
			Type:    EventHandoff,
			AgentID: agentID,
			Text:    action.Message,
			Metadata: map[string]any{
				"target": action.Metadata["target"],
			},
		}, nil)
		return Observation{Action: action, Latency: time.Since(start)}

	default:
		return Observation{
			Action:  action,
			Error:   fmt.Errorf("unknown action type: %s", action.Type),
			Latency: time.Since(start),
		}
	}
}

// executeToolAction handles ActionTool: looks up the tool, parses arguments,
// executes it, and emits the appropriate events.
func (e *Executor) executeToolAction(
	ctx context.Context,
	agentID string,
	action Action,
	reg *tool.Registry,
	hooks Hooks,
	yield func(Event, error) bool,
	start time.Time,
) Observation {
	if action.ToolCall == nil {
		return Observation{
			Action:  action,
			Error:   fmt.Errorf("tool action missing tool call"),
			Latency: time.Since(start),
		}
	}

	yield(Event{Type: EventToolCall, AgentID: agentID, ToolCall: action.ToolCall}, nil)

	callInfo := ToolCallInfo{
		Name:      action.ToolCall.Name,
		Arguments: action.ToolCall.Arguments,
		CallID:    action.ToolCall.ID,
	}
	if hooks.OnToolCall != nil {
		if err := hooks.OnToolCall(ctx, callInfo); err != nil {
			return Observation{Action: action, Error: err, Latency: time.Since(start)}
		}
	}

	result, err := e.lookupAndExecuteTool(ctx, action, reg)

	if hooks.OnToolResult != nil {
		_ = hooks.OnToolResult(ctx, callInfo, result)
	}

	yield(Event{Type: EventToolResult, AgentID: agentID, ToolResult: result}, nil)

	return Observation{Action: action, Result: result, Error: err, Latency: time.Since(start)}
}

// lookupAndExecuteTool resolves the tool, parses args, and executes it.
func (e *Executor) lookupAndExecuteTool(
	ctx context.Context,
	action Action,
	reg *tool.Registry,
) (*tool.Result, error) {
	t, err := reg.Get(action.ToolCall.Name)
	if err != nil {
		return tool.ErrorResult(fmt.Errorf("tool not found: %s", action.ToolCall.Name)), err
	}

	var args map[string]any
	if action.ToolCall.Arguments != "" {
		if err := json.Unmarshal([]byte(action.ToolCall.Arguments), &args); err != nil {
			return tool.ErrorResult(fmt.Errorf("invalid tool arguments: %w", err)), err
		}
	}

	result, err := t.Execute(ctx, args)
	if err != nil {
		return tool.ErrorResult(err), err
	}
	return result, nil
}

// buildMessages converts the current state (input + observations) into
// messages for the next LLM call.
func (e *Executor) buildMessages(state PlannerState) []schema.Message {
	msgs := make([]schema.Message, 0, len(state.Messages)+len(state.Observations)*2)
	msgs = append(msgs, state.Messages...)

	for _, obs := range state.Observations {
		if obs.Action.Type == ActionTool && obs.Action.ToolCall != nil {
			// Add the AI message with tool call
			msgs = append(msgs, &schema.AIMessage{
				ToolCalls: []schema.ToolCall{*obs.Action.ToolCall},
			})

			// Add the tool result message
			var text string
			if obs.Result != nil {
				text = extractResultText(obs.Result)
			} else if obs.Error != nil {
				text = obs.Error.Error()
			}
			msgs = append(msgs, schema.NewToolMessage(obs.Action.ToolCall.ID, text))
		}
	}

	return msgs
}

// extractResultText extracts text content from a tool result.
func extractResultText(result *tool.Result) string {
	var parts []string
	for _, part := range result.Content {
		if tp, ok := part.(schema.TextPart); ok {
			parts = append(parts, tp.Text)
		}
	}
	return strings.Join(parts, "\n")
}
