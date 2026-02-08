package hitl

import (
	"context"
	"fmt"
	"testing"
)

func TestComposeHooks_OnRequest(t *testing.T) {
	var calls []string

	h1 := Hooks{
		OnRequest: func(_ context.Context, _ InteractionRequest) error {
			calls = append(calls, "h1")
			return nil
		},
	}
	h2 := Hooks{
		OnRequest: func(_ context.Context, _ InteractionRequest) error {
			calls = append(calls, "h2")
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.OnRequest(context.Background(), InteractionRequest{})
	if err != nil {
		t.Fatalf("OnRequest: %v", err)
	}
	if len(calls) != 2 || calls[0] != "h1" || calls[1] != "h2" {
		t.Errorf("expected [h1 h2], got %v", calls)
	}
}

func TestComposeHooks_OnRequest_ShortCircuit(t *testing.T) {
	var calls []string

	h1 := Hooks{
		OnRequest: func(_ context.Context, _ InteractionRequest) error {
			calls = append(calls, "h1")
			return fmt.Errorf("rejected")
		},
	}
	h2 := Hooks{
		OnRequest: func(_ context.Context, _ InteractionRequest) error {
			calls = append(calls, "h2")
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.OnRequest(context.Background(), InteractionRequest{})
	if err == nil {
		t.Fatal("expected error from short-circuit")
	}
	if len(calls) != 1 {
		t.Errorf("expected only h1 called, got %v", calls)
	}
}

func TestComposeHooks_OnApprove(t *testing.T) {
	var calls []string

	h1 := Hooks{
		OnApprove: func(_ context.Context, _ InteractionRequest, _ InteractionResponse) {
			calls = append(calls, "h1")
		},
	}
	h2 := Hooks{
		OnApprove: func(_ context.Context, _ InteractionRequest, _ InteractionResponse) {
			calls = append(calls, "h2")
		},
	}

	composed := ComposeHooks(h1, h2)
	composed.OnApprove(context.Background(), InteractionRequest{}, InteractionResponse{})
	if len(calls) != 2 {
		t.Errorf("expected 2 calls, got %d", len(calls))
	}
}

func TestComposeHooks_OnReject(t *testing.T) {
	called := false
	h := Hooks{
		OnReject: func(_ context.Context, _ InteractionRequest, _ InteractionResponse) {
			called = true
		},
	}

	composed := ComposeHooks(h)
	composed.OnReject(context.Background(), InteractionRequest{}, InteractionResponse{})
	if !called {
		t.Error("expected OnReject to be called")
	}
}

func TestComposeHooks_OnTimeout(t *testing.T) {
	called := false
	h := Hooks{
		OnTimeout: func(_ context.Context, _ InteractionRequest) {
			called = true
		},
	}

	composed := ComposeHooks(h)
	composed.OnTimeout(context.Background(), InteractionRequest{})
	if !called {
		t.Error("expected OnTimeout to be called")
	}
}

func TestComposeHooks_OnError(t *testing.T) {
	h := Hooks{
		OnError: func(_ context.Context, err error) error {
			return err
		},
	}

	composed := ComposeHooks(h)
	err := composed.OnError(context.Background(), fmt.Errorf("test error"))
	if err == nil {
		t.Error("expected error to be returned")
	}
}

func TestComposeHooks_OnError_ShortCircuit(t *testing.T) {
	var calls []string

	h1 := Hooks{
		OnError: func(_ context.Context, _ error) error {
			calls = append(calls, "h1")
			return fmt.Errorf("h1 error")
		},
	}
	h2 := Hooks{
		OnError: func(_ context.Context, _ error) error {
			calls = append(calls, "h2")
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.OnError(context.Background(), fmt.Errorf("original"))
	if err == nil || err.Error() != "h1 error" {
		t.Errorf("expected 'h1 error', got %v", err)
	}
	if len(calls) != 1 {
		t.Errorf("expected only h1 called, got %v", calls)
	}
}

func TestComposeHooks_OnError_Passthrough(t *testing.T) {
	h := Hooks{
		OnError: func(_ context.Context, _ error) error {
			return nil // suppress
		},
	}

	composed := ComposeHooks(h)
	original := fmt.Errorf("original error")
	err := composed.OnError(context.Background(), original)
	// When all hooks return nil, the original error is returned.
	if err != original {
		t.Errorf("expected original error passthrough, got %v", err)
	}
}

func TestComposeHooks_NilHooks(t *testing.T) {
	h1 := Hooks{} // All nil
	h2 := Hooks{
		OnRequest: func(_ context.Context, _ InteractionRequest) error {
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	err := composed.OnRequest(context.Background(), InteractionRequest{})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}
