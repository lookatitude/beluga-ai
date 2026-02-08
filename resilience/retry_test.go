package resilience

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/core"
)

func TestRetry_SucceedsFirstAttempt(t *testing.T) {
	calls := 0
	result, err := Retry(context.Background(), RetryPolicy{MaxAttempts: 3, InitialBackoff: time.Millisecond}, func(_ context.Context) (string, error) {
		calls++
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("Retry() error = %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestRetry_SucceedsAfterRetries(t *testing.T) {
	var calls atomic.Int32
	result, err := Retry(context.Background(), RetryPolicy{
		MaxAttempts:    5,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     10 * time.Millisecond,
		BackoffFactor:  1.5,
		Jitter:         false,
	}, func(_ context.Context) (string, error) {
		n := calls.Add(1)
		if n < 3 {
			return "", core.NewError("op", core.ErrRateLimit, "throttled", nil)
		}
		return "success", nil
	})

	if err != nil {
		t.Fatalf("Retry() error = %v", err)
	}
	if result != "success" {
		t.Errorf("result = %q, want %q", result, "success")
	}
	if got := calls.Load(); got != 3 {
		t.Errorf("calls = %d, want 3", got)
	}
}

func TestRetry_ExhaustsAttempts(t *testing.T) {
	calls := 0
	_, err := Retry(context.Background(), RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		BackoffFactor:  1.0,
	}, func(_ context.Context) (int, error) {
		calls++
		return 0, core.NewError("op", core.ErrTimeout, "timed out", nil)
	})

	if err == nil {
		t.Fatal("Retry() expected error after exhausting attempts")
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestRetry_NonRetryableError(t *testing.T) {
	calls := 0
	_, err := Retry(context.Background(), RetryPolicy{
		MaxAttempts:    5,
		InitialBackoff: time.Millisecond,
	}, func(_ context.Context) (int, error) {
		calls++
		return 0, core.NewError("op", core.ErrAuth, "unauthorized", nil)
	})

	if err == nil {
		t.Fatal("Retry() expected error for non-retryable error")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (no retries for non-retryable)", calls)
	}
}

func TestRetry_PlainErrorNotRetryable(t *testing.T) {
	calls := 0
	_, err := Retry(context.Background(), RetryPolicy{
		MaxAttempts:    5,
		InitialBackoff: time.Millisecond,
	}, func(_ context.Context) (int, error) {
		calls++
		return 0, fmt.Errorf("plain error")
	})

	if err == nil {
		t.Fatal("Retry() expected error")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (plain errors not retryable)", calls)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := Retry(ctx, RetryPolicy{
		MaxAttempts:    100,
		InitialBackoff: 50 * time.Millisecond,
	}, func(_ context.Context) (int, error) {
		calls++
		return 0, core.NewError("op", core.ErrRateLimit, "throttled", nil)
	})

	if err == nil {
		t.Fatal("Retry() expected context cancellation error")
	}
	if err != context.Canceled {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestRetry_CustomRetryableErrors(t *testing.T) {
	calls := 0
	_, err := Retry(context.Background(), RetryPolicy{
		MaxAttempts:     3,
		InitialBackoff:  time.Millisecond,
		RetryableErrors: []core.ErrorCode{core.ErrAuth}, // Make auth errors retryable.
	}, func(_ context.Context) (int, error) {
		calls++
		return 0, core.NewError("op", core.ErrAuth, "auth failed", nil)
	})

	if err == nil {
		t.Fatal("Retry() expected error after exhausting attempts")
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3 (auth made retryable)", calls)
	}
}

func TestRetry_DefaultPolicyNormalization(t *testing.T) {
	// Zero policy should be normalized to defaults.
	calls := 0
	_, err := Retry(context.Background(), RetryPolicy{}, func(_ context.Context) (int, error) {
		calls++
		return 0, core.NewError("op", core.ErrTimeout, "timeout", nil)
	})

	if err == nil {
		t.Fatal("Retry() expected error")
	}
	// Default is 3 attempts.
	if calls != 3 {
		t.Errorf("calls = %d, want 3 (default MaxAttempts)", calls)
	}
}

func TestRetry_SingleAttempt(t *testing.T) {
	calls := 0
	_, err := Retry(context.Background(), RetryPolicy{
		MaxAttempts:    1,
		InitialBackoff: time.Millisecond,
	}, func(_ context.Context) (int, error) {
		calls++
		return 0, core.NewError("op", core.ErrRateLimit, "throttled", nil)
	})

	if err == nil {
		t.Fatal("Retry() expected error")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (MaxAttempts=1 means no retries)", calls)
	}
}

func TestDefaultRetryPolicy(t *testing.T) {
	p := DefaultRetryPolicy()

	if p.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %d, want 3", p.MaxAttempts)
	}
	if p.InitialBackoff != 500*time.Millisecond {
		t.Errorf("InitialBackoff = %v, want 500ms", p.InitialBackoff)
	}
	if p.MaxBackoff != 30*time.Second {
		t.Errorf("MaxBackoff = %v, want 30s", p.MaxBackoff)
	}
	if p.BackoffFactor != 2.0 {
		t.Errorf("BackoffFactor = %f, want 2.0", p.BackoffFactor)
	}
	if !p.Jitter {
		t.Error("Jitter = false, want true")
	}
}

func TestRetry_BackoffGrowth(t *testing.T) {
	// Verify that subsequent retries have increasing delays.
	var timestamps []time.Time

	_, _ = Retry(context.Background(), RetryPolicy{
		MaxAttempts:    4,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
		BackoffFactor:  2.0,
		Jitter:         false,
	}, func(_ context.Context) (int, error) {
		timestamps = append(timestamps, time.Now())
		return 0, core.NewError("op", core.ErrRateLimit, "throttled", nil)
	})

	if len(timestamps) != 4 {
		t.Fatalf("expected 4 timestamps, got %d", len(timestamps))
	}

	// Verify increasing gaps (with some tolerance for scheduler delays).
	for i := 2; i < len(timestamps); i++ {
		gap := timestamps[i].Sub(timestamps[i-1])
		prevGap := timestamps[i-1].Sub(timestamps[i-2])
		// The gap should generally be larger (backoff * 2), but allow tolerance.
		if gap < prevGap/2 {
			t.Errorf("gap[%d]=%v should be >= gap[%d]=%v (backoff growth)", i, gap, i-1, prevGap)
		}
	}
}
