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

// Run executes the reasoning loop, yielding events as they occur.
// The loop: Plan → Execute actions → Observe results → Replan → repeat.
func (e *Executor) Run(ctx context.Context, input string, agentID string, tools []tool.Tool, messages []schema.Message, hooks Hooks) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		// Apply timeout
		if e.timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, e.timeout)
			defer cancel()
		}

		// Build tool registry from provided tools
		reg := tool.NewRegistry()
		for _, t := range tools {
			_ = reg.Add(t)
		}

		// Initialize planner state
		state := PlannerState{
			Input:    input,
			Messages: messages,
			Tools:    tools,
			Metadata: make(map[string]any),
		}

		// OnStart hook
		if hooks.OnStart != nil {
			if err := hooks.OnStart(ctx, input); err != nil {
				yield(Event{Type: EventError, AgentID: agentID}, err)
				return
			}
		}

		var finalResult string
		var finalErr error

		defer func() {
			if hooks.OnEnd != nil {
				hooks.OnEnd(ctx, finalResult, finalErr)
			}
		}()

		for i := 0; i < e.maxIterations; i++ {
			// Check context
			if err := ctx.Err(); err != nil {
				finalErr = fmt.Errorf("agent execution cancelled: %w", err)
				yield(Event{Type: EventError, AgentID: agentID}, finalErr)
				return
			}

			state.Iteration = i

			// BeforePlan hook
			if hooks.BeforePlan != nil {
				if err := hooks.BeforePlan(ctx, state); err != nil {
					finalErr = err
					yield(Event{Type: EventError, AgentID: agentID}, err)
					return
				}
			}

			// Plan or Replan
			var actions []Action
			var err error
			if i == 0 {
				actions, err = e.planner.Plan(ctx, state)
			} else {
				actions, err = e.planner.Replan(ctx, state)
			}
			if err != nil {
				if hooks.OnError != nil {
					err = hooks.OnError(ctx, err)
				}
				if err != nil {
					finalErr = err
					yield(Event{Type: EventError, AgentID: agentID}, err)
					return
				}
			}

			// AfterPlan hook
			if hooks.AfterPlan != nil {
				if err := hooks.AfterPlan(ctx, actions); err != nil {
					finalErr = err
					yield(Event{Type: EventError, AgentID: agentID}, err)
					return
				}
			}

			// Execute each action
			done := false
			for _, action := range actions {
				// BeforeAct hook
				if hooks.BeforeAct != nil {
					if err := hooks.BeforeAct(ctx, action); err != nil {
						finalErr = err
						yield(Event{Type: EventError, AgentID: agentID}, err)
						return
					}
				}

				obs := e.executeAction(ctx, agentID, action, reg, hooks, yield)

				// AfterAct hook
				if hooks.AfterAct != nil {
					if err := hooks.AfterAct(ctx, action, obs); err != nil {
						finalErr = err
						yield(Event{Type: EventError, AgentID: agentID}, err)
						return
					}
				}

				state.Observations = append(state.Observations, obs)

				if action.Type == ActionFinish {
					finalResult = action.Message
					done = true
					break
				}
				if action.Type == ActionHandoff {
					done = true
					break
				}
			}

			if done {
				if !yield(Event{Type: EventDone, AgentID: agentID, Text: finalResult}, nil) {
					return
				}
				return
			}

			// OnIteration hook
			if hooks.OnIteration != nil {
				if err := hooks.OnIteration(ctx, i); err != nil {
					finalErr = err
					yield(Event{Type: EventError, AgentID: agentID}, err)
					return
				}
			}

			// Update messages from observations for next iteration
			state.Messages = e.buildMessages(state)
		}

		// Max iterations reached
		finalErr = fmt.Errorf("agent reached maximum iterations (%d)", e.maxIterations)
		yield(Event{Type: EventError, AgentID: agentID}, finalErr)
	}
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
		if action.ToolCall == nil {
			return Observation{
				Action:  action,
				Error:   fmt.Errorf("tool action missing tool call"),
				Latency: time.Since(start),
			}
		}

		// Emit tool call event
		yield(Event{
			Type:     EventToolCall,
			AgentID:  agentID,
			ToolCall: action.ToolCall,
		}, nil)

		// OnToolCall hook
		callInfo := ToolCallInfo{
			Name:      action.ToolCall.Name,
			Arguments: action.ToolCall.Arguments,
			CallID:    action.ToolCall.ID,
		}
		if hooks.OnToolCall != nil {
			if err := hooks.OnToolCall(ctx, callInfo); err != nil {
				return Observation{
					Action:  action,
					Error:   err,
					Latency: time.Since(start),
				}
			}
		}

		// Look up and execute tool
		t, err := reg.Get(action.ToolCall.Name)
		if err != nil {
			result := tool.ErrorResult(fmt.Errorf("tool not found: %s", action.ToolCall.Name))
			yield(Event{
				Type:       EventToolResult,
				AgentID:    agentID,
				ToolResult: result,
			}, nil)
			return Observation{
				Action:  action,
				Result:  result,
				Error:   err,
				Latency: time.Since(start),
			}
		}

		// Parse arguments
		var args map[string]any
		if action.ToolCall.Arguments != "" {
			if err := json.Unmarshal([]byte(action.ToolCall.Arguments), &args); err != nil {
				result := tool.ErrorResult(fmt.Errorf("invalid tool arguments: %w", err))
				yield(Event{
					Type:       EventToolResult,
					AgentID:    agentID,
					ToolResult: result,
				}, nil)
				return Observation{
					Action:  action,
					Result:  result,
					Error:   err,
					Latency: time.Since(start),
				}
			}
		}

		result, err := t.Execute(ctx, args)
		if err != nil {
			result = tool.ErrorResult(err)
		}

		// OnToolResult hook
		if hooks.OnToolResult != nil {
			_ = hooks.OnToolResult(ctx, callInfo, result)
		}

		// Emit tool result event
		yield(Event{
			Type:       EventToolResult,
			AgentID:    agentID,
			ToolResult: result,
		}, nil)

		return Observation{
			Action:  action,
			Result:  result,
			Error:   err,
			Latency: time.Since(start),
		}

	case ActionRespond:
		yield(Event{
			Type:    EventText,
			AgentID: agentID,
			Text:    action.Message,
		}, nil)
		return Observation{
			Action:  action,
			Latency: time.Since(start),
		}

	case ActionFinish:
		yield(Event{
			Type:    EventText,
			AgentID: agentID,
			Text:    action.Message,
		}, nil)
		return Observation{
			Action:  action,
			Latency: time.Since(start),
		}

	case ActionHandoff:
		yield(Event{
			Type:    EventHandoff,
			AgentID: agentID,
			Text:    action.Message,
			Metadata: map[string]any{
				"target": action.Metadata["target"],
			},
		}, nil)
		return Observation{
			Action:  action,
			Latency: time.Since(start),
		}

	default:
		return Observation{
			Action:  action,
			Error:   fmt.Errorf("unknown action type: %s", action.Type),
			Latency: time.Since(start),
		}
	}
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
