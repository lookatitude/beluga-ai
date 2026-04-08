package telephony

import (
	"context"
	"testing"
	"time"
)

func TestPrefixRouter(t *testing.T) {
	defaultAction := CallAction{Type: "reject", Reason: "no route"}
	router := NewPrefixRouter(defaultAction)

	router.AddRule("+1", CallAction{Type: "accept", AgentID: "us-agent"})
	router.AddRule("+44", CallAction{Type: "accept", AgentID: "uk-agent"})
	router.AddRule("+1415", CallAction{Type: "accept", AgentID: "sf-agent"})

	tests := []struct {
		name     string
		to       string
		wantType string
		wantID   string
	}{
		{"US number", "+15551234567", "accept", "us-agent"},
		{"UK number", "+447911123456", "accept", "uk-agent"},
		{"SF number (longest match)", "+14155551234", "accept", "sf-agent"},
		{"no match", "+33612345678", "reject", ""},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := router.Route(ctx, IncomingCall{To: tt.to})
			if err != nil {
				t.Fatalf("Route: %v", err)
			}
			if action.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", action.Type, tt.wantType)
			}
			if action.AgentID != tt.wantID {
				t.Errorf("AgentID = %q, want %q", action.AgentID, tt.wantID)
			}
		})
	}
}

func TestInMemoryEndpoint(t *testing.T) {
	ctx := context.Background()
	ep := NewInMemoryEndpoint()

	// Initially disconnected.
	status, err := ep.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Connected {
		t.Error("expected disconnected initially")
	}

	// Connect.
	if err := ep.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	status, err = ep.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !status.Connected {
		t.Error("expected connected after Connect")
	}

	// Disconnect.
	if err := ep.Disconnect(ctx); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}

	status, err = ep.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Connected {
		t.Error("expected disconnected after Disconnect")
	}
}

func TestCallTypes(t *testing.T) {
	call := Call{
		ID:        "call-1",
		Status:    "in_progress",
		From:      "+15551234567",
		To:        "+15559876543",
		StartTime: time.Now(),
	}

	if call.ID != "call-1" {
		t.Errorf("Call.ID = %q, want call-1", call.ID)
	}

	req := CallRequest{
		To:      "+15559876543",
		From:    "+15551234567",
		AgentID: "test-agent",
		Timeout: 30 * time.Second,
	}

	if req.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", req.Timeout)
	}
}

func TestRegistry(t *testing.T) {
	// Register a test provider.
	Register("test", func(cfg Config) (TelephonyProvider, error) {
		return nil, nil
	})

	names := List()
	found := false
	for _, n := range names {
		if n == "test" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'test' in registry")
	}

	_, err := New("nonexistent", Config{})
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestOptions(t *testing.T) {
	var opts providerOptions
	WithMaxConcurrentCalls(10)(&opts)
	WithDefaultTimeout(30 * time.Second)(&opts)

	if opts.maxConcurrentCalls != 10 {
		t.Errorf("maxConcurrentCalls = %d, want 10", opts.maxConcurrentCalls)
	}
	if opts.defaultTimeout != 30*time.Second {
		t.Errorf("defaultTimeout = %v, want 30s", opts.defaultTimeout)
	}
}
