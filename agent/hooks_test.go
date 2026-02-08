package agent

import (
	"context"
	"errors"
	"testing"
)

func TestComposeHooks_OnStart_AllCalled(t *testing.T) {
	var calls []string

	h1 := Hooks{
		OnStart: func(ctx context.Context, input string) error {
			calls = append(calls, "h1:"+input)
			return nil
		},
	}
	h2 := Hooks{
		OnStart: func(ctx context.Context, input string) error {
			calls = append(calls, "h2:"+input)
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.OnStart(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	if calls[0] != "h1:test" || calls[1] != "h2:test" {
		t.Errorf("unexpected calls: %v", calls)
	}
}

func TestComposeHooks_OnStart_ShortCircuitsOnError(t *testing.T) {
	h1 := Hooks{
		OnStart: func(ctx context.Context, input string) error {
			return errors.New("blocked")
		},
	}
	h2Calls := 0
	h2 := Hooks{
		OnStart: func(ctx context.Context, input string) error {
			h2Calls++
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.OnStart(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error")
	}
	if h2Calls != 0 {
		t.Errorf("h2 should not have been called, got %d calls", h2Calls)
	}
}

func TestComposeHooks_OnEnd_AllCalled(t *testing.T) {
	var calls []string

	h1 := Hooks{
		OnEnd: func(ctx context.Context, result string, err error) {
			calls = append(calls, "h1:"+result)
		},
	}
	h2 := Hooks{
		OnEnd: func(ctx context.Context, result string, err error) {
			calls = append(calls, "h2:"+result)
		},
	}

	composed := ComposeHooks(h1, h2)
	composed.OnEnd(context.Background(), "done", nil)
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
}

func TestComposeHooks_OnError_ShortCircuitsOnError(t *testing.T) {
	h1 := Hooks{
		OnError: func(ctx context.Context, err error) error {
			return errors.New("replaced")
		},
	}
	h2Calls := 0
	h2 := Hooks{
		OnError: func(ctx context.Context, err error) error {
			h2Calls++
			return err
		},
	}

	composed := ComposeHooks(h1, h2)
	result := composed.OnError(context.Background(), errors.New("original"))
	if result == nil || result.Error() != "replaced" {
		t.Errorf("expected 'replaced' error, got: %v", result)
	}
	if h2Calls != 0 {
		t.Errorf("h2 should not have been called")
	}
}

func TestComposeHooks_OnError_PassesThrough(t *testing.T) {
	// Both hooks return nil (suppress error); the composed hook returns the original.
	h1 := Hooks{
		OnError: func(ctx context.Context, err error) error {
			return nil
		},
	}
	h2 := Hooks{
		OnError: func(ctx context.Context, err error) error {
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	result := composed.OnError(context.Background(), errors.New("original"))
	// When first hook returns nil, it short-circuits with nil.
	// Actually looking at the code: if h.OnError returns nil, it's skipped... no.
	// The code: if e := h.OnError(ctx, err); e != nil { return e }
	// So if h1 returns nil, it does NOT short-circuit but continues.
	// After both return nil, it returns err (the original).
	if result == nil || result.Error() != "original" {
		t.Errorf("expected original error, got: %v", result)
	}
}

func TestComposeHooks_NilHooksSkipped(t *testing.T) {
	// Hooks with nil fields should not panic.
	h1 := Hooks{} // All nil.
	h2 := Hooks{
		OnStart: func(ctx context.Context, input string) error {
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.OnStart(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestComposeHooks_OnToolCall(t *testing.T) {
	var calls []string

	h1 := Hooks{
		OnToolCall: func(ctx context.Context, call ToolCallInfo) error {
			calls = append(calls, "h1:"+call.Name)
			return nil
		},
	}
	h2 := Hooks{
		OnToolCall: func(ctx context.Context, call ToolCallInfo) error {
			calls = append(calls, "h2:"+call.Name)
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.OnToolCall(context.Background(), ToolCallInfo{Name: "search"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	if calls[0] != "h1:search" || calls[1] != "h2:search" {
		t.Errorf("unexpected calls: %v", calls)
	}
}

func TestComposeHooks_OnHandoff(t *testing.T) {
	var handoffs []string

	h := Hooks{
		OnHandoff: func(ctx context.Context, from, to string) error {
			handoffs = append(handoffs, from+"->"+to)
			return nil
		},
	}

	composed := ComposeHooks(h)
	err := composed.OnHandoff(context.Background(), "agent-a", "agent-b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handoffs) != 1 || handoffs[0] != "agent-a->agent-b" {
		t.Errorf("unexpected handoffs: %v", handoffs)
	}
}

func TestComposeHooks_OnIteration(t *testing.T) {
	var iters []int

	h := Hooks{
		OnIteration: func(ctx context.Context, iteration int) error {
			iters = append(iters, iteration)
			return nil
		},
	}

	composed := ComposeHooks(h)
	_ = composed.OnIteration(context.Background(), 0)
	_ = composed.OnIteration(context.Background(), 1)

	if len(iters) != 2 || iters[0] != 0 || iters[1] != 1 {
		t.Errorf("unexpected iterations: %v", iters)
	}
}

func TestComposeHooks_BeforeAfterPlan(t *testing.T) {
	var calls []string

	h := Hooks{
		BeforePlan: func(ctx context.Context, state PlannerState) error {
			calls = append(calls, "before")
			return nil
		},
		AfterPlan: func(ctx context.Context, actions []Action) error {
			calls = append(calls, "after")
			return nil
		},
	}

	composed := ComposeHooks(h)
	_ = composed.BeforePlan(context.Background(), PlannerState{})
	_ = composed.AfterPlan(context.Background(), nil)

	if len(calls) != 2 || calls[0] != "before" || calls[1] != "after" {
		t.Errorf("unexpected calls: %v", calls)
	}
}

func TestComposeHooks_BeforeAfterAct(t *testing.T) {
	var calls []string

	h := Hooks{
		BeforeAct: func(ctx context.Context, action Action) error {
			calls = append(calls, "before:"+action.Message)
			return nil
		},
		AfterAct: func(ctx context.Context, action Action, obs Observation) error {
			calls = append(calls, "after:"+action.Message)
			return nil
		},
	}

	composed := ComposeHooks(h)
	action := Action{Type: ActionRespond, Message: "hello"}
	_ = composed.BeforeAct(context.Background(), action)
	_ = composed.AfterAct(context.Background(), action, Observation{})

	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d: %v", len(calls), calls)
	}
}
