package core

import (
	"context"
	"sync"
	"time"
)

// BatchOptions controls the concurrency and timeout behaviour of BatchInvoke.
type BatchOptions struct {
	// MaxConcurrency limits the number of concurrent executions. Zero or
	// negative means no limit.
	MaxConcurrency int

	// BatchSize is the maximum number of items per batch API call. It is
	// informational for callers; BatchInvoke processes one item per fn call.
	BatchSize int

	// Timeout is a per-item timeout. Zero means no per-item timeout (the
	// parent context deadline still applies).
	Timeout time.Duration

	// RetryPolicy optionally configures retry behaviour for failed items.
	RetryPolicy *RetryPolicy
}

// RetryPolicy specifies how failed operations should be retried.
type RetryPolicy struct {
	// MaxAttempts is the total number of attempts (including the first).
	MaxAttempts int

	// InitialBackoff is the delay before the first retry.
	InitialBackoff time.Duration

	// MaxBackoff caps the backoff duration.
	MaxBackoff time.Duration

	// BackoffFactor multiplies the backoff after each retry.
	BackoffFactor float64

	// Jitter adds randomness to the backoff to avoid thundering herds.
	Jitter bool
}

// BatchResult holds the output and error for a single item in a batch.
type BatchResult[O any] struct {
	// Value is the result of the invocation.
	Value O

	// Err is the error from the invocation, if any.
	Err error
}

// BatchInvoke executes fn for each input concurrently, respecting the
// concurrency limit in opts. It returns a result for every input at the
// corresponding index.
func BatchInvoke[I, O any](
	ctx context.Context,
	fn func(context.Context, I) (O, error),
	inputs []I,
	opts BatchOptions,
) []BatchResult[O] {
	results := make([]BatchResult[O], len(inputs))

	var sem chan struct{}
	if opts.MaxConcurrency > 0 {
		sem = make(chan struct{}, opts.MaxConcurrency)
	}

	var wg sync.WaitGroup
	wg.Add(len(inputs))

	for i, input := range inputs {
		if sem != nil {
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				// Fill remaining results with context error.
				for j := i; j < len(inputs); j++ {
					results[j].Err = ctx.Err()
					wg.Done()
				}
				wg.Wait()
				return results
			}
		}

		go func(i int, input I) {
			defer wg.Done()
			if sem != nil {
				defer func() { <-sem }()
			}

			itemCtx := ctx
			if opts.Timeout > 0 {
				var cancel context.CancelFunc
				itemCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
				defer cancel()
			}

			results[i].Value, results[i].Err = fn(itemCtx, input)
		}(i, input)
	}

	wg.Wait()
	return results
}
