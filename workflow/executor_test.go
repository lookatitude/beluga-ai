package workflow

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// mockStore implements WorkflowStore for testing.
type mockStore struct {
	states map[string]WorkflowState
}

func newMockStore() *mockStore {
	return &mockStore{states: make(map[string]WorkflowState)}
}

func (s *mockStore) Save(_ context.Context, state WorkflowState) error {
	s.states[state.WorkflowID] = state
	return nil
}

func (s *mockStore) Load(_ context.Context, id string) (*WorkflowState, error) {
	state, ok := s.states[id]
	if !ok {
		return nil, nil
	}
	return &state, nil
}

func (s *mockStore) List(_ context.Context, filter WorkflowFilter) ([]WorkflowState, error) {
	var results []WorkflowState
	for _, state := range s.states {
		if filter.Status != "" && state.Status != filter.Status {
			continue
		}
		results = append(results, state)
	}
	return results, nil
}

func (s *mockStore) Delete(_ context.Context, id string) error {
	delete(s.states, id)
	return nil
}

func TestExecutor_SimpleWorkflow(t *testing.T) {
	store := newMockStore()
	exec := NewExecutor(WithStore(store))

	handle, err := exec.Execute(context.Background(), func(ctx WorkflowContext, input any) (any, error) {
		return fmt.Sprintf("result: %v", input), nil
	}, WorkflowOptions{
		ID:    "wf-simple",
		Input: "hello",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if handle.ID() != "wf-simple" {
		t.Errorf("expected ID 'wf-simple', got %q", handle.ID())
	}
	if handle.RunID() == "" {
		t.Error("expected non-empty RunID")
	}

	result, err := handle.Result(context.Background())
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	if result != "result: hello" {
		t.Errorf("expected 'result: hello', got %v", result)
	}
	if handle.Status() != StatusCompleted {
		t.Errorf("expected completed, got %s", handle.Status())
	}
}

func TestExecutor_FailingWorkflow(t *testing.T) {
	exec := NewExecutor()

	handle, err := exec.Execute(context.Background(), func(ctx WorkflowContext, input any) (any, error) {
		return nil, fmt.Errorf("workflow failed")
	}, WorkflowOptions{ID: "wf-fail"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	_, err = handle.Result(context.Background())
	if err == nil {
		t.Fatal("expected error from failing workflow")
	}
	if handle.Status() != StatusFailed {
		t.Errorf("expected failed, got %s", handle.Status())
	}
}

func TestExecutor_Cancel(t *testing.T) {
	exec := NewExecutor()

	handle, err := exec.Execute(context.Background(), func(ctx WorkflowContext, _ any) (any, error) {
		if err := ctx.Sleep(10 * time.Second); err != nil {
			return nil, err
		}
		return "done", nil
	}, WorkflowOptions{ID: "wf-cancel"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Wait for it to start.
	time.Sleep(50 * time.Millisecond)

	if err := exec.Cancel(context.Background(), "wf-cancel"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	_, err = handle.Result(context.Background())
	if err == nil {
		t.Fatal("expected error after cancel")
	}
}

func TestExecutor_CancelNotFound(t *testing.T) {
	exec := NewExecutor()
	if exec.Cancel(context.Background(), "nonexistent") == nil {
		t.Fatal("expected error for nonexistent workflow")
	}
}

func TestExecutor_Signal(t *testing.T) {
	exec := NewExecutor()

	handle, err := exec.Execute(context.Background(), func(ctx WorkflowContext, _ any) (any, error) {
		sigCh := ctx.ReceiveSignal("approval")
		select {
		case payload := <-sigCh:
			return fmt.Sprintf("approved: %v", payload), nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}, WorkflowOptions{ID: "wf-signal"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// Wait for it to start.
	time.Sleep(50 * time.Millisecond)

	if err := exec.Signal(context.Background(), "wf-signal", Signal{Name: "approval", Payload: true}); err != nil {
		t.Fatalf("Signal: %v", err)
	}

	result, err := handle.Result(context.Background())
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	if result != "approved: true" {
		t.Errorf("expected 'approved: true', got %v", result)
	}
}

func TestExecutor_SignalNotFound(t *testing.T) {
	exec := NewExecutor()
	if exec.Signal(context.Background(), "nonexistent", Signal{Name: "test"}) == nil {
		t.Fatal("expected error for nonexistent workflow")
	}
}

func TestExecutor_Query(t *testing.T) {
	exec := NewExecutor()

	handle, _ := exec.Execute(context.Background(), func(ctx WorkflowContext, _ any) (any, error) {
		ctx.Sleep(1 * time.Second)
		return "done", nil
	}, WorkflowOptions{ID: "wf-query"})
	_ = handle

	time.Sleep(50 * time.Millisecond)

	status, err := exec.Query(context.Background(), "wf-query", "status")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if status != StatusRunning {
		t.Errorf("expected running, got %v", status)
	}

	// Unknown query type.
	_, err = exec.Query(context.Background(), "wf-query", "unknown")
	if err == nil {
		t.Fatal("expected error for unknown query type")
	}
}

func TestExecutor_QueryNotFound(t *testing.T) {
	exec := NewExecutor()
	_, err := exec.Query(context.Background(), "nonexistent", "status")
	if err == nil {
		t.Fatal("expected error for nonexistent workflow")
	}
}

func TestExecutor_Activity(t *testing.T) {
	exec := NewExecutor()

	handle, err := exec.Execute(context.Background(), func(ctx WorkflowContext, input any) (any, error) {
		result, err := ctx.ExecuteActivity(func(_ context.Context, in any) (any, error) {
			return fmt.Sprintf("processed: %v", in), nil
		}, input)
		if err != nil {
			return nil, err
		}
		return result, nil
	}, WorkflowOptions{ID: "wf-activity", Input: "data"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	result, err := handle.Result(context.Background())
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	if result != "processed: data" {
		t.Errorf("expected 'processed: data', got %v", result)
	}
}

func TestExecutor_ActivityWithRetry(t *testing.T) {
	exec := NewExecutor()
	attempt := 0

	handle, err := exec.Execute(context.Background(), func(ctx WorkflowContext, _ any) (any, error) {
		return ctx.ExecuteActivity(func(_ context.Context, _ any) (any, error) {
			attempt++
			if attempt < 3 {
				return nil, fmt.Errorf("temporary error")
			}
			return "success", nil
		}, nil, WithActivityRetry(RetryPolicy{
			MaxAttempts:        5,
			InitialInterval:   1 * time.Millisecond,
			BackoffCoefficient: 1.5,
		}))
	}, WorkflowOptions{ID: "wf-retry"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	result, err := handle.Result(context.Background())
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %v", result)
	}
	if attempt != 3 {
		t.Errorf("expected 3 attempts, got %d", attempt)
	}
}

func TestExecutor_ActivityTimeout(t *testing.T) {
	exec := NewExecutor()

	handle, err := exec.Execute(context.Background(), func(ctx WorkflowContext, _ any) (any, error) {
		return ctx.ExecuteActivity(func(actCtx context.Context, _ any) (any, error) {
			select {
			case <-time.After(10 * time.Second):
				return "done", nil
			case <-actCtx.Done():
				return nil, actCtx.Err()
			}
		}, nil, WithActivityTimeout(50*time.Millisecond))
	}, WorkflowOptions{ID: "wf-act-timeout"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	_, err = handle.Result(context.Background())
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestExecutor_Sleep(t *testing.T) {
	exec := NewExecutor()

	start := time.Now()
	handle, err := exec.Execute(context.Background(), func(ctx WorkflowContext, _ any) (any, error) {
		if err := ctx.Sleep(50 * time.Millisecond); err != nil {
			return nil, err
		}
		return "awake", nil
	}, WorkflowOptions{ID: "wf-sleep"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	result, err := handle.Result(context.Background())
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	if result != "awake" {
		t.Errorf("expected 'awake', got %v", result)
	}
	if time.Since(start) < 40*time.Millisecond {
		t.Error("expected sleep to take at least 40ms")
	}
}

func TestExecutor_Hooks(t *testing.T) {
	var started, completed bool
	exec := NewExecutor(WithExecutorHooks(Hooks{
		OnWorkflowStart: func(_ context.Context, _ string, _ any) {
			started = true
		},
		OnWorkflowComplete: func(_ context.Context, _ string, _ any) {
			completed = true
		},
	}))

	handle, _ := exec.Execute(context.Background(), func(ctx WorkflowContext, _ any) (any, error) {
		return "done", nil
	}, WorkflowOptions{ID: "wf-hooks"})

	handle.Result(context.Background())

	if !started {
		t.Error("expected OnWorkflowStart to be called")
	}
	if !completed {
		t.Error("expected OnWorkflowComplete to be called")
	}
}

func TestExecutor_HooksOnFail(t *testing.T) {
	var failed bool
	exec := NewExecutor(WithExecutorHooks(Hooks{
		OnWorkflowFail: func(_ context.Context, _ string, _ error) {
			failed = true
		},
	}))

	handle, _ := exec.Execute(context.Background(), func(ctx WorkflowContext, _ any) (any, error) {
		return nil, fmt.Errorf("oops")
	}, WorkflowOptions{ID: "wf-hooks-fail"})

	handle.Result(context.Background())

	if !failed {
		t.Error("expected OnWorkflowFail to be called")
	}
}

func TestExecutor_GeneratedID(t *testing.T) {
	exec := NewExecutor()

	handle, err := exec.Execute(context.Background(), func(ctx WorkflowContext, _ any) (any, error) {
		return "ok", nil
	}, WorkflowOptions{})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if handle.ID() == "" {
		t.Error("expected generated ID")
	}
	handle.Result(context.Background())
}

func TestExecutor_WorkflowTimeout(t *testing.T) {
	exec := NewExecutor()

	handle, err := exec.Execute(context.Background(), func(ctx WorkflowContext, _ any) (any, error) {
		ctx.Sleep(10 * time.Second)
		return "done", nil
	}, WorkflowOptions{ID: "wf-timeout", Timeout: 50 * time.Millisecond})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	_, err = handle.Result(context.Background())
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestExecutor_ResultTimeout(t *testing.T) {
	exec := NewExecutor()

	handle, _ := exec.Execute(context.Background(), func(ctx WorkflowContext, _ any) (any, error) {
		ctx.Sleep(10 * time.Second)
		return "done", nil
	}, WorkflowOptions{ID: "wf-result-timeout"})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := handle.Result(ctx)
	if err == nil {
		t.Fatal("expected context deadline error")
	}
}
