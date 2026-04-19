package debate

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/core"
)

// Compile-time check.
var _ core.Runnable = (*GeneratorEvaluator)(nil)

// ApprovalStrategy determines how evaluator approvals are aggregated.
type ApprovalStrategy string

const (
	// ApprovalAll requires all evaluators to approve.
	ApprovalAll ApprovalStrategy = "all"
	// ApprovalMajority requires a majority (>50%) of evaluators to approve.
	ApprovalMajority ApprovalStrategy = "majority"
)

// GeneratorEvaluatorEventType identifies events during generation-evaluation.
type GeneratorEvaluatorEventType string

const (
	// GEEventGenerate indicates the generator produced a response.
	GEEventGenerate GeneratorEvaluatorEventType = "generate"
	// GEEventEvaluate indicates an evaluator returned a critique.
	GEEventEvaluate GeneratorEvaluatorEventType = "evaluate"
	// GEEventApproved indicates the response was approved.
	GEEventApproved GeneratorEvaluatorEventType = "approved"
	// GEEventRejected indicates the response was rejected and will be refined.
	GEEventRejected GeneratorEvaluatorEventType = "rejected"
	// GEEventComplete indicates the loop has finished.
	GEEventComplete GeneratorEvaluatorEventType = "complete"
)

// GeneratorEvaluatorEvent is an event emitted during generation-evaluation.
type GeneratorEvaluatorEvent struct {
	// Type identifies the kind of event.
	Type GeneratorEvaluatorEventType
	// Iteration is the one-based iteration number.
	Iteration int
	// Content holds the generated text or final response.
	Content string
	// Critiques holds evaluator feedback for the current iteration.
	Critiques []Critique
	// Approved indicates whether the current response was approved.
	Approved bool
}

// GeneratorEvaluatorResult is the final output of the generator-evaluator loop.
type GeneratorEvaluatorResult struct {
	// FinalResponse is the approved or best response.
	FinalResponse string
	// Iterations is the number of generate-evaluate iterations completed.
	Iterations int
	// Approved indicates whether the final response was approved by evaluators.
	Approved bool
	// AllCritiques contains critiques from every iteration.
	AllCritiques [][]Critique
	// Duration is the wall-clock time of the loop.
	Duration time.Duration
}

// geOptions holds configuration for GeneratorEvaluator.
type geOptions struct {
	maxIterations    int
	approvalStrategy ApprovalStrategy
	hooks            Hooks
}

func defaultGEOptions() geOptions {
	return geOptions{
		maxIterations:    3,
		approvalStrategy: ApprovalAll,
	}
}

// GeneratorEvaluator implements the generate-evaluate-refine loop.
// A generator agent produces responses that are scored by evaluator functions.
// The loop continues until all evaluators approve or max iterations is reached.
type GeneratorEvaluator struct {
	generator  agent.Agent
	evaluators []EvaluatorFunc
	opts       geOptions
}

// NewGeneratorEvaluator creates a new GeneratorEvaluator.
func NewGeneratorEvaluator(generator agent.Agent, evaluators []EvaluatorFunc, opts ...GEOption) *GeneratorEvaluator {
	o := defaultGEOptions()
	for _, opt := range opts {
		opt(&o)
	}

	return &GeneratorEvaluator{
		generator:  generator,
		evaluators: evaluators,
		opts:       o,
	}
}

// GEOption configures a GeneratorEvaluator.
type GEOption func(*geOptions)

// WithGEMaxIterations sets the maximum number of generate-evaluate iterations.
func WithGEMaxIterations(n int) GEOption {
	return func(o *geOptions) {
		if n > 0 {
			o.maxIterations = n
		}
	}
}

// WithGEApprovalStrategy sets the approval aggregation strategy.
func WithGEApprovalStrategy(s ApprovalStrategy) GEOption {
	return func(o *geOptions) {
		o.approvalStrategy = s
	}
}

// WithGEHooks sets the lifecycle hooks for the generator-evaluator.
func WithGEHooks(h Hooks) GEOption {
	return func(o *geOptions) {
		o.hooks = h
	}
}

// Invoke runs the generate-evaluate-refine loop and returns the
// GeneratorEvaluatorResult. The input must be a string prompt.
func (ge *GeneratorEvaluator) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	prompt, err := ge.extractPrompt(input)
	if err != nil {
		return nil, err
	}

	if ge.generator == nil {
		return nil, core.NewError("generator_evaluator.invoke", core.ErrInvalidInput, "generator agent is required", nil)
	}

	start := time.Now()
	var allCritiques [][]Critique
	currentPrompt := prompt
	lastResponse := ""

	for iteration := 0; iteration < ge.opts.maxIterations; iteration++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Generate.
		response, err := ge.generator.Invoke(ctx, currentPrompt)
		if err != nil {
			return nil, ge.handleError(ctx, core.Errorf(core.ErrProviderDown, "generator_evaluator: generation iteration %d: %w", iteration+1, err))
		}
		lastResponse = response

		// Evaluate.
		critiques, err := ge.evaluate(ctx, prompt, response)
		if err != nil {
			return nil, ge.handleError(ctx, core.Errorf(core.ErrProviderDown, "generator_evaluator: evaluation iteration %d: %w", iteration+1, err))
		}
		allCritiques = append(allCritiques, critiques)

		// Check approval.
		if ge.isApproved(critiques) {
			return &GeneratorEvaluatorResult{
				FinalResponse: response,
				Iterations:    iteration + 1,
				Approved:      true,
				AllCritiques:  allCritiques,
				Duration:      time.Since(start),
			}, nil
		}

		// Build refinement prompt.
		currentPrompt = ge.buildRefinementPrompt(prompt, response, critiques)
	}

	// Max iterations exhausted.
	return &GeneratorEvaluatorResult{
		FinalResponse: lastResponse,
		Iterations:    ge.opts.maxIterations,
		Approved:      false,
		AllCritiques:  allCritiques,
		Duration:      time.Since(start),
	}, nil
}

// Stream runs the generate-evaluate-refine loop, yielding events for each step.
func (ge *GeneratorEvaluator) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		prompt, err := ge.extractPrompt(input)
		if err != nil {
			yield(nil, err)
			return
		}

		if ge.generator == nil {
			yield(nil, core.NewError("generator_evaluator.stream", core.ErrInvalidInput, "generator agent is required", nil))
			return
		}

		start := time.Now()
		var allCritiques [][]Critique
		currentPrompt := prompt
		lastResponse := ""

		for iteration := 0; iteration < ge.opts.maxIterations; iteration++ {
			select {
			case <-ctx.Done():
				yield(nil, ctx.Err())
				return
			default:
			}

			// Generate.
			response, err := ge.generator.Invoke(ctx, currentPrompt)
			if err != nil {
				yield(nil, ge.handleError(ctx, core.Errorf(core.ErrProviderDown, "generator_evaluator: generation iteration %d: %w", iteration+1, err)))
				return
			}
			lastResponse = response

			if !yield(GeneratorEvaluatorEvent{
				Type:      GEEventGenerate,
				Iteration: iteration + 1,
				Content:   response,
			}, nil) {
				return
			}

			// Evaluate.
			critiques, err := ge.evaluate(ctx, prompt, response)
			if err != nil {
				yield(nil, ge.handleError(ctx, core.Errorf(core.ErrProviderDown, "generator_evaluator: evaluation iteration %d: %w", iteration+1, err)))
				return
			}
			allCritiques = append(allCritiques, critiques)

			if !yield(GeneratorEvaluatorEvent{
				Type:      GEEventEvaluate,
				Iteration: iteration + 1,
				Critiques: critiques,
			}, nil) {
				return
			}

			approved := ge.isApproved(critiques)
			if approved {
				if !yield(GeneratorEvaluatorEvent{
					Type:      GEEventApproved,
					Iteration: iteration + 1,
					Content:   response,
					Approved:  true,
				}, nil) {
					return
				}

				result := &GeneratorEvaluatorResult{
					FinalResponse: response,
					Iterations:    iteration + 1,
					Approved:      true,
					AllCritiques:  allCritiques,
					Duration:      time.Since(start),
				}
				yield(GeneratorEvaluatorEvent{
					Type:    GEEventComplete,
					Content: response,
				}, nil)
				_ = result // Event carries sufficient info.
				return
			}

			if !yield(GeneratorEvaluatorEvent{
				Type:      GEEventRejected,
				Iteration: iteration + 1,
				Critiques: critiques,
			}, nil) {
				return
			}

			currentPrompt = ge.buildRefinementPrompt(prompt, response, critiques)
		}

		// Max iterations exhausted.
		yield(GeneratorEvaluatorEvent{
			Type:    GEEventComplete,
			Content: lastResponse,
		}, nil)
	}
}

// evaluate runs all evaluators against the response.
func (ge *GeneratorEvaluator) evaluate(ctx context.Context, input, response string) ([]Critique, error) {
	critiques := make([]Critique, 0, len(ge.evaluators))
	for i, eval := range ge.evaluators {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		c, err := eval(ctx, input, response)
		if err != nil {
			return nil, core.Errorf(core.ErrProviderDown, "evaluator %d: %w", i, err)
		}
		critiques = append(critiques, c)
	}
	return critiques, nil
}

// isApproved checks whether the critiques meet the approval strategy.
// An empty critique list means no evaluators ran and cannot count as
// approval — returning false forces the loop to exit on max iterations
// rather than silently rubber-stamping the first generation.
func (ge *GeneratorEvaluator) isApproved(critiques []Critique) bool {
	if len(critiques) == 0 {
		return false
	}

	approvedCount := 0
	for _, c := range critiques {
		if c.Approved {
			approvedCount++
		}
	}

	switch ge.opts.approvalStrategy {
	case ApprovalMajority:
		return approvedCount > len(critiques)/2
	default: // ApprovalAll
		return approvedCount == len(critiques)
	}
}

// buildRefinementPrompt creates a prompt that includes feedback for refinement.
func (ge *GeneratorEvaluator) buildRefinementPrompt(originalPrompt, response string, critiques []Critique) string {
	var sb strings.Builder
	sb.WriteString("Original request: ")
	sb.WriteString(originalPrompt)
	sb.WriteString("\n\nYour previous response:\n")
	sb.WriteString(response)
	sb.WriteString("\n\nFeedback from evaluators:\n")
	for i, c := range critiques {
		sb.WriteString(fmt.Sprintf("- Evaluator %d (score: %.2f, approved: %v): %s\n", i+1, c.Score, c.Approved, c.Feedback))
	}
	sb.WriteString("\nPlease revise your response addressing the feedback above.")
	return sb.String()
}

// extractPrompt validates and extracts the prompt string from input.
func (ge *GeneratorEvaluator) extractPrompt(input any) (string, error) {
	prompt, ok := input.(string)
	if !ok {
		return "", core.NewError("generator_evaluator", core.ErrInvalidInput, "input must be a string prompt", nil)
	}
	if strings.TrimSpace(prompt) == "" {
		return "", core.NewError("generator_evaluator", core.ErrInvalidInput, "prompt must not be empty", nil)
	}
	return prompt, nil
}

// handleError invokes the OnError hook if configured.
func (ge *GeneratorEvaluator) handleError(ctx context.Context, err error) error {
	if ge.opts.hooks.OnError != nil {
		return ge.opts.hooks.OnError(ctx, err)
	}
	return err
}
