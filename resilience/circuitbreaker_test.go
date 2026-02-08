package resilience

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNewCircuitBreaker_Defaults(t *testing.T) {
	cb := NewCircuitBreaker(0, 0)
	if cb.failureThreshold != 5 {
		t.Errorf("failureThreshold = %d, want 5 (default)", cb.failureThreshold)
	}
	if cb.resetTimeout != 30*time.Second {
		t.Errorf("resetTimeout = %v, want 30s (default)", cb.resetTimeout)
	}
	if cb.State() != StateClosed {
		t.Errorf("State() = %q, want %q", cb.State(), StateClosed)
	}
}

func TestNewCircuitBreaker_CustomValues(t *testing.T) {
	cb := NewCircuitBreaker(3, 5*time.Second)
	if cb.failureThreshold != 3 {
		t.Errorf("failureThreshold = %d, want 3", cb.failureThreshold)
	}
	if cb.resetTimeout != 5*time.Second {
		t.Errorf("resetTimeout = %v, want 5s", cb.resetTimeout)
	}
}

func TestCircuitBreaker_ClosedState_Success(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Second)

	result, err := cb.Execute(context.Background(), func(_ context.Context) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %v, want %q", result, "ok")
	}
	if cb.State() != StateClosed {
		t.Errorf("State() = %q, want %q", cb.State(), StateClosed)
	}
}

func TestCircuitBreaker_ClosedToOpen(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Second)

	for i := 0; i < 3; i++ {
		_, _ = cb.Execute(context.Background(), func(_ context.Context) (any, error) {
			return nil, fmt.Errorf("failure %d", i)
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("State() = %q, want %q after %d failures", cb.State(), StateOpen, 3)
	}
}

func TestCircuitBreaker_OpenRejectsRequests(t *testing.T) {
	cb := NewCircuitBreaker(1, time.Hour) // Long reset so we stay open.

	// Trip the breaker.
	_, _ = cb.Execute(context.Background(), func(_ context.Context) (any, error) {
		return nil, fmt.Errorf("fail")
	})

	if cb.State() != StateOpen {
		t.Fatalf("State() = %q, want %q", cb.State(), StateOpen)
	}

	// Subsequent calls should fail immediately.
	_, err := cb.Execute(context.Background(), func(_ context.Context) (any, error) {
		t.Error("function should not be called when circuit is open")
		return nil, nil
	})
	if err != ErrCircuitOpen {
		t.Errorf("error = %v, want ErrCircuitOpen", err)
	}
}

func TestCircuitBreaker_OpenToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(1, 10*time.Millisecond)

	// Trip the breaker.
	_, _ = cb.Execute(context.Background(), func(_ context.Context) (any, error) {
		return nil, fmt.Errorf("fail")
	})

	if cb.State() != StateOpen {
		t.Fatalf("State() = %q, want %q", cb.State(), StateOpen)
	}

	// Wait for reset timeout.
	time.Sleep(20 * time.Millisecond)

	if cb.State() != StateHalfOpen {
		t.Errorf("State() = %q, want %q after reset timeout", cb.State(), StateHalfOpen)
	}
}

func TestCircuitBreaker_HalfOpenToClosed(t *testing.T) {
	cb := NewCircuitBreaker(1, 10*time.Millisecond)

	// Trip the breaker.
	_, _ = cb.Execute(context.Background(), func(_ context.Context) (any, error) {
		return nil, fmt.Errorf("fail")
	})

	// Wait for reset timeout.
	time.Sleep(20 * time.Millisecond)

	// Probe call succeeds → should close.
	result, err := cb.Execute(context.Background(), func(_ context.Context) (any, error) {
		return "recovered", nil
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "recovered" {
		t.Errorf("result = %v, want %q", result, "recovered")
	}
	if cb.State() != StateClosed {
		t.Errorf("State() = %q, want %q after successful probe", cb.State(), StateClosed)
	}
}

func TestCircuitBreaker_HalfOpenToOpen(t *testing.T) {
	cb := NewCircuitBreaker(1, 10*time.Millisecond)

	// Trip the breaker.
	_, _ = cb.Execute(context.Background(), func(_ context.Context) (any, error) {
		return nil, fmt.Errorf("fail")
	})

	// Wait for reset timeout.
	time.Sleep(20 * time.Millisecond)

	// Probe call fails → should go back to open.
	_, _ = cb.Execute(context.Background(), func(_ context.Context) (any, error) {
		return nil, fmt.Errorf("probe failed")
	})

	if cb.State() != StateOpen {
		t.Errorf("State() = %q, want %q after failed probe", cb.State(), StateOpen)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(1, time.Hour)

	// Trip the breaker.
	_, _ = cb.Execute(context.Background(), func(_ context.Context) (any, error) {
		return nil, fmt.Errorf("fail")
	})
	if cb.State() != StateOpen {
		t.Fatalf("State() = %q, want %q", cb.State(), StateOpen)
	}

	// Manually reset.
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("State() = %q, want %q after Reset()", cb.State(), StateClosed)
	}

	// Should be able to execute again.
	result, err := cb.Execute(context.Background(), func(_ context.Context) (any, error) {
		return "after reset", nil
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "after reset" {
		t.Errorf("result = %v, want %q", result, "after reset")
	}
}

func TestCircuitBreaker_SuccessResetsFailureCount(t *testing.T) {
	cb := NewCircuitBreaker(3, time.Second)

	// Two failures.
	for i := 0; i < 2; i++ {
		_, _ = cb.Execute(context.Background(), func(_ context.Context) (any, error) {
			return nil, fmt.Errorf("fail")
		})
	}

	// One success resets failure count.
	_, _ = cb.Execute(context.Background(), func(_ context.Context) (any, error) {
		return "ok", nil
	})

	// Two more failures should not trip (counter was reset).
	for i := 0; i < 2; i++ {
		_, _ = cb.Execute(context.Background(), func(_ context.Context) (any, error) {
			return nil, fmt.Errorf("fail")
		})
	}

	if cb.State() != StateClosed {
		t.Errorf("State() = %q, want %q (failure counter was reset by success)", cb.State(), StateClosed)
	}

	// One more failure should trip it (3 consecutive).
	_, _ = cb.Execute(context.Background(), func(_ context.Context) (any, error) {
		return nil, fmt.Errorf("fail")
	})

	if cb.State() != StateOpen {
		t.Errorf("State() = %q, want %q (3rd consecutive failure)", cb.State(), StateOpen)
	}
}

func TestCircuitBreaker_State_Values(t *testing.T) {
	states := map[State]string{
		StateClosed:   "closed",
		StateOpen:     "open",
		StateHalfOpen: "half_open",
	}
	for state, want := range states {
		if string(state) != want {
			t.Errorf("State %v = %q, want %q", state, string(state), want)
		}
	}
}

func TestCircuitBreaker_ErrorPassedThrough(t *testing.T) {
	cb := NewCircuitBreaker(5, time.Second)

	expectedErr := fmt.Errorf("specific error")
	_, err := cb.Execute(context.Background(), func(_ context.Context) (any, error) {
		return nil, expectedErr
	})

	if err != expectedErr {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}
}

func TestCircuitBreaker_FullCycle(t *testing.T) {
	cb := NewCircuitBreaker(2, 10*time.Millisecond)

	// Phase 1: Closed → fail twice → Open.
	for i := 0; i < 2; i++ {
		_, _ = cb.Execute(context.Background(), func(_ context.Context) (any, error) {
			return nil, fmt.Errorf("fail")
		})
	}
	if cb.State() != StateOpen {
		t.Fatalf("phase 1: State() = %q, want %q", cb.State(), StateOpen)
	}

	// Phase 2: Open → wait → Half-Open.
	time.Sleep(20 * time.Millisecond)
	if cb.State() != StateHalfOpen {
		t.Fatalf("phase 2: State() = %q, want %q", cb.State(), StateHalfOpen)
	}

	// Phase 3: Half-Open → probe fail → Open.
	_, _ = cb.Execute(context.Background(), func(_ context.Context) (any, error) {
		return nil, fmt.Errorf("probe fail")
	})
	if cb.State() != StateOpen {
		t.Fatalf("phase 3: State() = %q, want %q", cb.State(), StateOpen)
	}

	// Phase 4: Open → wait → Half-Open → probe success → Closed.
	time.Sleep(20 * time.Millisecond)
	_, err := cb.Execute(context.Background(), func(_ context.Context) (any, error) {
		return "recovered", nil
	})
	if err != nil {
		t.Fatalf("phase 4: Execute() error = %v", err)
	}
	if cb.State() != StateClosed {
		t.Fatalf("phase 4: State() = %q, want %q", cb.State(), StateClosed)
	}
}
