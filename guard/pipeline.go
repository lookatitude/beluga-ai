package guard

import "context"

// Pipeline orchestrates the three-stage guard validation: input guards run on
// user messages, output guards run on model responses, and tool guards run on
// tool call arguments. Guards within each stage execute in order; the first
// guard that blocks stops the pipeline for that stage.
type Pipeline struct {
	inputGuards  []Guard
	outputGuards []Guard
	toolGuards   []Guard
}

// PipelineOption configures a Pipeline during construction.
type PipelineOption func(*Pipeline)

// NewPipeline creates a Pipeline configured with the given options.
func NewPipeline(opts ...PipelineOption) *Pipeline {
	p := &Pipeline{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Input returns a PipelineOption that appends guards to the input stage.
func Input(guards ...Guard) PipelineOption {
	return func(p *Pipeline) {
		p.inputGuards = append(p.inputGuards, guards...)
	}
}

// Output returns a PipelineOption that appends guards to the output stage.
func Output(guards ...Guard) PipelineOption {
	return func(p *Pipeline) {
		p.outputGuards = append(p.outputGuards, guards...)
	}
}

// Tool returns a PipelineOption that appends guards to the tool stage.
func Tool(guards ...Guard) PipelineOption {
	return func(p *Pipeline) {
		p.toolGuards = append(p.toolGuards, guards...)
	}
}

// ValidateInput runs all input guards sequentially against the given content.
// It returns the first blocking result or an aggregate allowed result. If any
// guard modifies the content, subsequent guards see the modified version.
func (p *Pipeline) ValidateInput(ctx context.Context, content string) (GuardResult, error) {
	return p.runGuards(ctx, p.inputGuards, content, "input", nil)
}

// ValidateOutput runs all output guards sequentially against the given content.
// It returns the first blocking result or an aggregate allowed result.
func (p *Pipeline) ValidateOutput(ctx context.Context, content string) (GuardResult, error) {
	return p.runGuards(ctx, p.outputGuards, content, "output", nil)
}

// ValidateTool runs all tool guards sequentially against the given tool input.
// The toolName is passed in Metadata["tool_name"] so guards can apply
// tool-specific policies.
func (p *Pipeline) ValidateTool(ctx context.Context, toolName, input string) (GuardResult, error) {
	meta := map[string]any{"tool_name": toolName}
	return p.runGuards(ctx, p.toolGuards, input, "tool", meta)
}

// runGuards executes guards in order. Each guard receives the (possibly
// modified) content from the previous guard. The first non-allowed result
// stops the chain. If all guards allow, the final (possibly modified) content
// is returned.
func (p *Pipeline) runGuards(ctx context.Context, guards []Guard, content, role string, meta map[string]any) (GuardResult, error) {
	current := content
	for _, g := range guards {
		select {
		case <-ctx.Done():
			return GuardResult{}, ctx.Err()
		default:
		}

		result, err := g.Validate(ctx, GuardInput{
			Content:  current,
			Role:     role,
			Metadata: meta,
		})
		if err != nil {
			return GuardResult{}, err
		}
		if !result.Allowed {
			result.GuardName = g.Name()
			return result, nil
		}
		// If the guard modified the content, pass the modified version
		// to subsequent guards.
		if result.Modified != "" {
			current = result.Modified
		}
	}

	// All guards passed.
	result := GuardResult{Allowed: true}
	if current != content {
		result.Modified = current
	}
	return result, nil
}
