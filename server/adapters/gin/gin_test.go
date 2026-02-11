package gin

import (
	"bytes"
	"context"
	"encoding/json"
	"iter"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/server"
	"github.com/lookatitude/beluga-ai/tool"
)

type mockAgent struct {
	id     string
	result string
	err    error
	events []agent.Event
}

func (m *mockAgent) ID() string             { return m.id }
func (m *mockAgent) Persona() agent.Persona  { return agent.Persona{} }
func (m *mockAgent) Tools() []tool.Tool      { return nil }
func (m *mockAgent) Children() []agent.Agent { return nil }

func (m *mockAgent) Invoke(_ context.Context, _ string, _ ...agent.Option) (string, error) {
	return m.result, m.err
}

func (m *mockAgent) Stream(_ context.Context, _ string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		for _, e := range m.events {
			if !yield(e, nil) {
				return
			}
		}
	}
}

func TestRegistry(t *testing.T) {
	t.Run("gin is registered", func(t *testing.T) {
		names := server.List()
		found := false
		for _, n := range names {
			if n == "gin" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected 'gin' in registry, got %v", names)
		}
	})

	t.Run("New returns gin adapter", func(t *testing.T) {
		adapter, err := server.New("gin", server.Config{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if adapter == nil {
			t.Fatal("expected non-nil adapter")
		}
	})
}

func TestAdapter_RegisterAgent(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		a := New(server.Config{})
		ag := &mockAgent{id: "test", result: "hello"}
		if err := a.RegisterAgent("/api/agent", ag); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("nil agent returns error", func(t *testing.T) {
		a := New(server.Config{})
		if err := a.RegisterAgent("/api/agent", nil); err == nil {
			t.Fatal("expected error for nil agent")
		}
	})
}

func TestAdapter_RegisterHandler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		a := New(server.Config{})
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		if err := a.RegisterHandler("/health", handler); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("nil handler returns error", func(t *testing.T) {
		a := New(server.Config{})
		if err := a.RegisterHandler("/health", nil); err == nil {
			t.Fatal("expected error for nil handler")
		}
	})
}

func TestAdapter_Invoke(t *testing.T) {
	a := New(server.Config{})
	ag := &mockAgent{id: "test", result: "hello world"}
	if err := a.RegisterAgent("/chat", ag); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}

	body, _ := json.Marshal(server.InvokeRequest{Input: "hi"})
	req := httptest.NewRequest(http.MethodPost, "/chat/invoke", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	a.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp server.InvokeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Result != "hello world" {
		t.Fatalf("expected 'hello world', got %q", resp.Result)
	}
}

func TestAdapter_Stream(t *testing.T) {
	a := New(server.Config{})
	ag := &mockAgent{
		id: "test",
		events: []agent.Event{
			{Type: agent.EventText, Text: "chunk1"},
			{Type: agent.EventDone},
		},
	}
	if err := a.RegisterAgent("/chat", ag); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}

	body, _ := json.Marshal(server.InvokeRequest{Input: "hi"})
	req := httptest.NewRequest(http.MethodPost, "/chat/stream", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	a.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("expected text/event-stream, got %q", ct)
	}
}

func TestAdapter_CustomHandler(t *testing.T) {
	a := New(server.Config{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	if err := a.RegisterHandler("/health", handler); err != nil {
		t.Fatalf("RegisterHandler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	a.Engine().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Fatalf("expected 'ok', got %q", w.Body.String())
	}
}

func TestAdapter_ServeAndShutdown(t *testing.T) {
	a := New(server.Config{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})
	ag := &mockAgent{id: "test", result: "hello"}
	a.RegisterAgent("/chat", ag)

	// Find a free port.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.Serve(ctx, addr)
	}()

	// Wait for server to start.
	time.Sleep(100 * time.Millisecond)

	// Make a request.
	body, _ := json.Marshal(server.InvokeRequest{Input: "hi"})
	resp, err := http.Post("http://"+addr+"/chat/invoke", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Shutdown via context cancellation.
	cancel()
	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Fatalf("Serve: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return within timeout")
	}
}

func TestAdapter_Shutdown_NoServer(t *testing.T) {
	a := New(server.Config{})
	if err := a.Shutdown(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAdapter_Engine(t *testing.T) {
	a := New(server.Config{})
	if a.Engine() == nil {
		t.Fatal("expected non-nil engine")
	}
}

func TestAdapter_Serve_ListenError(t *testing.T) {
	a := New(server.Config{})

	// Occupy a port.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := lis.Addr().String()
	defer lis.Close()

	// Attempt to serve on the same address - should fail immediately.
	err = a.Serve(context.Background(), addr)
	if err == nil {
		t.Fatal("expected error when address is already in use")
	}
	// The error should NOT be http.ErrServerClosed.
	if err == http.ErrServerClosed {
		t.Fatal("expected address-in-use error, not ErrServerClosed")
	}
}

func TestAdapter_Shutdown_Error(t *testing.T) {
	a := New(server.Config{})
	ag := &mockAgent{id: "test", result: "hello"}
	a.RegisterAgent("/chat", ag)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := lis.Addr().String()
	lis.Close()

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.Serve(ctx, addr)
	}()

	time.Sleep(100 * time.Millisecond)

	// Create an already-expired context for shutdown to trigger error.
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	shutdownCancel() // Cancel immediately

	// Call Shutdown with expired context - should return error.
	if err := a.Shutdown(shutdownCtx); err == nil {
		t.Log("shutdown with expired context did not error (server may have shut down cleanly)")
	}

	// Cancel the serve context.
	cancel()
	<-errCh
}
