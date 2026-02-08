package resilience

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestHedge_PrimarySucceedsBeforeDelay(t *testing.T) {
	secondaryCalled := false
	result, err := Hedge(
		context.Background(),
		func(_ context.Context) (string, error) {
			return "primary", nil
		},
		func(_ context.Context) (string, error) {
			secondaryCalled = true
			return "secondary", nil
		},
		time.Second, // long delay; primary should win
	)

	if err != nil {
		t.Fatalf("Hedge() error = %v", err)
	}
	if result != "primary" {
		t.Errorf("result = %q, want %q", result, "primary")
	}
	if secondaryCalled {
		t.Error("secondary should not be called when primary succeeds before delay")
	}
}

func TestHedge_SecondaryStartsAfterDelay(t *testing.T) {
	var secondaryStarted atomic.Bool

	result, err := Hedge(
		context.Background(),
		func(ctx context.Context) (string, error) {
			// Slow primary — wait long enough for secondary to start and finish.
			select {
			case <-time.After(200 * time.Millisecond):
				return "primary", nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		},
		func(ctx context.Context) (string, error) {
			secondaryStarted.Store(true)
			return "secondary", nil
		},
		10*time.Millisecond, // short delay
	)

	if err != nil {
		t.Fatalf("Hedge() error = %v", err)
	}
	if result != "secondary" {
		t.Errorf("result = %q, want %q", result, "secondary")
	}
	if !secondaryStarted.Load() {
		t.Error("secondary should have been started after delay")
	}
}

func TestHedge_PrimaryFailsBeforeDelay_SecondaryStarts(t *testing.T) {
	result, err := Hedge(
		context.Background(),
		func(_ context.Context) (string, error) {
			return "", errors.New("primary failed")
		},
		func(_ context.Context) (string, error) {
			return "secondary", nil
		},
		time.Second, // long delay, but primary fails immediately
	)

	if err != nil {
		t.Fatalf("Hedge() error = %v", err)
	}
	if result != "secondary" {
		t.Errorf("result = %q, want %q", result, "secondary")
	}
}

func TestHedge_BothFail_ReturnsPrimaryError(t *testing.T) {
	primaryErr := errors.New("primary error")
	_, err := Hedge(
		context.Background(),
		func(_ context.Context) (string, error) {
			return "", primaryErr
		},
		func(_ context.Context) (string, error) {
			return "", errors.New("secondary error")
		},
		time.Millisecond,
	)

	if err == nil {
		t.Fatal("Hedge() expected error when both fail")
	}
	// The implementation returns primary error when primary fails before delay.
	if err != primaryErr {
		t.Errorf("error = %v, want %v (primary error)", err, primaryErr)
	}
}

func TestHedge_BothFail_AfterDelay(t *testing.T) {
	_, err := Hedge(
		context.Background(),
		func(ctx context.Context) (string, error) {
			// Slow enough for delay to fire.
			select {
			case <-time.After(50 * time.Millisecond):
				return "", errors.New("primary failed")
			case <-ctx.Done():
				return "", ctx.Err()
			}
		},
		func(_ context.Context) (string, error) {
			return "", errors.New("secondary failed")
		},
		10*time.Millisecond,
	)

	if err == nil {
		t.Fatal("Hedge() expected error when both fail after delay")
	}
}

func TestHedge_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	_, err := Hedge(
		ctx,
		func(ctx context.Context) (string, error) {
			return "primary", nil
		},
		func(ctx context.Context) (string, error) {
			return "secondary", nil
		},
		10*time.Millisecond,
	)

	// With a cancelled context, the primary might succeed or context error might win.
	// Either way, it should not hang.
	_ = err
}

func TestHedge_ZeroDelay(t *testing.T) {
	// With zero delay, secondary should start immediately alongside primary.
	// Secondary returns faster, so it should win.
	result, err := Hedge(
		context.Background(),
		func(ctx context.Context) (string, error) {
			select {
			case <-time.After(100 * time.Millisecond):
				return "primary", nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		},
		func(_ context.Context) (string, error) {
			return "secondary", nil
		},
		0, // zero delay
	)

	if err != nil {
		t.Fatalf("Hedge() error = %v", err)
	}
	if result != "secondary" {
		t.Errorf("result = %q, want %q (secondary should be faster)", result, "secondary")
	}
}

func TestHedge_PrimaryFailsBeforeDelay_SecondaryAlsoFails(t *testing.T) {
	primaryErr := errors.New("primary boom")

	_, err := Hedge(
		context.Background(),
		func(_ context.Context) (string, error) {
			return "", primaryErr
		},
		func(_ context.Context) (string, error) {
			return "", errors.New("secondary boom")
		},
		time.Second, // long delay, primary fails first
	)

	if err == nil {
		t.Fatal("Hedge() expected error when both fail")
	}
	if err != primaryErr {
		t.Errorf("error = %v, want %v (primary error)", err, primaryErr)
	}
}

func TestHedge_GenericTypes(t *testing.T) {
	// Test with int type.
	result, err := Hedge(
		context.Background(),
		func(_ context.Context) (int, error) {
			return 42, nil
		},
		func(_ context.Context) (int, error) {
			return 0, errors.New("unused")
		},
		time.Second,
	)
	if err != nil {
		t.Fatalf("Hedge() error = %v", err)
	}
	if result != 42 {
		t.Errorf("result = %d, want 42", result)
	}
}

func TestHedge_SecondaryWinsWhenBothRunning(t *testing.T) {
	// Primary is slow, secondary is fast, both run after delay.
	result, err := Hedge(
		context.Background(),
		func(ctx context.Context) (string, error) {
			select {
			case <-time.After(200 * time.Millisecond):
				return "primary", nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		},
		func(_ context.Context) (string, error) {
			time.Sleep(5 * time.Millisecond)
			return "secondary", nil
		},
		10*time.Millisecond,
	)

	if err != nil {
		t.Fatalf("Hedge() error = %v", err)
	}
	if result != "secondary" {
		t.Errorf("result = %q, want %q", result, "secondary")
	}
}
