package rest

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	id        string
	invokeFn  func(ctx context.Context, input string) (string, error)
	streamFn  func(ctx context.Context, input string) iter.Seq2[agent.Event, error]
}

func (m *mockAgent) ID() string               { return m.id }
func (m *mockAgent) Persona() agent.Persona    { return agent.Persona{Role: m.id} }
func (m *mockAgent) Tools() []tool.Tool        { return nil }
func (m *mockAgent) Children() []agent.Agent   { return nil }

func (m *mockAgent) Invoke(ctx context.Context, input string, _ ...agent.Option) (string, error) {
	if m.invokeFn != nil {
		return m.invokeFn(ctx, input)
	}
	return "result: " + input, nil
}

func (m *mockAgent) Stream(ctx context.Context, input string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	if m.streamFn != nil {
		return m.streamFn(ctx, input)
	}
	return func(yield func(agent.Event, error) bool) {
		if !yield(agent.Event{Type: agent.EventText, Text: "chunk1", AgentID: m.id}, nil) {
			return
		}
		if !yield(agent.Event{Type: agent.EventText, Text: "chunk2", AgentID: m.id}, nil) {
			return
		}
		yield(agent.Event{Type: agent.EventDone, AgentID: m.id}, nil)
	}
}

func setupRESTTestServer() (*RESTServer, *httptest.Server) {
	srv := NewServer()
	srv.RegisterAgent("assistant", &mockAgent{id: "assistant"})
	ts := httptest.NewServer(srv.Handler())
	return srv, ts
}

func TestRESTServer_RegisterAgent(t *testing.T) {
	srv := NewServer()
	if err := srv.RegisterAgent("test", &mockAgent{id: "test"}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}

	// Duplicate registration should fail.
	if err := srv.RegisterAgent("test", &mockAgent{id: "test2"}); err == nil {
		t.Fatal("expected error for duplicate registration")
	}

	// Empty path should fail.
	if err := srv.RegisterAgent("", &mockAgent{id: "test3"}); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestRESTServer_Invoke(t *testing.T) {
	_, ts := setupRESTTestServer()
	defer ts.Close()

	body := `{"input":"hello"}`
	resp, err := http.Post(ts.URL+"/assistant/invoke", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var result InvokeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Result != "result: hello" {
		t.Errorf("expected 'result: hello', got %q", result.Result)
	}
}

func TestRESTServer_Invoke_EmptyInput(t *testing.T) {
	_, ts := setupRESTTestServer()
	defer ts.Close()

	body := `{"input":""}`
	resp, err := http.Post(ts.URL+"/assistant/invoke", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRESTServer_Invoke_InvalidBody(t *testing.T) {
	_, ts := setupRESTTestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/assistant/invoke", "application/json", strings.NewReader("not json"))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRESTServer_Invoke_AgentNotFound(t *testing.T) {
	_, ts := setupRESTTestServer()
	defer ts.Close()

	body := `{"input":"hello"}`
	resp, err := http.Post(ts.URL+"/nonexistent/invoke", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestRESTServer_Invoke_Error(t *testing.T) {
	srv := NewServer()
	srv.RegisterAgent("fail", &mockAgent{
		id: "fail",
		invokeFn: func(_ context.Context, _ string) (string, error) {
			return "", fmt.Errorf("agent error")
		},
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"input":"hello"}`
	resp, err := http.Post(ts.URL+"/fail/invoke", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestRESTServer_Stream(t *testing.T) {
	_, ts := setupRESTTestServer()
	defer ts.Close()

	body := `{"input":"hello"}`
	resp, err := http.Post(ts.URL+"/assistant/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("expected Content-Type 'text/event-stream', got %q", ct)
	}

	// Parse SSE events.
	events := parseSSEEvents(t, resp)

	// We expect: text, text, done (from agent), done (from server).
	if len(events) < 3 {
		t.Fatalf("expected at least 3 events, got %d", len(events))
	}
}

func TestRESTServer_Stream_EmptyInput(t *testing.T) {
	_, ts := setupRESTTestServer()
	defer ts.Close()

	body := `{"input":""}`
	resp, err := http.Post(ts.URL+"/assistant/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRESTServer_MethodNotAllowed(t *testing.T) {
	_, ts := setupRESTTestServer()
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/assistant/invoke")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", resp.StatusCode)
	}
}

func TestRESTServer_InvalidPath(t *testing.T) {
	_, ts := setupRESTTestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/noslash", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestRESTServer_UnknownAction(t *testing.T) {
	_, ts := setupRESTTestServer()
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/assistant/unknown", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestSSEWriter_WriteEvent(t *testing.T) {
	w := httptest.NewRecorder()
	sw, err := NewSSEWriter(w)
	if err != nil {
		t.Fatalf("NewSSEWriter: %v", err)
	}

	if err := sw.WriteEvent(SSEEvent{Event: "message", Data: "hello", ID: "1"}); err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	output := w.Body.String()
	if !strings.Contains(output, "id: 1") {
		t.Error("expected id line")
	}
	if !strings.Contains(output, "event: message") {
		t.Error("expected event line")
	}
	if !strings.Contains(output, "data: hello") {
		t.Error("expected data line")
	}
}

func TestSSEWriter_WriteEvent_DataOnly(t *testing.T) {
	w := httptest.NewRecorder()
	sw, err := NewSSEWriter(w)
	if err != nil {
		t.Fatalf("NewSSEWriter: %v", err)
	}

	if err := sw.WriteEvent(SSEEvent{Data: "just data"}); err != nil {
		t.Fatalf("WriteEvent: %v", err)
	}

	output := w.Body.String()
	if strings.Contains(output, "id:") {
		t.Error("expected no id line")
	}
	if strings.Contains(output, "event:") {
		t.Error("expected no event line")
	}
	if !strings.Contains(output, "data: just data") {
		t.Error("expected data line")
	}
}

func TestSSEWriter_WriteHeartbeat(t *testing.T) {
	w := httptest.NewRecorder()
	sw, err := NewSSEWriter(w)
	if err != nil {
		t.Fatalf("NewSSEWriter: %v", err)
	}

	if err := sw.WriteHeartbeat(); err != nil {
		t.Fatalf("WriteHeartbeat: %v", err)
	}

	output := w.Body.String()
	if !strings.Contains(output, ": heartbeat") {
		t.Error("expected heartbeat comment")
	}
}

func TestSSEWriter_Headers(t *testing.T) {
	w := httptest.NewRecorder()
	_, err := NewSSEWriter(w)
	if err != nil {
		t.Fatalf("NewSSEWriter: %v", err)
	}

	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected Content-Type 'text/event-stream', got %q", ct)
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected Cache-Control 'no-cache', got %q", cc)
	}
	if conn := w.Header().Get("Connection"); conn != "keep-alive" {
		t.Errorf("expected Connection 'keep-alive', got %q", conn)
	}
}

func TestRESTServer_StreamError(t *testing.T) {
	srv := NewServer()
	srv.RegisterAgent("err", &mockAgent{
		id: "err",
		streamFn: func(_ context.Context, _ string) iter.Seq2[agent.Event, error] {
			return func(yield func(agent.Event, error) bool) {
				yield(agent.Event{}, fmt.Errorf("stream error"))
			}
		},
	})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"input":"hello"}`
	resp, err := http.Post(ts.URL+"/err/stream", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	events := parseSSEEvents(t, resp)
	hasError := false
	for _, e := range events {
		if e.Event == "error" {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected at least one error event")
	}
}

func TestRESTServer_RegisterAgent_TrimSlashes(t *testing.T) {
	srv := NewServer()
	if err := srv.RegisterAgent("/trimmed/", &mockAgent{id: "t"}); err != nil {
		t.Fatalf("RegisterAgent: %v", err)
	}
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body := `{"input":"hello"}`
	resp, err := http.Post(ts.URL+"/trimmed/invoke", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// parseSSEEvents reads SSE events from the response body.
func parseSSEEvents(t *testing.T, resp *http.Response) []sseTestEvent {
	t.Helper()
	var events []sseTestEvent
	scanner := bufio.NewScanner(resp.Body)
	var current sseTestEvent
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if current.Data != "" || current.Event != "" {
				events = append(events, current)
				current = sseTestEvent{}
			}
			continue
		}
		if strings.HasPrefix(line, "event: ") {
			current.Event = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			current.Data = strings.TrimPrefix(line, "data: ")
		} else if strings.HasPrefix(line, "id: ") {
			current.ID = strings.TrimPrefix(line, "id: ")
		}
	}
	return events
}

type sseTestEvent struct {
	Event string
	Data  string
	ID    string
}

