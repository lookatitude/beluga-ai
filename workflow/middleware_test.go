package workflow

import (
	"context"
	"fmt"
	"testing"
)

func TestApplyMiddleware_Order(t *testing.T) {
	var order []string
	mw := func(label string) Middleware {
		return func(next DurableExecutor) DurableExecutor {
			return &trackingExecutor{next: next, label: label, order: &order}
		}
	}

	exec := NewExecutor()
	wrapped := ApplyMiddleware(exec, mw("outer"), mw("inner"))

	handle, err := wrapped.Execute(context.Background(), func(ctx WorkflowContext, _ any) (any, error) {
		return "done", nil
	}, WorkflowOptions{ID: "mw-order"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	handle.Result(context.Background())

	if len(order) != 2 {
		t.Fatalf("expected 2 middleware calls, got %d", len(order))
	}
	if order[0] != "outer" {
		t.Errorf("expected outer first, got %s", order[0])
	}
	if order[1] != "inner" {
		t.Errorf("expected inner second, got %s", order[1])
	}
}

type trackingExecutor struct {
	next  DurableExecutor
	label string
	order *[]string
}

func (e *trackingExecutor) Execute(ctx context.Context, fn WorkflowFunc, opts WorkflowOptions) (WorkflowHandle, error) {
	*e.order = append(*e.order, e.label)
	return e.next.Execute(ctx, fn, opts)
}

func (e *trackingExecutor) Signal(ctx context.Context, wfID string, signal Signal) error {
	return e.next.Signal(ctx, wfID, signal)
}

func (e *trackingExecutor) Query(ctx context.Context, wfID string, queryType string) (any, error) {
	return e.next.Query(ctx, wfID, queryType)
}

func (e *trackingExecutor) Cancel(ctx context.Context, wfID string) error {
	return e.next.Cancel(ctx, wfID)
}

func TestWithHooks_Signal(t *testing.T) {
	var signalReceived string
	mw := WithHooks(Hooks{
		OnSignal: func(_ context.Context, _ string, s Signal) {
			signalReceived = s.Name
		},
	})

	exec := NewExecutor()
	wrapped := mw(exec)

	handle, _ := wrapped.Execute(context.Background(), func(ctx WorkflowContext, _ any) (any, error) {
		ch := ctx.ReceiveSignal("go")
		select {
		case <-ch:
			return "got-signal", nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}, WorkflowOptions{ID: "mw-signal"})

	_ = handle
	// The executor stores the running workflow by ID.
	// Signal goes through the wrapped executor, triggering the hook.
	wrapped.Signal(context.Background(), "mw-signal", Signal{Name: "go"})

	if signalReceived != "go" {
		t.Errorf("expected signal 'go', got %q", signalReceived)
	}
}

func TestWithHooks_Execute(t *testing.T) {
	var started bool
	mw := WithHooks(Hooks{
		OnWorkflowStart: func(_ context.Context, _ string, _ any) {
			started = true
		},
	})

	exec := NewExecutor()
	wrapped := mw(exec)

	handle, err := wrapped.Execute(context.Background(), func(ctx WorkflowContext, _ any) (any, error) {
		return "ok", nil
	}, WorkflowOptions{ID: "mw-exec"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	handle.Result(context.Background())

	if !started {
		t.Error("expected OnWorkflowStart hook to fire")
	}
}

func TestApplyMiddleware_Empty(t *testing.T) {
	exec := NewExecutor()
	wrapped := ApplyMiddleware(exec) // no middlewares

	handle, err := wrapped.Execute(context.Background(), func(_ WorkflowContext, _ any) (any, error) {
		return "ok", nil
	}, WorkflowOptions{ID: "mw-empty"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	result, err := handle.Result(context.Background())
	if err != nil {
		t.Fatalf("Result: %v", err)
	}
	if result != "ok" {
		t.Errorf("expected 'ok', got %v", result)
	}
}

func TestHookedExecutor_Cancel(t *testing.T) {
	mw := WithHooks(Hooks{})
	exec := NewExecutor()
	wrapped := mw(exec)

	err := wrapped.Cancel(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent workflow")
	}
}

func TestHookedExecutor_Query(t *testing.T) {
	mw := WithHooks(Hooks{})
	exec := NewExecutor()
	wrapped := mw(exec)

	_, err := wrapped.Query(context.Background(), "nonexistent", "status")
	if err == nil {
		t.Fatal("expected error for nonexistent workflow")
	}
}

func TestHookedExecutor_CompileTimeCheck(t *testing.T) {
	// Just ensure it satisfies the interface.
	var _ DurableExecutor = &hookedExecutor{}
	_ = fmt.Sprintf("ok")
}
