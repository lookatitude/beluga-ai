package core

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestBatchInvoke_AllSucceed(t *testing.T) {
	fn := func(_ context.Context, n int) (string, error) {
		return fmt.Sprintf("result-%d", n), nil
	}

	results := BatchInvoke(context.Background(), fn, []int{1, 2, 3}, BatchOptions{})

	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
	for i, r := range results {
		if r.Err != nil {
			t.Errorf("results[%d].Err = %v, want nil", i, r.Err)
		}
		want := fmt.Sprintf("result-%d", i+1)
		if r.Value != want {
			t.Errorf("results[%d].Value = %q, want %q", i, r.Value, want)
		}
	}
}

func TestBatchInvoke_SomeFail(t *testing.T) {
	fn := func(_ context.Context, n int) (int, error) {
		if n%2 == 0 {
			return 0, fmt.Errorf("even number: %d", n)
		}
		return n * 10, nil
	}

	results := BatchInvoke(context.Background(), fn, []int{1, 2, 3, 4}, BatchOptions{})

	if len(results) != 4 {
		t.Fatalf("len(results) = %d, want 4", len(results))
	}
	if results[0].Err != nil || results[0].Value != 10 {
		t.Errorf("results[0] = {%v, %v}, want {10, nil}", results[0].Value, results[0].Err)
	}
	if results[1].Err == nil {
		t.Error("results[1].Err should not be nil for even input")
	}
	if results[2].Err != nil || results[2].Value != 30 {
		t.Errorf("results[2] = {%v, %v}, want {30, nil}", results[2].Value, results[2].Err)
	}
	if results[3].Err == nil {
		t.Error("results[3].Err should not be nil for even input")
	}
}

func TestBatchInvoke_EmptyInputs(t *testing.T) {
	fn := func(_ context.Context, n int) (int, error) {
		return n, nil
	}

	results := BatchInvoke(context.Background(), fn, []int{}, BatchOptions{})
	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
}

func TestBatchInvoke_MaxConcurrency(t *testing.T) {
	var concurrent atomic.Int32
	var maxSeen atomic.Int32

	fn := func(_ context.Context, _ int) (int, error) {
		cur := concurrent.Add(1)
		// Track max concurrent.
		for {
			old := maxSeen.Load()
			if cur <= old || maxSeen.CompareAndSwap(old, cur) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		concurrent.Add(-1)
		return 0, nil
	}

	results := BatchInvoke(context.Background(), fn, []int{1, 2, 3, 4, 5, 6, 7, 8}, BatchOptions{
		MaxConcurrency: 2,
	})

	if len(results) != 8 {
		t.Fatalf("len(results) = %d, want 8", len(results))
	}
	for i, r := range results {
		if r.Err != nil {
			t.Errorf("results[%d].Err = %v", i, r.Err)
		}
	}

	if max := maxSeen.Load(); max > 2 {
		t.Errorf("max concurrent = %d, want <= 2", max)
	}
}

func TestBatchInvoke_PerItemTimeout(t *testing.T) {
	fn := func(ctx context.Context, n int) (int, error) {
		if n == 2 {
			// Simulate a slow operation.
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(5 * time.Second):
				return n, nil
			}
		}
		return n * 10, nil
	}

	results := BatchInvoke(context.Background(), fn, []int{1, 2, 3}, BatchOptions{
		Timeout: 50 * time.Millisecond,
	})

	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
	if results[0].Err != nil || results[0].Value != 10 {
		t.Errorf("results[0] = {%v, %v}, want {10, nil}", results[0].Value, results[0].Err)
	}
	if results[1].Err == nil {
		t.Error("results[1].Err should be non-nil (timeout)")
	}
	if results[2].Err != nil || results[2].Value != 30 {
		t.Errorf("results[2] = {%v, %v}, want {30, nil}", results[2].Value, results[2].Err)
	}
}

func TestBatchInvoke_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	callCount := atomic.Int32{}
	fn := func(ctx context.Context, n int) (int, error) {
		callCount.Add(1)
		// Simulate work.
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			return n, nil
		}
	}

	// Cancel quickly so remaining items get context error.
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	results := BatchInvoke(ctx, fn, []int{1, 2, 3, 4, 5}, BatchOptions{
		MaxConcurrency: 1,
	})

	if len(results) != 5 {
		t.Fatalf("len(results) = %d, want 5", len(results))
	}

	// At least some should have context errors.
	hasCtxErr := false
	for _, r := range results {
		if r.Err == context.Canceled {
			hasCtxErr = true
			break
		}
	}
	if !hasCtxErr {
		t.Error("expected at least one result with context.Canceled error")
	}
}

func TestBatchInvoke_NoConcurrencyLimit(t *testing.T) {
	fn := func(_ context.Context, n int) (int, error) {
		return n, nil
	}

	// MaxConcurrency = 0 means unlimited.
	results := BatchInvoke(context.Background(), fn, []int{1, 2, 3}, BatchOptions{
		MaxConcurrency: 0,
	})

	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
	for i, r := range results {
		if r.Err != nil {
			t.Errorf("results[%d].Err = %v", i, r.Err)
		}
		if r.Value != i+1 {
			t.Errorf("results[%d].Value = %d, want %d", i, r.Value, i+1)
		}
	}
}

func TestBatchInvoke_PreservesOrder(t *testing.T) {
	fn := func(_ context.Context, n int) (int, error) {
		// Vary sleep to encourage out-of-order completion.
		time.Sleep(time.Duration(10-n) * time.Millisecond)
		return n * 100, nil
	}

	results := BatchInvoke(context.Background(), fn, []int{1, 2, 3, 4, 5}, BatchOptions{})

	for i, r := range results {
		want := (i + 1) * 100
		if r.Value != want {
			t.Errorf("results[%d].Value = %d, want %d", i, r.Value, want)
		}
	}
}
