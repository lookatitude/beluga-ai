package server

import (
	"context"
	"iter"
	"net/http"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	id     string
	result string
	err    error
	events []agent.Event
}

func (m *mockAgent) ID() string               { return m.id }
func (m *mockAgent) Persona() agent.Persona    { return agent.Persona{} }
func (m *mockAgent) Tools() []tool.Tool        { return nil }
func (m *mockAgent) Children() []agent.Agent   { return nil }

func (m *mockAgent) Invoke(ctx context.Context, input string, opts ...agent.Option) (string, error) {
	return m.result, m.err
}

func (m *mockAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		for _, e := range m.events {
			if !yield(e, nil) {
				return
			}
		}
	}
}

func TestRegistry(t *testing.T) {
	t.Run("stdlib is registered by default", func(t *testing.T) {
		names := List()
		found := false
		for _, n := range names {
			if n == "stdlib" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected 'stdlib' in registry, got %v", names)
		}
	})

	t.Run("New returns stdlib adapter", func(t *testing.T) {
		adapter, err := New("stdlib", Config{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if adapter == nil {
			t.Fatal("expected non-nil adapter")
		}
	})

	t.Run("New returns error for unknown adapter", func(t *testing.T) {
		_, err := New("unknown", Config{})
		if err == nil {
			t.Fatal("expected error for unknown adapter")
		}
	})

	t.Run("Register and New custom adapter", func(t *testing.T) {
		Register("test-adapter", func(cfg Config) (ServerAdapter, error) {
			return NewStdlibAdapter(cfg), nil
		})
		adapter, err := New("test-adapter", Config{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if adapter == nil {
			t.Fatal("expected non-nil adapter")
		}
	})

	t.Run("List returns sorted names", func(t *testing.T) {
		names := List()
		for i := 1; i < len(names); i++ {
			if names[i-1] > names[i] {
				t.Fatalf("names not sorted: %v", names)
			}
		}
	})
}

func TestStdlibAdapter_RegisterAgent(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		adapter := NewStdlibAdapter(Config{})
		a := &mockAgent{id: "test-agent", result: "hello"}
		if err := adapter.RegisterAgent("/api/agent", a); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("nil agent returns error", func(t *testing.T) {
		adapter := NewStdlibAdapter(Config{})
		if err := adapter.RegisterAgent("/api/agent", nil); err == nil {
			t.Fatal("expected error for nil agent")
		}
	})
}

func TestStdlibAdapter_RegisterHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		adapter := NewStdlibAdapter(Config{})
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		if err := adapter.RegisterHandler("/health", handler); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("nil handler returns error", func(t *testing.T) {
		adapter := NewStdlibAdapter(Config{})
		if err := adapter.RegisterHandler("/health", nil); err == nil {
			t.Fatal("expected error for nil handler")
		}
	})
}

func TestStdlibAdapter_Shutdown_NoServer(t *testing.T) {
	adapter := NewStdlibAdapter(Config{})
	if err := adapter.Shutdown(context.Background()); err != nil {
		t.Fatalf("unexpected error shutting down unstarted adapter: %v", err)
	}
}
