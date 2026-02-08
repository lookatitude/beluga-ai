package orchestration

import (
	"context"
	"fmt"
	"iter"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/core"
)

// AggregatorFunc combines results from multiple workers into a single result.
type AggregatorFunc func(results []any) (any, error)

// ScatterGather fans out input to multiple workers concurrently and aggregates
// their results. An optional timeout limits the total execution time.
type ScatterGather struct {
	workers    []core.Runnable
	aggregator AggregatorFunc
	timeout    time.Duration
}

// NewScatterGather creates a ScatterGather with the given aggregator and workers.
func NewScatterGather(aggregator AggregatorFunc, workers ...core.Runnable) *ScatterGather {
	return &ScatterGather{
		workers:    workers,
		aggregator: aggregator,
	}
}

// WithTimeout sets a timeout for the scatter-gather operation.
// A zero or negative duration means no timeout.
func (sg *ScatterGather) WithTimeout(d time.Duration) *ScatterGather {
	sg.timeout = d
	return sg
}

// Invoke fans out input to all workers concurrently, collects results, and
// calls the aggregator.
func (sg *ScatterGather) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	if len(sg.workers) == 0 {
		return sg.aggregator(nil)
	}

	if sg.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, sg.timeout)
		defer cancel()
	}

	results := make([]any, len(sg.workers))
	errs := make([]error, len(sg.workers))
	var wg sync.WaitGroup
	wg.Add(len(sg.workers))

	for i, w := range sg.workers {
		go func(i int, w core.Runnable) {
			defer wg.Done()
			results[i], errs[i] = w.Invoke(ctx, input, opts...)
		}(i, w)
	}
	wg.Wait()

	// Check for errors.
	for i, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("orchestration/scatter: worker %d: %w", i, err)
		}
	}

	aggregated, err := sg.aggregator(results)
	if err != nil {
		return nil, fmt.Errorf("orchestration/scatter: aggregator: %w", err)
	}
	return aggregated, nil
}

// Stream fans out input to all workers, aggregates results, and yields the
// aggregated result as a single value.
func (sg *ScatterGather) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		result, err := sg.Invoke(ctx, input, opts...)
		yield(result, err)
	}
}
