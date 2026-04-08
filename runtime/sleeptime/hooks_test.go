package sleeptime

import (
	"context"
	"errors"
	"testing"
)

func TestComposeHooks_OnIdle(t *testing.T) {
	var calls []string
	h1 := Hooks{OnIdle: func(_ context.Context, _ SessionState) { calls = append(calls, "h1") }}
	h2 := Hooks{OnIdle: func(_ context.Context, _ SessionState) { calls = append(calls, "h2") }}
	h3 := Hooks{} // nil OnIdle

	composed := ComposeHooks(h1, h2, h3)
	composed.OnIdle(context.Background(), SessionState{})

	if len(calls) != 2 || calls[0] != "h1" || calls[1] != "h2" {
		t.Errorf("OnIdle calls = %v, want [h1, h2]", calls)
	}
}

func TestComposeHooks_OnWake(t *testing.T) {
	var calls []string
	h1 := Hooks{OnWake: func(_ context.Context, _ SessionState) { calls = append(calls, "w1") }}
	h2 := Hooks{OnWake: func(_ context.Context, _ SessionState) { calls = append(calls, "w2") }}

	composed := ComposeHooks(h1, h2)
	composed.OnWake(context.Background(), SessionState{})

	if len(calls) != 2 || calls[0] != "w1" || calls[1] != "w2" {
		t.Errorf("OnWake calls = %v, want [w1, w2]", calls)
	}
}

func TestComposeHooks_BeforeTask_StopsOnError(t *testing.T) {
	errBlocked := errors.New("blocked")
	var calls []string

	h1 := Hooks{BeforeTask: func(_ context.Context, _ string, _ SessionState) error {
		calls = append(calls, "bt1")
		return nil
	}}
	h2 := Hooks{BeforeTask: func(_ context.Context, _ string, _ SessionState) error {
		calls = append(calls, "bt2")
		return errBlocked
	}}
	h3 := Hooks{BeforeTask: func(_ context.Context, _ string, _ SessionState) error {
		calls = append(calls, "bt3")
		return nil
	}}

	composed := ComposeHooks(h1, h2, h3)
	err := composed.BeforeTask(context.Background(), "test", SessionState{})

	if !errors.Is(err, errBlocked) {
		t.Errorf("BeforeTask error = %v, want %v", err, errBlocked)
	}
	if len(calls) != 2 || calls[0] != "bt1" || calls[1] != "bt2" {
		t.Errorf("BeforeTask calls = %v, want [bt1, bt2]", calls)
	}
}

func TestComposeHooks_AfterTask(t *testing.T) {
	var results []string
	h1 := Hooks{AfterTask: func(_ context.Context, r TaskResult) { results = append(results, r.TaskName+"-h1") }}
	h2 := Hooks{AfterTask: func(_ context.Context, r TaskResult) { results = append(results, r.TaskName+"-h2") }}

	composed := ComposeHooks(h1, h2)
	composed.AfterTask(context.Background(), TaskResult{TaskName: "test"})

	if len(results) != 2 || results[0] != "test-h1" || results[1] != "test-h2" {
		t.Errorf("AfterTask results = %v, want [test-h1, test-h2]", results)
	}
}

func TestComposeHooks_Empty(t *testing.T) {
	composed := ComposeHooks()

	// All hooks should be non-nil and safe to call.
	composed.OnIdle(context.Background(), SessionState{})
	composed.OnWake(context.Background(), SessionState{})
	if err := composed.BeforeTask(context.Background(), "x", SessionState{}); err != nil {
		t.Errorf("BeforeTask error = %v, want nil", err)
	}
	composed.AfterTask(context.Background(), TaskResult{})
}
