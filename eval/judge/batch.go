package judge

import (
	"context"
	"sync"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/eval"
)

// batchOptions holds configuration for BatchJudge.
type batchOptions struct {
	metric   *JudgeMetric
	parallel int
	onResult func(index int, score float64, err error)
}

// BatchOption configures a BatchJudge.
type BatchOption func(*batchOptions)

// WithJudgeMetric sets the JudgeMetric used for batch evaluation.
func WithJudgeMetric(m *JudgeMetric) BatchOption {
	return func(o *batchOptions) {
		o.metric = m
	}
}

// WithParallel sets the maximum number of concurrent evaluations.
// Defaults to 4.
func WithParallel(n int) BatchOption {
	return func(o *batchOptions) {
		if n > 0 {
			o.parallel = n
		}
	}
}

// WithOnResult sets a callback invoked after each sample is scored.
func WithOnResult(fn func(index int, score float64, err error)) BatchOption {
	return func(o *batchOptions) {
		o.onResult = fn
	}
}

// BatchResult contains scores for a batch evaluation run.
type BatchResult struct {
	// Scores maps sample index to score. Missing entries indicate errors.
	Scores map[int]float64

	// Errors maps sample index to error for failed evaluations.
	Errors map[int]error
}

// BatchJudge evaluates multiple samples concurrently with bounded parallelism.
type BatchJudge struct {
	opts batchOptions
}

// NewBatchJudge creates a new BatchJudge with the given options.
func NewBatchJudge(opts ...BatchOption) (*BatchJudge, error) {
	o := batchOptions{parallel: 4}
	for _, opt := range opts {
		opt(&o)
	}
	if o.metric == nil {
		return nil, core.NewError("judge.batch.new", core.ErrInvalidInput, "judge metric is required", nil)
	}
	return &BatchJudge{opts: o}, nil
}

// Evaluate scores all samples concurrently using the configured JudgeMetric
// and returns the aggregated results.
func (b *BatchJudge) Evaluate(ctx context.Context, samples []eval.EvalSample) (*BatchResult, error) {
	result := &BatchResult{
		Scores: make(map[int]float64, len(samples)),
		Errors: make(map[int]error),
	}

	sem := make(chan struct{}, b.opts.parallel)
	var mu sync.Mutex
	var wg sync.WaitGroup

dispatch:
	for i, sample := range samples {
		if ctx.Err() != nil {
			break
		}

		// Acquire a slot, respecting context cancellation so a cancelled
		// context is noticed even when all parallel slots are in use.
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break dispatch
		}

		wg.Add(1)
		go func(idx int, s eval.EvalSample) {
			defer wg.Done()
			defer func() { <-sem }()

			score, err := b.opts.metric.Score(ctx, s)

			mu.Lock()
			if err != nil {
				result.Errors[idx] = err
			} else {
				result.Scores[idx] = score
			}
			mu.Unlock()

			if b.opts.onResult != nil {
				b.opts.onResult(idx, score, err)
			}
		}(i, sample)
	}

	wg.Wait()
	return result, nil
}
