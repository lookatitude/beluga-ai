package agent

import (
	"context"

	"github.com/lookatitude/beluga-ai/tool"
)

// Hooks provides optional callback functions invoked at various points
// during agent execution. All fields are optional; nil hooks are skipped.
// Hooks are composable via ComposeHooks.
type Hooks struct {
	// OnStart is called when the agent begins execution with the input.
	OnStart func(ctx context.Context, input string) error
	// OnEnd is called when the agent finishes execution with the result.
	OnEnd func(ctx context.Context, result string, err error)
	// OnError is called when an error occurs. The returned error replaces the
	// original; returning nil suppresses the error.
	OnError func(ctx context.Context, err error) error

	// BeforePlan is called before the planner generates actions.
	BeforePlan func(ctx context.Context, state PlannerState) error
	// AfterPlan is called after the planner generates actions.
	AfterPlan func(ctx context.Context, actions []Action) error

	// BeforeAct is called before an action is executed.
	BeforeAct func(ctx context.Context, action Action) error
	// AfterAct is called after an action is executed.
	AfterAct func(ctx context.Context, action Action, obs Observation) error

	// OnToolCall is called when a tool is about to be invoked.
	OnToolCall func(ctx context.Context, call ToolCallInfo) error
	// OnToolResult is called after a tool returns a result.
	OnToolResult func(ctx context.Context, call ToolCallInfo, result *tool.Result) error

	// OnIteration is called at the end of each reasoning loop iteration.
	OnIteration func(ctx context.Context, iteration int) error

	// OnHandoff is called when a handoff to another agent occurs.
	OnHandoff func(ctx context.Context, from, to string) error

	// BeforeGenerate is called before an LLM generation.
	BeforeGenerate func(ctx context.Context) error
	// AfterGenerate is called after an LLM generation.
	AfterGenerate func(ctx context.Context) error
}

// ToolCallInfo carries information about a tool call for hooks.
type ToolCallInfo struct {
	// Name is the tool name.
	Name string
	// Arguments is the JSON-encoded arguments.
	Arguments string
	// CallID is the unique identifier for this call.
	CallID string
}

func composeOnStart(hooks []Hooks) func(context.Context, string) error {
	return func(ctx context.Context, input string) error {
		for _, h := range hooks {
			if h.OnStart != nil {
				if err := h.OnStart(ctx, input); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeOnEnd(hooks []Hooks) func(context.Context, string, error) {
	return func(ctx context.Context, result string, err error) {
		for _, h := range hooks {
			if h.OnEnd != nil {
				h.OnEnd(ctx, result, err)
			}
		}
	}
}

func composeOnError(hooks []Hooks) func(context.Context, error) error {
	return func(ctx context.Context, err error) error {
		for _, h := range hooks {
			if h.OnError != nil {
				if e := h.OnError(ctx, err); e != nil {
					return e
				}
			}
		}
		return err
	}
}

func composeBeforePlan(hooks []Hooks) func(context.Context, PlannerState) error {
	return func(ctx context.Context, state PlannerState) error {
		for _, h := range hooks {
			if h.BeforePlan != nil {
				if err := h.BeforePlan(ctx, state); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeAfterPlan(hooks []Hooks) func(context.Context, []Action) error {
	return func(ctx context.Context, actions []Action) error {
		for _, h := range hooks {
			if h.AfterPlan != nil {
				if err := h.AfterPlan(ctx, actions); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeBeforeAct(hooks []Hooks) func(context.Context, Action) error {
	return func(ctx context.Context, action Action) error {
		for _, h := range hooks {
			if h.BeforeAct != nil {
				if err := h.BeforeAct(ctx, action); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeAfterAct(hooks []Hooks) func(context.Context, Action, Observation) error {
	return func(ctx context.Context, action Action, obs Observation) error {
		for _, h := range hooks {
			if h.AfterAct != nil {
				if err := h.AfterAct(ctx, action, obs); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeOnToolCall(hooks []Hooks) func(context.Context, ToolCallInfo) error {
	return func(ctx context.Context, call ToolCallInfo) error {
		for _, h := range hooks {
			if h.OnToolCall != nil {
				if err := h.OnToolCall(ctx, call); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeOnToolResult(hooks []Hooks) func(context.Context, ToolCallInfo, *tool.Result) error {
	return func(ctx context.Context, call ToolCallInfo, result *tool.Result) error {
		for _, h := range hooks {
			if h.OnToolResult != nil {
				if err := h.OnToolResult(ctx, call, result); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeOnIteration(hooks []Hooks) func(context.Context, int) error {
	return func(ctx context.Context, iteration int) error {
		for _, h := range hooks {
			if h.OnIteration != nil {
				if err := h.OnIteration(ctx, iteration); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeOnHandoff(hooks []Hooks) func(context.Context, string, string) error {
	return func(ctx context.Context, from, to string) error {
		for _, h := range hooks {
			if h.OnHandoff != nil {
				if err := h.OnHandoff(ctx, from, to); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeBeforeGenerate(hooks []Hooks) func(context.Context) error {
	return func(ctx context.Context) error {
		for _, h := range hooks {
			if h.BeforeGenerate != nil {
				if err := h.BeforeGenerate(ctx); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func composeAfterGenerate(hooks []Hooks) func(context.Context) error {
	return func(ctx context.Context) error {
		for _, h := range hooks {
			if h.AfterGenerate != nil {
				if err := h.AfterGenerate(ctx); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are called in the order the hooks were provided.
// For error-returning callbacks, the first error short-circuits.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		OnStart:        composeOnStart(h),
		OnEnd:          composeOnEnd(h),
		OnError:        composeOnError(h),
		BeforePlan:     composeBeforePlan(h),
		AfterPlan:      composeAfterPlan(h),
		BeforeAct:      composeBeforeAct(h),
		AfterAct:       composeAfterAct(h),
		OnToolCall:     composeOnToolCall(h),
		OnToolResult:   composeOnToolResult(h),
		OnIteration:    composeOnIteration(h),
		OnHandoff:      composeOnHandoff(h),
		BeforeGenerate: composeBeforeGenerate(h),
		AfterGenerate:  composeAfterGenerate(h),
	}
}
