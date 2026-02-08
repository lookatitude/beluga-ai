package fiber

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"iter"
	"net"
	"net/http"
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
	t.Run("fiber is registered", func(t *testing.T) {
		names := server.List()
		found := false
		for _, n := range names {
			if n == "fiber" {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected 'fiber' in registry, got %v", names)
		}
	})

	t.Run("New returns fiber adapter", func(t *testing.T) {
		adapter, err := server.New("fiber", server.Config{})
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
	req, _ := http.NewRequest(http.MethodPost, "http://localhost/chat/invoke", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.App().Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var invokeResp server.InvokeResponse
	if err := json.NewDecoder(resp.Body).Decode(&invokeResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if invokeResp.Result != "hello world" {
		t.Fatalf("expected 'hello world', got %q", invokeResp.Result)
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
	req, _ := http.NewRequest(http.MethodPost, "http://localhost/chat/stream", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.App().Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
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

	req, _ := http.NewRequest(http.MethodGet, "http://localhost/health", nil)
	resp, err := a.App().Test(req)
	if err != nil {
		t.Fatalf("Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	if string(bodyBytes) != "ok" {
		t.Fatalf("expected 'ok', got %q", string(bodyBytes))
	}
}

func TestAdapter_ServeAndShutdown(t *testing.T) {
	a := New(server.Config{
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})
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

	time.Sleep(200 * time.Millisecond)

	body, _ := json.Marshal(server.InvokeRequest{Input: "hi"})
	resp, err := http.Post("http://"+addr+"/chat/invoke", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

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
	// Fiber Shutdown before Listen may return error; that's acceptable.
	_ = a.Shutdown(context.Background())
}

func TestAdapter_App(t *testing.T) {
	a := New(server.Config{})
	if a.App() == nil {
		t.Fatal("expected non-nil app")
	}
}
