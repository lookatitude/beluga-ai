package orchestration

import (
	"context"
	"fmt"
	"iter"

	"github.com/lookatitude/beluga-ai/core"
)

// Chain composes steps sequentially: the output of step N becomes the input
// of step N+1. The resulting Runnable supports both Invoke and Stream.
//
// For Invoke, all steps are executed in order, each receiving the output of
// the previous step.
//
// For Stream, all steps except the last are invoked synchronously, and the
// last step is streamed.
//
// An empty chain returns the input unchanged.
func Chain(steps ...core.Runnable) core.Runnable {
	return &chainRunnable{steps: steps}
}

type chainRunnable struct {
	steps []core.Runnable
}

func (c *chainRunnable) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	if len(c.steps) == 0 {
		return input, nil
	}

	current := input
	for i, step := range c.steps {
		result, err := step.Invoke(ctx, current, opts...)
		if err != nil {
			return nil, fmt.Errorf("orchestration/chain: step %d: %w", i, err)
		}
		current = result
	}
	return current, nil
}

func (c *chainRunnable) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		if len(c.steps) == 0 {
			yield(input, nil)
			return
		}

		// Invoke all steps except the last synchronously.
		current := input
		for i, step := range c.steps[:len(c.steps)-1] {
			result, err := step.Invoke(ctx, current, opts...)
			if err != nil {
				yield(nil, fmt.Errorf("orchestration/chain: step %d: %w", i, err))
				return
			}
			current = result
		}

		// Stream the last step.
		last := c.steps[len(c.steps)-1]
		for val, err := range last.Stream(ctx, current, opts...) {
			if err != nil {
				yield(nil, fmt.Errorf("orchestration/chain: step %d: %w", len(c.steps)-1, err))
				return
			}
			if !yield(val, nil) {
				return
			}
		}
	}
}
