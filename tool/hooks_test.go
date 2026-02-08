package tool

import (
	"context"
	"errors"
	"testing"
)

func TestComposeHooks_BeforeExecute_RunsInOrder(t *testing.T) {
	var order []string

	h1 := Hooks{
		BeforeExecute: func(_ context.Context, name string, _ map[string]any) error {
			order = append(order, "h1:"+name)
			return nil
		},
	}
	h2 := Hooks{
		BeforeExecute: func(_ context.Context, name string, _ map[string]any) error {
			order = append(order, "h2:"+name)
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.BeforeExecute(context.Background(), "test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(order))
	}
	if order[0] != "h1:test" || order[1] != "h2:test" {
		t.Errorf("order = %v, want [h1:test, h2:test]", order)
	}
}

func TestComposeHooks_BeforeExecute_StopsOnError(t *testing.T) {
	sentinel := errors.New("blocked")
	var called []string

	h1 := Hooks{
		BeforeExecute: func(_ context.Context, _ string, _ map[string]any) error {
			called = append(called, "h1")
			return sentinel
		},
	}
	h2 := Hooks{
		BeforeExecute: func(_ context.Context, _ string, _ map[string]any) error {
			called = append(called, "h2")
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.BeforeExecute(context.Background(), "t", nil)
	if err != sentinel {
		t.Errorf("error = %v, want %v", err, sentinel)
	}
	if len(called) != 1 || called[0] != "h1" {
		t.Errorf("called = %v, want [h1] (h2 should be skipped)", called)
	}
}

func TestComposeHooks_BeforeExecute_SkipsNil(t *testing.T) {
	called := false
	h1 := Hooks{} // nil BeforeExecute
	h2 := Hooks{
		BeforeExecute: func(_ context.Context, _ string, _ map[string]any) error {
			called = true
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.BeforeExecute(context.Background(), "t", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("h2.BeforeExecute should have been called")
	}
}

func TestComposeHooks_AfterExecute_RunsAll(t *testing.T) {
	var order []string

	h1 := Hooks{
		AfterExecute: func(_ context.Context, name string, _ *Result, _ error) {
			order = append(order, "h1")
		},
	}
	h2 := Hooks{
		AfterExecute: func(_ context.Context, name string, _ *Result, _ error) {
			order = append(order, "h2")
		},
	}

	composed := ComposeHooks(h1, h2)
	composed.AfterExecute(context.Background(), "t", nil, nil)
	if len(order) != 2 || order[0] != "h1" || order[1] != "h2" {
		t.Errorf("order = %v, want [h1, h2]", order)
	}
}

func TestComposeHooks_AfterExecute_SkipsNil(t *testing.T) {
	called := false
	h1 := Hooks{} // nil AfterExecute
	h2 := Hooks{
		AfterExecute: func(_ context.Context, _ string, _ *Result, _ error) {
			called = true
		},
	}

	composed := ComposeHooks(h1, h2)
	composed.AfterExecute(context.Background(), "t", nil, nil)
	if !called {
		t.Error("h2.AfterExecute should have been called")
	}
}

func TestComposeHooks_AfterExecute_ReceivesResultAndError(t *testing.T) {
	var gotResult *Result
	var gotErr error

	h := Hooks{
		AfterExecute: func(_ context.Context, _ string, r *Result, err error) {
			gotResult = r
			gotErr = err
		},
	}

	composed := ComposeHooks(h)
	expectedResult := TextResult("hello")
	expectedErr := errors.New("oops")

	composed.AfterExecute(context.Background(), "t", expectedResult, expectedErr)
	if gotResult != expectedResult {
		t.Errorf("result = %v, want %v", gotResult, expectedResult)
	}
	if gotErr != expectedErr {
		t.Errorf("error = %v, want %v", gotErr, expectedErr)
	}
}

func TestComposeHooks_OnError_FirstNonNilWins(t *testing.T) {
	original := errors.New("original")
	replacement := errors.New("replaced")

	h1 := Hooks{
		OnError: func(_ context.Context, _ string, _ error) error {
			return replacement
		},
	}
	h2 := Hooks{
		OnError: func(_ context.Context, _ string, _ error) error {
			t.Error("h2 should not be called when h1 returns non-nil")
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.OnError(context.Background(), "t", original)
	if err != replacement {
		t.Errorf("error = %v, want %v", err, replacement)
	}
}

func TestComposeHooks_OnError_ReturnsOriginalIfAllNil(t *testing.T) {
	original := errors.New("original")

	h1 := Hooks{
		OnError: func(_ context.Context, _ string, _ error) error {
			return nil // suppress
		},
	}
	h2 := Hooks{
		OnError: func(_ context.Context, _ string, _ error) error {
			return nil // suppress
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.OnError(context.Background(), "t", original)
	if err != original {
		t.Errorf("error = %v, want %v (original when all return nil)", err, original)
	}
}

func TestComposeHooks_OnError_SkipsNil(t *testing.T) {
	original := errors.New("original")
	called := false

	h1 := Hooks{} // nil OnError
	h2 := Hooks{
		OnError: func(_ context.Context, _ string, err error) error {
			called = true
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	composed.OnError(context.Background(), "t", original)
	if !called {
		t.Error("h2.OnError should have been called")
	}
}

func TestComposeHooks_Empty(t *testing.T) {
	composed := ComposeHooks()

	// All hooks should be non-nil and not panic.
	if err := composed.BeforeExecute(context.Background(), "t", nil); err != nil {
		t.Errorf("BeforeExecute error = %v, want nil", err)
	}
	composed.AfterExecute(context.Background(), "t", nil, nil) // should not panic

	original := errors.New("err")
	if err := composed.OnError(context.Background(), "t", original); err != original {
		t.Errorf("OnError = %v, want %v", err, original)
	}
}

func TestWithHooks_BeforeExecuteAborts(t *testing.T) {
	sentinel := errors.New("abort")
	base := &mockTool{name: "test"}
	hooked := WithHooks(base, Hooks{
		BeforeExecute: func(_ context.Context, _ string, _ map[string]any) error {
			return sentinel
		},
	})

	_, err := hooked.Execute(context.Background(), nil)
	if err != sentinel {
		t.Errorf("error = %v, want %v", err, sentinel)
	}
}

func TestWithHooks_FullLifecycle(t *testing.T) {
	var order []string

	base := &mockTool{
		name: "test",
		executeFn: func(input map[string]any) (*Result, error) {
			order = append(order, "execute")
			return TextResult("ok"), nil
		},
	}

	hooked := WithHooks(base, Hooks{
		BeforeExecute: func(_ context.Context, _ string, _ map[string]any) error {
			order = append(order, "before")
			return nil
		},
		AfterExecute: func(_ context.Context, _ string, _ *Result, _ error) {
			order = append(order, "after")
		},
	})

	result, err := hooked.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	expected := []string{"before", "execute", "after"}
	if len(order) != len(expected) {
		t.Fatalf("order = %v, want %v", order, expected)
	}
	for i := range expected {
		if order[i] != expected[i] {
			t.Errorf("order[%d] = %q, want %q", i, order[i], expected[i])
		}
	}
}

func TestWithHooks_OnErrorCalled(t *testing.T) {
	execErr := errors.New("exec failed")
	replacement := errors.New("replaced")

	base := &mockTool{
		name: "fail",
		executeFn: func(input map[string]any) (*Result, error) {
			return nil, execErr
		},
	}

	var afterErr error
	hooked := WithHooks(base, Hooks{
		OnError: func(_ context.Context, _ string, err error) error {
			return replacement
		},
		AfterExecute: func(_ context.Context, _ string, _ *Result, err error) {
			afterErr = err
		},
	})

	_, err := hooked.Execute(context.Background(), nil)
	if err != replacement {
		t.Errorf("error = %v, want %v", err, replacement)
	}
	if afterErr != replacement {
		t.Errorf("AfterExecute received error = %v, want %v", afterErr, replacement)
	}
}

func TestWithHooks_OnErrorSuppresses(t *testing.T) {
	execErr := errors.New("exec failed")

	base := &mockTool{
		name: "fail",
		executeFn: func(input map[string]any) (*Result, error) {
			return nil, execErr
		},
	}

	hooked := WithHooks(base, Hooks{
		OnError: func(_ context.Context, _ string, _ error) error {
			return nil // suppress the error
		},
	})

	_, err := hooked.Execute(context.Background(), nil)
	if err != nil {
		t.Errorf("error = %v, want nil (suppressed)", err)
	}
}

func TestWithHooks_PreservesToolMetadata(t *testing.T) {
	base := &mockTool{
		name:        "mytool",
		description: "A test tool",
		inputSchema: map[string]any{"type": "object"},
	}

	hooked := WithHooks(base, Hooks{})
	if hooked.Name() != "mytool" {
		t.Errorf("Name() = %q, want %q", hooked.Name(), "mytool")
	}
	if hooked.Description() != "A test tool" {
		t.Errorf("Description() = %q, want %q", hooked.Description(), "A test tool")
	}
	if hooked.InputSchema() == nil {
		t.Error("InputSchema() should not be nil")
	}
}

func TestWithHooks_NilHooksNoOp(t *testing.T) {
	base := &mockTool{
		name: "test",
		executeFn: func(input map[string]any) (*Result, error) {
			return TextResult("ok"), nil
		},
	}

	// All hooks nil - should execute without panicking.
	hooked := WithHooks(base, Hooks{})
	result, err := hooked.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestWithHooks_OnErrorNotCalledOnSuccess(t *testing.T) {
	onErrorCalled := false
	base := &mockTool{
		name: "ok",
		executeFn: func(input map[string]any) (*Result, error) {
			return TextResult("ok"), nil
		},
	}

	hooked := WithHooks(base, Hooks{
		OnError: func(_ context.Context, _ string, _ error) error {
			onErrorCalled = true
			return nil
		},
	})

	_, err := hooked.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if onErrorCalled {
		t.Error("OnError should not be called on success")
	}
}
