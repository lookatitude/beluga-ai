package core

import (
	"context"
	"math/rand/v2"
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
	// Only errors for which IsRetryable reports true are retried, and retries
	// stop as soon as the item's context is done. Nil means no retries.
	RetryPolicy *RetryPolicy
}

// RetryPolicy specifies how failed operations should be retried. It is used
// by BatchOptions; resilience.RetryPolicy is the general-purpose equivalent
// for other call sites (core cannot import resilience).
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
		if !acquireSemaphore(ctx, sem) {
			fillRemainingErrors(results, i, len(inputs), ctx.Err(), &wg)
			wg.Wait()
			return results
		}

		go batchInvokeItem(ctx, batchItemParams[I, O]{
			fn:      fn,
			input:   input,
			idx:     i,
			results: results,
			sem:     sem,
			timeout: opts.Timeout,
			retry:   opts.RetryPolicy,
			wg:      &wg,
		})
	}

	wg.Wait()
	return results
}

// acquireSemaphore blocks until a slot is available on sem, or ctx is cancelled.
// Returns true if acquired, false if ctx was cancelled. If sem is nil, always returns true.
func acquireSemaphore(ctx context.Context, sem chan struct{}) bool {
	if sem == nil {
		return true
	}
	select {
	case sem <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}

// fillRemainingErrors fills results[from:to] with the given error and calls wg.Done for each.
func fillRemainingErrors[O any](results []BatchResult[O], from, to int, err error, wg *sync.WaitGroup) {
	for j := from; j < to; j++ {
		results[j].Err = err
		wg.Done()
	}
}

// batchItemParams groups the parameters for batchInvokeItem.
type batchItemParams[I, O any] struct {
	fn      func(context.Context, I) (O, error)
	input   I
	idx     int
	results []BatchResult[O]
	sem     chan struct{}
	timeout time.Duration
	retry   *RetryPolicy
	wg      *sync.WaitGroup
}

// batchInvokeItem executes fn for a single input and stores the result.
func batchInvokeItem[I, O any](ctx context.Context, p batchItemParams[I, O]) {
	defer p.wg.Done()
	if p.sem != nil {
		defer func() { <-p.sem }()
	}

	itemCtx := ctx
	if p.timeout > 0 {
		var cancel context.CancelFunc
		itemCtx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}

	p.results[p.idx].Value, p.results[p.idx].Err = invokeWithRetry(itemCtx, p.fn, p.input, p.retry)
}

// invokeWithRetry calls fn, retrying retryable errors according to policy.
func invokeWithRetry[I, O any](ctx context.Context, fn func(context.Context, I) (O, error), input I, policy *RetryPolicy) (O, error) {
	out, err := fn(ctx, input)
	if policy == nil || policy.MaxAttempts <= 1 {
		return out, err
	}
	backoff := policy.InitialBackoff
	for attempt := 1; attempt < policy.MaxAttempts && err != nil && IsRetryable(err); attempt++ {
		if !sleepCtx(ctx, jittered(backoff, policy.Jitter)) {
			return out, err
		}
		out, err = fn(ctx, input)
		backoff = nextBackoff(backoff, policy)
	}
	return out, err
}

// nextBackoff grows d by the policy factor (default 2), capped at MaxBackoff.
func nextBackoff(d time.Duration, policy *RetryPolicy) time.Duration {
	factor := policy.BackoffFactor
	if factor <= 0 {
		factor = 2
	}
	next := time.Duration(float64(d) * factor)
	if policy.MaxBackoff > 0 && next > policy.MaxBackoff {
		next = policy.MaxBackoff
	}
	return next
}

// jittered returns a duration in [d/2, d] when jitter is enabled.
func jittered(d time.Duration, jitter bool) time.Duration {
	if !jitter || d <= 1 {
		return d
	}
	half := d / 2
	return half + time.Duration(rand.Int64N(int64(half)+1)) // #nosec G404 -- jitter, not security sensitive
}

// sleepCtx waits for d or until ctx is done. It reports whether the full
// duration elapsed.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
