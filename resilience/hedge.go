package resilience

import (
	"context"
	"sync"
	"time"
)

// Hedge executes primary immediately. If primary does not return within delay,
// secondary is started concurrently. The result from whichever function
// completes successfully first is returned. Both goroutines are cancelled once
// a result is available. If both fail, the primary error is returned.
func Hedge[T any](
	ctx context.Context,
	primary func(ctx context.Context) (T, error),
	secondary func(ctx context.Context) (T, error),
	delay time.Duration,
) (T, error) {
	type result struct {
		val T
		err error
	}

	// Create a cancellable context so we can stop the loser.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := make(chan result, 2)

	var wg sync.WaitGroup

	// Always start primary immediately.
	wg.Add(1)
	go func() {
		defer wg.Done()
		v, err := primary(ctx)
		ch <- result{val: v, err: err}
	}()

	// Schedule secondary after the delay, unless primary finishes first.
	secondaryStarted := false
	timer := time.NewTimer(delay)
	defer timer.Stop()

	// Wait for primary result OR timer to fire.
	select {
	case r := <-ch:
		// Primary returned before the delay elapsed.
		if r.err == nil {
			cancel()
			return r.val, nil
		}
		// Primary failed before delay — start secondary now.
		wg.Add(1)
		secondaryStarted = true
		go func() {
			defer wg.Done()
			v, err := secondary(ctx)
			ch <- result{val: v, err: err}
		}()

		// Wait for secondary.
		r2 := <-ch
		if r2.err == nil {
			return r2.val, nil
		}
		// Both failed; return primary error.
		return r.val, r.err

	case <-timer.C:
		// Delay elapsed — start secondary.
		wg.Add(1)
		secondaryStarted = true
		go func() {
			defer wg.Done()
			v, err := secondary(ctx)
			ch <- result{val: v, err: err}
		}()

	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	}

	// Both primary and secondary are running. Wait for the first success.
	var primaryErr error
	for i := 0; i < 2; i++ {
		r := <-ch
		if r.err == nil {
			cancel()
			return r.val, nil
		}
		if primaryErr == nil {
			primaryErr = r.err
		}
	}

	_ = secondaryStarted // used above

	var zero T
	return zero, primaryErr
}
