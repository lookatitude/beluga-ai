package orchestration

import (
	"context"
	"errors"
	"testing"
)

func TestComposeHooks_BeforeStep(t *testing.T) {
	var calls []string
	h1 := Hooks{
		BeforeStep: func(_ context.Context, name string, _ any) error {
			calls = append(calls, "h1:"+name)
			return nil
		},
	}
	h2 := Hooks{
		BeforeStep: func(_ context.Context, name string, _ any) error {
			calls = append(calls, "h2:"+name)
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.BeforeStep(context.Background(), "step1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 2 || calls[0] != "h1:step1" || calls[1] != "h2:step1" {
		t.Fatalf("expected [h1:step1 h2:step1], got %v", calls)
	}
}

func TestComposeHooks_BeforeStep_ShortCircuit(t *testing.T) {
	errAbort := errors.New("abort")
	var calls []string
	h1 := Hooks{
		BeforeStep: func(_ context.Context, _ string, _ any) error {
			calls = append(calls, "h1")
			return errAbort
		},
	}
	h2 := Hooks{
		BeforeStep: func(_ context.Context, _ string, _ any) error {
			calls = append(calls, "h2")
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.BeforeStep(context.Background(), "s", nil)
	if !errors.Is(err, errAbort) {
		t.Fatalf("expected abort error, got %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("expected h1 only (short-circuit), got %v", calls)
	}
}

func TestComposeHooks_AfterStep(t *testing.T) {
	var calls []string
	h1 := Hooks{
		AfterStep: func(_ context.Context, name string, _ any, _ error) {
			calls = append(calls, "h1:"+name)
		},
	}
	h2 := Hooks{
		AfterStep: func(_ context.Context, name string, _ any, _ error) {
			calls = append(calls, "h2:"+name)
		},
	}

	composed := ComposeHooks(h1, h2)
	composed.AfterStep(context.Background(), "step1", nil, nil)
	if len(calls) != 2 || calls[0] != "h1:step1" || calls[1] != "h2:step1" {
		t.Fatalf("expected [h1:step1 h2:step1], got %v", calls)
	}
}

func TestComposeHooks_OnBranch(t *testing.T) {
	var calls []string
	h1 := Hooks{
		OnBranch: func(_ context.Context, from, to string) error {
			calls = append(calls, from+"->"+to)
			return nil
		},
	}

	composed := ComposeHooks(h1)
	err := composed.OnBranch(context.Background(), "a", "b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(calls) != 1 || calls[0] != "a->b" {
		t.Fatalf("expected [a->b], got %v", calls)
	}
}

func TestComposeHooks_OnBranch_ShortCircuit(t *testing.T) {
	errBranch := errors.New("branch error")
	h1 := Hooks{
		OnBranch: func(_ context.Context, _, _ string) error { return errBranch },
	}
	h2 := Hooks{
		OnBranch: func(_ context.Context, _, _ string) error { return nil },
	}

	composed := ComposeHooks(h1, h2)
	err := composed.OnBranch(context.Background(), "a", "b")
	if !errors.Is(err, errBranch) {
		t.Fatalf("expected branch error, got %v", err)
	}
}

func TestComposeHooks_OnError(t *testing.T) {
	original := errors.New("original")
	h1 := Hooks{
		OnError: func(_ context.Context, err error) error {
			return nil // Suppress.
		},
	}

	composed := ComposeHooks(h1)
	err := composed.OnError(context.Background(), original)
	// nil returned from h1 does NOT short-circuit; falls through to return original.
	// Wait — per the pattern, non-nil return short-circuits. nil continues.
	// After all hooks, if none returned non-nil, the original error is returned.
	// But h1 returned nil, so we continue. No more hooks. Return original.
	if err != original {
		t.Fatalf("expected original error (nil return continues), got %v", err)
	}
}

func TestComposeHooks_OnError_ShortCircuit(t *testing.T) {
	original := errors.New("original")
	replacement := errors.New("replaced")

	h1 := Hooks{
		OnError: func(_ context.Context, _ error) error { return replacement },
	}
	h2 := Hooks{
		OnError: func(_ context.Context, _ error) error {
			panic("should not reach h2")
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.OnError(context.Background(), original)
	if !errors.Is(err, replacement) {
		t.Fatalf("expected replacement error, got %v", err)
	}
}

func TestComposeHooks_NilHooksSkipped(t *testing.T) {
	// Hooks with nil callbacks should be silently skipped.
	h1 := Hooks{} // All nil.
	h2 := Hooks{
		BeforeStep: func(_ context.Context, _ string, _ any) error { return nil },
		AfterStep:  func(_ context.Context, _ string, _ any, _ error) {},
		OnBranch:   func(_ context.Context, _, _ string) error { return nil },
		OnError:    func(_ context.Context, err error) error { return err },
	}

	composed := ComposeHooks(h1, h2)
	// Should not panic.
	_ = composed.BeforeStep(context.Background(), "s", nil)
	composed.AfterStep(context.Background(), "s", nil, nil)
	_ = composed.OnBranch(context.Background(), "a", "b")
	_ = composed.OnError(context.Background(), errors.New("x"))
}
