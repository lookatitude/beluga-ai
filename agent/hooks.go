package agent

import (
	"context"

	"github.com/lookatitude/beluga-ai/internal/hookutil"
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

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are called in the order the hooks were provided.
// For error-returning callbacks, the first error short-circuits.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		OnStart: hookutil.ComposeError1(h, func(hk Hooks) func(context.Context, string) error {
			return hk.OnStart
		}),
		OnEnd: hookutil.ComposeVoid2(h, func(hk Hooks) func(context.Context, string, error) {
			return hk.OnEnd
		}),
		OnError: hookutil.ComposeErrorPassthrough(h, func(hk Hooks) func(context.Context, error) error {
			return hk.OnError
		}),
		BeforePlan: hookutil.ComposeError1(h, func(hk Hooks) func(context.Context, PlannerState) error {
			return hk.BeforePlan
		}),
		AfterPlan: hookutil.ComposeError1(h, func(hk Hooks) func(context.Context, []Action) error {
			return hk.AfterPlan
		}),
		BeforeAct: hookutil.ComposeError1(h, func(hk Hooks) func(context.Context, Action) error {
			return hk.BeforeAct
		}),
		AfterAct: hookutil.ComposeError2(h, func(hk Hooks) func(context.Context, Action, Observation) error {
			return hk.AfterAct
		}),
		OnToolCall: hookutil.ComposeError1(h, func(hk Hooks) func(context.Context, ToolCallInfo) error {
			return hk.OnToolCall
		}),
		OnToolResult: hookutil.ComposeError2(h, func(hk Hooks) func(context.Context, ToolCallInfo, *tool.Result) error {
			return hk.OnToolResult
		}),
		OnIteration: hookutil.ComposeError1(h, func(hk Hooks) func(context.Context, int) error {
			return hk.OnIteration
		}),
		OnHandoff: hookutil.ComposeError2(h, func(hk Hooks) func(context.Context, string, string) error {
			return hk.OnHandoff
		}),
		BeforeGenerate: hookutil.ComposeError0(h, func(hk Hooks) func(context.Context) error {
			return hk.BeforeGenerate
		}),
		AfterGenerate: hookutil.ComposeError0(h, func(hk Hooks) func(context.Context) error {
			return hk.AfterGenerate
		}),
	}
}
