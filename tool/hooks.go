package tool

import "context"

// Hooks provides lifecycle callbacks for tool execution. All fields are
// optional — nil hooks are skipped. Hooks can be composed using ComposeHooks.
type Hooks struct {
	// BeforeExecute is called before a tool executes. It receives the tool
	// name and the input map. Returning an error aborts execution.
	BeforeExecute func(ctx context.Context, toolName string, input map[string]any) error

	// AfterExecute is called after a tool executes successfully or with an
	// error. It receives the tool name, the result (which may be nil), and
	// any error from execution.
	AfterExecute func(ctx context.Context, toolName string, result *Result, err error)

	// OnError is called when tool execution fails. It receives the tool name
	// and the error. Returning a non-nil error propagates it; returning nil
	// suppresses the original error.
	OnError func(ctx context.Context, toolName string, err error) error
}

// ComposeHooks merges multiple Hooks into a single Hooks struct.
// BeforeExecute hooks run in order; if any returns an error, subsequent hooks
// are skipped. AfterExecute hooks run in order unconditionally. OnError hooks
// run in order; the first non-nil return wins.
func ComposeHooks(hooks ...Hooks) Hooks {
	return Hooks{
		BeforeExecute: func(ctx context.Context, toolName string, input map[string]any) error {
			for _, h := range hooks {
				if h.BeforeExecute != nil {
					if err := h.BeforeExecute(ctx, toolName, input); err != nil {
						return err
					}
				}
			}
			return nil
		},
		AfterExecute: func(ctx context.Context, toolName string, result *Result, err error) {
			for _, h := range hooks {
				if h.AfterExecute != nil {
					h.AfterExecute(ctx, toolName, result, err)
				}
			}
		},
		OnError: func(ctx context.Context, toolName string, err error) error {
			for _, h := range hooks {
				if h.OnError != nil {
					if newErr := h.OnError(ctx, toolName, err); newErr != nil {
						return newErr
					}
				}
			}
			return err
		},
	}
}

// WithHooks wraps a tool so that the provided hooks are invoked around each
// Execute call.
func WithHooks(t Tool, h Hooks) Tool {
	return &hookedTool{tool: t, hooks: h}
}

type hookedTool struct {
	tool  Tool
	hooks Hooks
}

func (h *hookedTool) Name() string              { return h.tool.Name() }
func (h *hookedTool) Description() string        { return h.tool.Description() }
func (h *hookedTool) InputSchema() map[string]any { return h.tool.InputSchema() }

func (h *hookedTool) Execute(ctx context.Context, input map[string]any) (*Result, error) {
	name := h.tool.Name()

	// BeforeExecute
	if h.hooks.BeforeExecute != nil {
		if err := h.hooks.BeforeExecute(ctx, name, input); err != nil {
			return nil, err
		}
	}

	// Execute
	result, err := h.tool.Execute(ctx, input)

	// OnError
	if err != nil && h.hooks.OnError != nil {
		err = h.hooks.OnError(ctx, name, err)
	}

	// AfterExecute
	if h.hooks.AfterExecute != nil {
		h.hooks.AfterExecute(ctx, name, result, err)
	}

	return result, err
}
