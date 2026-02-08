package workflow

import (
	"context"
	"testing"
)

func TestComposeHooks_OnWorkflowStart(t *testing.T) {
	var calls []string
	h1 := Hooks{OnWorkflowStart: func(_ context.Context, id string, _ any) {
		calls = append(calls, "h1:"+id)
	}}
	h2 := Hooks{OnWorkflowStart: func(_ context.Context, id string, _ any) {
		calls = append(calls, "h2:"+id)
	}}
	composed := ComposeHooks(h1, h2)
	composed.OnWorkflowStart(context.Background(), "wf-1", nil)

	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	if calls[0] != "h1:wf-1" {
		t.Errorf("expected h1:wf-1, got %s", calls[0])
	}
	if calls[1] != "h2:wf-1" {
		t.Errorf("expected h2:wf-1, got %s", calls[1])
	}
}

func TestComposeHooks_OnWorkflowComplete(t *testing.T) {
	var calls int
	h := ComposeHooks(
		Hooks{OnWorkflowComplete: func(_ context.Context, _ string, _ any) { calls++ }},
		Hooks{}, // nil hook, should be skipped
		Hooks{OnWorkflowComplete: func(_ context.Context, _ string, _ any) { calls++ }},
	)
	h.OnWorkflowComplete(context.Background(), "wf-1", "result")
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestComposeHooks_OnWorkflowFail(t *testing.T) {
	var called bool
	h := ComposeHooks(Hooks{OnWorkflowFail: func(_ context.Context, _ string, _ error) {
		called = true
	}})
	h.OnWorkflowFail(context.Background(), "wf-1", nil)
	if !called {
		t.Error("expected OnWorkflowFail to be called")
	}
}

func TestComposeHooks_OnActivityStart(t *testing.T) {
	var called bool
	h := ComposeHooks(Hooks{OnActivityStart: func(_ context.Context, _ string, _ any) {
		called = true
	}})
	h.OnActivityStart(context.Background(), "wf-1", "input")
	if !called {
		t.Error("expected OnActivityStart to be called")
	}
}

func TestComposeHooks_OnActivityComplete(t *testing.T) {
	var called bool
	h := ComposeHooks(Hooks{OnActivityComplete: func(_ context.Context, _ string, _ any) {
		called = true
	}})
	h.OnActivityComplete(context.Background(), "wf-1", "result")
	if !called {
		t.Error("expected OnActivityComplete to be called")
	}
}

func TestComposeHooks_OnSignal(t *testing.T) {
	var received Signal
	h := ComposeHooks(Hooks{OnSignal: func(_ context.Context, _ string, s Signal) {
		received = s
	}})
	h.OnSignal(context.Background(), "wf-1", Signal{Name: "approve", Payload: true})
	if received.Name != "approve" {
		t.Errorf("expected approve, got %s", received.Name)
	}
}

func TestComposeHooks_OnRetry(t *testing.T) {
	var called bool
	h := ComposeHooks(Hooks{OnRetry: func(_ context.Context, _ string, _ error) {
		called = true
	}})
	h.OnRetry(context.Background(), "wf-1", nil)
	if !called {
		t.Error("expected OnRetry to be called")
	}
}

func TestComposeHooks_Empty(t *testing.T) {
	h := ComposeHooks()
	// Should not panic with no hooks.
	h.OnWorkflowStart(context.Background(), "wf-1", nil)
	h.OnWorkflowComplete(context.Background(), "wf-1", nil)
	h.OnWorkflowFail(context.Background(), "wf-1", nil)
	h.OnActivityStart(context.Background(), "wf-1", nil)
	h.OnActivityComplete(context.Background(), "wf-1", nil)
	h.OnSignal(context.Background(), "wf-1", Signal{})
	h.OnRetry(context.Background(), "wf-1", nil)
}
