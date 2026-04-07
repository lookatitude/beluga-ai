package orchestration

import (
	"context"
	"fmt"
	"iter"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/core"
)

const pipelineStageErrFmt = "orchestration/pipeline: stage %d (%s): %w"

// Pipeline executes a sequence of agents where the text output of agent N
// becomes the string input of agent N+1. Events from every stage are yielded
// in order. Pipeline satisfies [OrchestrationPattern].
//
// For Invoke, each agent is called synchronously in order. The final result is
// the text output of the last agent.
//
// For Stream, each agent is invoked synchronously except for the last, which
// is streamed. Events from all stages are yielded to the caller.
type Pipeline struct {
	stages []agent.Agent
}

// compile-time check.
var _ OrchestrationPattern = (*Pipeline)(nil)

// NewPipeline creates a Pipeline from the given agents in order.
func NewPipeline(stages ...agent.Agent) *Pipeline {
	return &Pipeline{stages: stages}
}

// Name returns the pattern identifier.
func (p *Pipeline) Name() string { return "pipeline" }

// Invoke executes each agent in sequence, passing the text output of agent N
// as the input to agent N+1. Returns the final agent's text output.
func (p *Pipeline) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	if len(p.stages) == 0 {
		return input, nil
	}

	current := fmt.Sprintf("%v", input)

	for i, stage := range p.stages {
		result, err := stage.Invoke(ctx, current)
		if err != nil {
			return nil, fmt.Errorf(pipelineStageErrFmt, i, stage.ID(), err)
		}
		current = result
	}

	return current, nil
}

// Stream executes the leading stages synchronously and streams the final
// stage. Events from the final stage are yielded to the caller.
//
// If there is only one stage, it is streamed directly. If the pipeline is
// empty, the input is yielded unchanged.
func (p *Pipeline) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		if len(p.stages) == 0 {
			yield(input, nil)
			return
		}

		// Run leading stages synchronously.
		current := fmt.Sprintf("%v", input)
		for i, stage := range p.stages[:len(p.stages)-1] {
			select {
			case <-ctx.Done():
				yield(nil, ctx.Err())
				return
			default:
			}

			result, err := stage.Invoke(ctx, current)
			if err != nil {
				yield(nil, fmt.Errorf(pipelineStageErrFmt, i, stage.ID(), err))
				return
			}
			current = result
		}

		// Stream the final stage.
		last := p.stages[len(p.stages)-1]
		lastIdx := len(p.stages) - 1

		for event, err := range last.Stream(ctx, current) {
			if err != nil {
				yield(nil, fmt.Errorf(pipelineStageErrFmt, lastIdx, last.ID(), err))
				return
			}
			if !yield(event, nil) {
				return
			}
		}
	}
}
