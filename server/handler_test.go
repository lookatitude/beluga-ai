package server

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

func TestHandleInvoke(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		a := &mockAgent{id: "test", result: "Hello, world!"}
		handler := NewAgentHandler(a)

		body := `{"input":"hi"}`
		req := httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp InvokeResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Result != "Hello, world!" {
			t.Errorf("result = %q, want %q", resp.Result, "Hello, world!")
		}
		if resp.Error != "" {
			t.Errorf("unexpected error in response: %q", resp.Error)
		}
	})

	t.Run("agent error", func(t *testing.T) {
		a := &mockAgent{id: "test", err: errors.New("agent failed")}
		handler := NewAgentHandler(a)

		body := `{"input":"hi"}`
		req := httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}

		var resp InvokeResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.Error == "" {
			t.Error("expected non-empty error in response")
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		a := &mockAgent{id: "test", result: "ok"}
		handler := NewAgentHandler(a)

		req := httptest.NewRequest(http.MethodPost, "/invoke", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		a := &mockAgent{id: "test", result: "ok"}
		handler := NewAgentHandler(a)

		req := httptest.NewRequest(http.MethodGet, "/invoke", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			t.Fatal("expected non-200 status for GET request")
		}
	})
}

func TestHandleStream(t *testing.T) {
	t.Run("success with events", func(t *testing.T) {
		a := &mockAgent{
			id: "test",
			events: []agent.Event{
				{Type: agent.EventText, Text: "Hello", AgentID: "test"},
				{Type: agent.EventText, Text: " World", AgentID: "test"},
				{Type: agent.EventDone, AgentID: "test"},
			},
		}
		handler := NewAgentHandler(a)

		body := `{"input":"hi"}`
		req := httptest.NewRequest(http.MethodPost, "/stream", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}

		if got := w.Header().Get("Content-Type"); got != "text/event-stream" {
			t.Errorf("Content-Type = %q, want %q", got, "text/event-stream")
		}

		respBody := w.Body.String()
		if !strings.Contains(respBody, "event: text") {
			t.Errorf("expected 'event: text' in response body, got:\n%s", respBody)
		}
		if !strings.Contains(respBody, `"Hello"`) {
			t.Errorf("expected 'Hello' in response body, got:\n%s", respBody)
		}
		if !strings.Contains(respBody, "event: done") {
			t.Errorf("expected 'event: done' in response body, got:\n%s", respBody)
		}
	})

	t.Run("stream error", func(t *testing.T) {
		a := &errorStreamAgent{
			id:  "test",
			err: errors.New("stream failed"),
		}
		handler := NewAgentHandler(a)

		body := `{"input":"hi"}`
		req := httptest.NewRequest(http.MethodPost, "/stream", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		respBody := w.Body.String()
		if !strings.Contains(respBody, "event: error") {
			t.Errorf("expected 'event: error' in response body, got:\n%s", respBody)
		}
		if !strings.Contains(respBody, "stream failed") {
			t.Errorf("expected 'stream failed' in response body, got:\n%s", respBody)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		a := &mockAgent{id: "test"}
		handler := NewAgentHandler(a)

		req := httptest.NewRequest(http.MethodPost, "/stream", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("empty events stream sends done", func(t *testing.T) {
		a := &mockAgent{id: "test", events: nil}
		handler := NewAgentHandler(a)

		body := `{"input":"hi"}`
		req := httptest.NewRequest(http.MethodPost, "/stream", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		respBody := w.Body.String()
		if !strings.Contains(respBody, "event: done") {
			t.Errorf("expected final 'event: done' in response body, got:\n%s", respBody)
		}
	})

	t.Run("event with empty type defaults to message", func(t *testing.T) {
		a := &mockAgent{
			id: "test",
			events: []agent.Event{
				{Type: "", Text: "some text", AgentID: "test"},
			},
		}
		handler := NewAgentHandler(a)

		body := `{"input":"hi"}`
		req := httptest.NewRequest(http.MethodPost, "/stream", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		respBody := w.Body.String()
		// Should have "event: message" due to default.
		if !strings.Contains(respBody, "event: message") {
			t.Errorf("expected 'event: message' for empty event type, got:\n%s", respBody)
		}
		if !strings.Contains(respBody, "some text") {
			t.Errorf("expected 'some text' in response body, got:\n%s", respBody)
		}
	})
}

// errorStreamAgent emits a single error from Stream.
type errorStreamAgent struct {
	id  string
	err error
}

func (a *errorStreamAgent) ID() string               { return a.id }
func (a *errorStreamAgent) Persona() agent.Persona    { return agent.Persona{} }
func (a *errorStreamAgent) Tools() []tool.Tool        { return nil }
func (a *errorStreamAgent) Children() []agent.Agent   { return nil }

func (a *errorStreamAgent) Invoke(_ context.Context, _ string, _ ...agent.Option) (string, error) {
	return "", a.err
}

func (a *errorStreamAgent) Stream(_ context.Context, _ string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		yield(agent.Event{}, a.err)
	}
}
