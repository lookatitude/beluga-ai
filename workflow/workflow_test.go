package workflow

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestWorkflowStatus(t *testing.T) {
	tests := []struct {
		s    WorkflowStatus
		want string
	}{
		{StatusRunning, "running"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{StatusCanceled, "canceled"},
	}
	for _, tt := range tests {
		if string(tt.s) != tt.want {
			t.Errorf("got %q, want %q", string(tt.s), tt.want)
		}
	}
}

func TestEventTypes(t *testing.T) {
	events := []EventType{
		EventWorkflowStarted, EventWorkflowCompleted, EventWorkflowFailed,
		EventWorkflowCanceled, EventActivityStarted, EventActivityCompleted,
		EventActivityFailed, EventSignalReceived, EventTimerFired,
	}
	for _, e := range events {
		if string(e) == "" {
			t.Error("expected non-empty event type")
		}
	}
}

func TestDefaultRetryPolicy(t *testing.T) {
	p := DefaultRetryPolicy()
	if p.MaxAttempts != 3 {
		t.Errorf("expected 3 attempts, got %d", p.MaxAttempts)
	}
	if p.InitialInterval != 100*time.Millisecond {
		t.Errorf("expected 100ms, got %v", p.InitialInterval)
	}
	if p.BackoffCoefficient != 2.0 {
		t.Errorf("expected 2.0, got %f", p.BackoffCoefficient)
	}
}

func TestComputeInterval(t *testing.T) {
	p := RetryPolicy{
		InitialInterval:    100 * time.Millisecond,
		BackoffCoefficient: 2.0,
		MaxInterval:        1 * time.Second,
	}

	i0 := computeInterval(p, 0)
	if i0 != 100*time.Millisecond {
		t.Errorf("attempt 0: expected 100ms, got %v", i0)
	}

	i1 := computeInterval(p, 1)
	if i1 != 200*time.Millisecond {
		t.Errorf("attempt 1: expected 200ms, got %v", i1)
	}

	// Should be capped at MaxInterval.
	i10 := computeInterval(p, 10)
	if i10 > 1*time.Second {
		t.Errorf("attempt 10: expected <= 1s, got %v", i10)
	}
}

func TestExecuteWithRetry_Success(t *testing.T) {
	count := 0
	err := executeWithRetry(context.Background(), RetryPolicy{
		MaxAttempts:        3,
		InitialInterval:   1 * time.Millisecond,
		BackoffCoefficient: 2.0,
	}, func(_ context.Context) error {
		count++
		if count < 2 {
			return fmt.Errorf("temporary error")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 attempts, got %d", count)
	}
}

func TestExecuteWithRetry_Exhausted(t *testing.T) {
	err := executeWithRetry(context.Background(), RetryPolicy{
		MaxAttempts:        2,
		InitialInterval:   1 * time.Millisecond,
		BackoffCoefficient: 2.0,
	}, func(_ context.Context) error {
		return fmt.Errorf("persistent error")
	})
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
}

func TestExecuteWithRetry_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := executeWithRetry(ctx, RetryPolicy{
		MaxAttempts:        5,
		InitialInterval:   1 * time.Second,
		BackoffCoefficient: 2.0,
	}, func(_ context.Context) error {
		return fmt.Errorf("error")
	})
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
}

func TestRegistry_Default(t *testing.T) {
	names := List()
	found := false
	for _, n := range names {
		if n == "default" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'default' in registry")
	}

	exec, err := New("default", Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if exec == nil {
		t.Fatal("expected non-nil executor")
	}
}

func TestRegistry_Unknown(t *testing.T) {
	_, err := New("nonexistent-wf-provider", Config{})
	if err == nil {
		t.Fatal("expected error for unknown executor")
	}
}
