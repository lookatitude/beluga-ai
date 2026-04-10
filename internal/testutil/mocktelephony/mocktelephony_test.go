package mocktelephony

import (
	"context"
	"testing"
)

func TestInMemoryEndpoint(t *testing.T) {
	ctx := context.Background()
	ep := NewInMemoryEndpoint()

	status, err := ep.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Connected {
		t.Error("expected disconnected initially")
	}
	if status.ActiveCalls != 0 {
		t.Errorf("ActiveCalls = %d, want 0", status.ActiveCalls)
	}

	if err := ep.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	status, _ = ep.Status(ctx)
	if !status.Connected {
		t.Error("expected connected after Connect")
	}

	ep.BeginCall()
	ep.BeginCall()
	status, _ = ep.Status(ctx)
	if status.ActiveCalls != 2 {
		t.Errorf("ActiveCalls = %d, want 2", status.ActiveCalls)
	}

	ep.EndCall()
	status, _ = ep.Status(ctx)
	if status.ActiveCalls != 1 {
		t.Errorf("ActiveCalls after EndCall = %d, want 1", status.ActiveCalls)
	}

	// EndCall clamps at zero.
	ep.EndCall()
	ep.EndCall()
	status, _ = ep.Status(ctx)
	if status.ActiveCalls != 0 {
		t.Errorf("ActiveCalls clamped = %d, want 0", status.ActiveCalls)
	}

	if err := ep.Disconnect(ctx); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	status, _ = ep.Status(ctx)
	if status.Connected {
		t.Error("expected disconnected after Disconnect")
	}
}
