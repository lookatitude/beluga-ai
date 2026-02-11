package server

import (
	"context"
	"iter"
	"net"
	"net/http"
	"testing"
	"time"

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

func TestStdlibAdapter_ServeAndShutdown(t *testing.T) {
	adapter := NewStdlibAdapter(Config{})

	// Register a simple health handler instead of agent (to avoid routing issues).
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	if err := adapter.RegisterHandler("/health", handler); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}

	// Get a random port.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get random port: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	// Start server in goroutine.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- adapter.Serve(ctx, addr)
	}()

	// Wait for server to start (give it some time to bind).
	time.Sleep(100 * time.Millisecond)

	// Make a request to verify server is running.
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + addr + "/health")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Cancel context to trigger shutdown.
	cancel()

	// Wait for Serve to return (should get context.Canceled).
	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Fatalf("expected context.Canceled, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shutdown in time")
	}
}

func TestStdlibAdapter_Serve_ListenError(t *testing.T) {
	adapter := NewStdlibAdapter(Config{})
	// Use an invalid address to trigger listen error.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := adapter.Serve(ctx, "invalid-address")
	if err == nil {
		t.Fatal("expected error for invalid address")
	}
	if err == context.Canceled || err == context.DeadlineExceeded {
		t.Fatal("should have gotten listen error, not context error")
	}
}

func TestStdlibAdapter_Shutdown_WithRunningServer(t *testing.T) {
	adapter := NewStdlibAdapter(Config{})

	// Get a random port.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get random port: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	// Start server in background.
	ctx := context.Background()
	go adapter.Serve(ctx, addr)

	// Wait for server to start.
	time.Sleep(50 * time.Millisecond)

	// Shutdown the server explicitly.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := adapter.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown error: %v", err)
	}
}
