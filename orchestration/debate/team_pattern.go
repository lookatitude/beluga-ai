package debate

import (
	"context"
	"iter"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/orchestration"
)

// DebatePattern creates an OrchestrationPattern from a DebateOrchestrator.
func DebatePattern(agents []agent.Agent, opts ...Option) orchestration.OrchestrationPattern {
	d := NewDebateOrchestrator(agents, opts...)
	return &debatePatternAdapter{
		orchestrator: d,
	}
}

// GeneratorEvaluatorPattern creates an OrchestrationPattern from a
// GeneratorEvaluator.
func GeneratorEvaluatorPattern(generator agent.Agent, evaluators []EvaluatorFunc, opts ...GEOption) orchestration.OrchestrationPattern {
	ge := NewGeneratorEvaluator(generator, evaluators, opts...)
	return &gePatternAdapter{
		ge: ge,
	}
}

// debatePatternAdapter wraps DebateOrchestrator as OrchestrationPattern.
type debatePatternAdapter struct {
	orchestrator *DebateOrchestrator
}

// Compile-time check.
var _ orchestration.OrchestrationPattern = (*debatePatternAdapter)(nil)

// Name returns the pattern name.
func (a *debatePatternAdapter) Name() string { return "debate" }

// Invoke delegates to the debate orchestrator.
func (a *debatePatternAdapter) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	return a.orchestrator.Invoke(ctx, input, opts...)
}

// Stream delegates to the debate orchestrator.
func (a *debatePatternAdapter) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return a.orchestrator.Stream(ctx, input, opts...)
}

// gePatternAdapter wraps GeneratorEvaluator as OrchestrationPattern.
type gePatternAdapter struct {
	ge *GeneratorEvaluator
}

// Compile-time check.
var _ orchestration.OrchestrationPattern = (*gePatternAdapter)(nil)

// Name returns the pattern name.
func (a *gePatternAdapter) Name() string { return "generator_evaluator" }

// Invoke delegates to the generator-evaluator.
func (a *gePatternAdapter) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	return a.ge.Invoke(ctx, input, opts...)
}

// Stream delegates to the generator-evaluator.
func (a *gePatternAdapter) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return a.ge.Stream(ctx, input, opts...)
}
