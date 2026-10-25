package tool

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/v2/core"
)

func TestWithTimeout_Success(t *testing.T) {
	base := &mockTool{
		name: "fast",
		executeFn: func(input map[string]any) (*Result, error) {
			return TextResult("ok"), nil
		},
	}

	wrapped := ApplyMiddleware(base, WithTimeout(1*time.Second))

	result, err := wrapped.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestWithTimeout_TimesOut(t *testing.T) {
	base := &mockTool{
		name: "slow",
		executeCtxFn: func(ctx context.Context, input map[string]any) (*Result, error) {
			// Wait for context to be cancelled by the timeout middleware.
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	wrapped := ApplyMiddleware(base, WithTimeout(50*time.Millisecond))

	_, err := wrapped.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestWithTimeout_PreservesMetadata(t *testing.T) {
	base := &mockTool{
		name:        "test",
		description: "Test tool",
		inputSchema: map[string]any{"type": "object"},
	}

	wrapped := ApplyMiddleware(base, WithTimeout(1*time.Second))

	if wrapped.Name() != "test" {
		t.Errorf("Name() = %q, want %q", wrapped.Name(), "test")
	}
	if wrapped.Description() != "Test tool" {
		t.Errorf("Description() = %q, want %q", wrapped.Description(), "Test tool")
	}
	if wrapped.InputSchema() == nil {
		t.Error("InputSchema() should not be nil")
	}
}

func TestWithRetry_SucceedsImmediately(t *testing.T) {
	calls := 0
	base := &mockTool{
		name: "once",
		executeFn: func(input map[string]any) (*Result, error) {
			calls++
			return TextResult("ok"), nil
		},
	}

	wrapped := ApplyMiddleware(base, WithRetry(3))
	result, err := wrapped.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestWithRetry_RetriesOnRetryableError(t *testing.T) {
	calls := 0
	retryableErr := core.NewError("test", core.ErrRateLimit, "rate limited", nil)

	base := &mockTool{
		name: "flaky",
		executeFn: func(input map[string]any) (*Result, error) {
			calls++
			if calls < 3 {
				return nil, retryableErr
			}
			return TextResult("ok"), nil
		},
	}

	wrapped := ApplyMiddleware(base, WithRetry(5))
	result, err := wrapped.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestWithRetry_StopsOnNonRetryable(t *testing.T) {
	nonRetryable := errors.New("permanent error")
	calls := 0

	base := &mockTool{
		name: "permanent-fail",
		executeFn: func(input map[string]any) (*Result, error) {
			calls++
			return nil, nonRetryable
		},
	}

	wrapped := ApplyMiddleware(base, WithRetry(5))
	_, err := wrapped.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry on non-retryable), got %d", calls)
	}
}

func TestWithRetry_ExhaustsAttempts(t *testing.T) {
	retryableErr := core.NewError("test", core.ErrTimeout, "timeout", nil)
	calls := 0

	base := &mockTool{
		name: "always-fail",
		executeFn: func(input map[string]any) (*Result, error) {
			calls++
			return nil, retryableErr
		},
	}

	wrapped := ApplyMiddleware(base, WithRetry(3))
	_, err := wrapped.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestWithRetry_RespectsContextCancellation(t *testing.T) {
	retryableErr := core.NewError("test", core.ErrProviderDown, "down", nil)
	calls := 0

	base := &mockTool{
		name: "cancelable",
		executeFn: func(input map[string]any) (*Result, error) {
			calls++
			return nil, retryableErr
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	wrapped := ApplyMiddleware(base, WithRetry(10))
	_, err := wrapped.Execute(ctx, nil)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	// Should stop after first attempt due to cancelled context.
	if calls > 2 {
		t.Errorf("expected at most 2 calls with cancelled context, got %d", calls)
	}
}

func TestApplyMiddleware_Order(t *testing.T) {
	var order []string

	mw1 := func(next Tool) Tool {
		return &orderTool{
			Tool:   next,
			before: func() { order = append(order, "mw1") },
		}
	}
	mw2 := func(next Tool) Tool {
		return &orderTool{
			Tool:   next,
			before: func() { order = append(order, "mw2") },
		}
	}

	base := &mockTool{name: "base"}
	wrapped := ApplyMiddleware(base, mw1, mw2)

	_, _ = wrapped.Execute(context.Background(), nil)

	// mw1 is outermost, executed first.
	if len(order) < 2 || order[0] != "mw1" || order[1] != "mw2" {
		t.Errorf("expected order [mw1, mw2], got %v", order)
	}
}

// orderTool tracks execution order for middleware testing.
type orderTool struct {
	Tool
	before func()
}

func (o *orderTool) Execute(ctx context.Context, input map[string]any) (*Result, error) {
	o.before()
	return o.Tool.Execute(ctx, input)
}

// TestWithRetry_BacksOffBetweenAttempts is a regression test for FWC-26: the
// retry loop had no delay, which immediately exhausted rate-limited tools.
func TestWithRetry_BacksOffBetweenAttempts(t *testing.T) {
	var times []time.Time
	base := &mockTool{name: "rl", executeFn: func(map[string]any) (*Result, error) {
		times = append(times, time.Now())
		return nil, core.NewError("t", core.ErrRateLimit, "slow down", nil)
	}}
	wrapped := ApplyMiddleware(base, WithRetryBackoff(3, 20*time.Millisecond, 40*time.Millisecond))
	if _, err := wrapped.Execute(context.Background(), nil); err == nil {
		t.Fatal("expected error")
	}
	if len(times) != 3 {
		t.Fatalf("calls = %d, want 3", len(times))
	}
	if gap := times[1].Sub(times[0]); gap < 10*time.Millisecond {
		t.Errorf("first retry after %v, want >= 10ms backoff", gap)
	}
	if gap := times[2].Sub(times[1]); gap < 20*time.Millisecond {
		t.Errorf("second retry after %v, want >= 20ms (doubled) backoff", gap)
	}
}

// TestWithRetry_NonPositiveAttemptsRunsOnce is a regression test for FWC-26:
// maxAttempts <= 0 never ran the tool and returned a "%w" of nil.
func TestWithRetry_NonPositiveAttemptsRunsOnce(t *testing.T) {
	for _, n := range []int{0, -3} {
		calls := 0
		base := &mockTool{name: "x", executeFn: func(map[string]any) (*Result, error) {
			calls++
			return TextResult("ok"), nil
		}}
		res, err := ApplyMiddleware(base, WithRetry(n)).Execute(context.Background(), nil)
		if err != nil || res == nil || calls != 1 {
			t.Errorf("WithRetry(%d): calls=%d res=%v err=%v, want one successful call", n, calls, res, err)
		}
	}
}

func TestWithRetry_CancelDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	base := &mockTool{name: "x", executeFn: func(map[string]any) (*Result, error) {
		cancel()
		return nil, core.NewError("t", core.ErrTimeout, "slow", nil)
	}}
	_, err := ApplyMiddleware(base, WithRetryBackoff(3, time.Hour, time.Hour)).Execute(ctx, nil)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if d := nextRetryBackoff(time.Second, 0); d != 2*time.Second {
		t.Errorf("uncapped backoff = %v", d)
	}
	if err := sleepWithContext(context.Background(), 0); err != nil {
		t.Errorf("sleepWithContext(0) = %v", err)
	}
}
