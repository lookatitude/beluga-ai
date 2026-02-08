package tool

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/core"
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
			Tool: next,
			before: func() { order = append(order, "mw1") },
		}
	}
	mw2 := func(next Tool) Tool {
		return &orderTool{
			Tool: next,
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
