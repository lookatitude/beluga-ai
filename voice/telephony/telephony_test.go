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

// stubProvider is a minimal TelephonyProvider used to observe Config values
// passed into factories during registry tests.
type stubProvider struct {
	cfg Config
}

func (s *stubProvider) PlaceCall(_ context.Context, _ CallRequest) (*Call, error) {
	return &Call{ID: "stub"}, nil
}
func (s *stubProvider) HangUp(_ context.Context, _ string) error { return nil }

// registerForTest inserts a factory into the registry without going through
// Register, so tests can manipulate the registry after FreezeRegistry may have
// been called elsewhere in the test run.
func registerForTest(name string, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = f
}

func TestRegistry(t *testing.T) {
	registerForTest("test-registry", func(cfg Config) (TelephonyProvider, error) {
		return &stubProvider{cfg: cfg}, nil
	})

	names := List()
	found := false
	for _, n := range names {
		if n == "test-registry" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'test-registry' in registry")
	}

	_, err := New("nonexistent", Config{})
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestNewAppliesOptions(t *testing.T) {
	var captured Config
	registerForTest("test-opts", func(cfg Config) (TelephonyProvider, error) {
		captured = cfg
		return &stubProvider{cfg: cfg}, nil
	})

	_, err := New("test-opts", Config{MaxConcurrentCalls: 1, DefaultTimeout: time.Second},
		WithMaxConcurrentCalls(10),
		WithDefaultTimeout(45*time.Second),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if captured.MaxConcurrentCalls != 10 {
		t.Errorf("MaxConcurrentCalls = %d, want 10", captured.MaxConcurrentCalls)
	}
	if captured.DefaultTimeout != 45*time.Second {
		t.Errorf("DefaultTimeout = %v, want 45s", captured.DefaultTimeout)
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

func TestRegisterValidation(t *testing.T) {
	// Empty name panics.
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for empty name")
			}
		}()
		Register("", func(Config) (TelephonyProvider, error) { return nil, nil })
	}()

	// Nil factory panics.
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for nil factory")
			}
		}()
		Register("nil-factory", nil)
	}()
}

func TestFreezeRegistryBlocksRegister(t *testing.T) {
	// Save and restore global state so this test doesn't affect others.
	prev := registryHot.Load()
	defer registryHot.Store(prev)

	FreezeRegistry()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic after FreezeRegistry")
		}
	}()
	Register("after-freeze", func(Config) (TelephonyProvider, error) { return nil, nil })
}
