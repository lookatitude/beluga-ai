package workflow

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/hitl"
)

// LLMActivity creates an ActivityFunc that invokes an LLM. The invoker function
// is called with the input (expected to be a string prompt).
func LLMActivity(invoker func(ctx context.Context, prompt string) (string, error)) ActivityFunc {
	return func(ctx context.Context, input any) (any, error) {
		prompt, ok := input.(string)
		if !ok {
			return nil, fmt.Errorf("workflow/llm_activity: expected string input, got %T", input)
		}
		return invoker(ctx, prompt)
	}
}

// ToolActivity creates an ActivityFunc that executes a tool. The executor function
// is called with the tool name and input arguments.
func ToolActivity(executor func(ctx context.Context, name string, args map[string]any) (any, error)) ActivityFunc {
	return func(ctx context.Context, input any) (any, error) {
		params, ok := input.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("workflow/tool_activity: expected map[string]any input, got %T", input)
		}
		name, _ := params["name"].(string)
		args, _ := params["args"].(map[string]any)
		if name == "" {
			return nil, fmt.Errorf("workflow/tool_activity: missing 'name' in input")
		}
		return executor(ctx, name, args)
	}
}

// HumanActivity creates an ActivityFunc that requests human interaction via
// a hitl.Manager. It creates an interaction request and blocks until a human
// responds or the context expires.
func HumanActivity(mgr hitl.Manager) ActivityFunc {
	return func(ctx context.Context, input any) (any, error) {
		req, ok := input.(hitl.InteractionRequest)
		if !ok {
			return nil, fmt.Errorf("workflow/human_activity: expected hitl.InteractionRequest, got %T", input)
		}
		resp, err := mgr.RequestInteraction(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("workflow/human_activity: %w", err)
		}
		return resp, nil
	}
}
