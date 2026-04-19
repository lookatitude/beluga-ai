package agui

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/tool"
)

type mockAgent struct {
	id        string
	role      string
	goal      string
	tools     []tool.Tool
	invokeErr error
	streamErr error
}

func (m *mockAgent) ID() string { return m.id }
func (m *mockAgent) Persona() agent.Persona {
	return agent.Persona{Role: m.role, Goal: m.goal}
}
func (m *mockAgent) Tools() []tool.Tool      { return m.tools }
func (m *mockAgent) Children() []agent.Agent { return nil }
func (m *mockAgent) Invoke(_ context.Context, input string, _ ...agent.Option) (string, error) {
	if m.invokeErr != nil {
		return "", m.invokeErr
	}
	return "response: " + input, nil
}
func (m *mockAgent) Stream(_ context.Context, input string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		if m.streamErr != nil {
			yield(agent.Event{}, m.streamErr)
			return
		}
		if !yield(agent.Event{Type: agent.EventText, Text: "streamed: " + input}, nil) {
			return
		}
		yield(agent.Event{Type: agent.EventDone, Text: ""}, nil)
	}
}

var _ agent.Agent = (*mockAgent)(nil)

func TestHandleAgents(t *testing.T) {
	agents := []agent.Agent{
		&mockAgent{id: "a1", role: "Helper"},
		&mockAgent{id: "a2", role: "Analyst"},
	}
	h := NewHandler(agents)

	req := httptest.NewRequest("GET", "/agui/agents", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var entries []AgentEntry
	if err := json.NewDecoder(w.Body).Decode(&entries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("entries = %d, want 2", len(entries))
	}
}

func TestHandleManifest(t *testing.T) {
	agents := []agent.Agent{&mockAgent{id: "a1", role: "Test"}}
	h := NewHandler(agents, WithVersion("2.0"))

	req := httptest.NewRequest("GET", "/agui/manifest", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var manifest AgentsManifest
	if err := json.NewDecoder(w.Body).Decode(&manifest); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if manifest.Version != "2.0" {
		t.Errorf("version = %q, want 2.0", manifest.Version)
	}
}

func TestHandleChat_NotFound(t *testing.T) {
	h := NewHandler(nil)

	req := httptest.NewRequest("POST", "/agui/chat/nonexistent", strings.NewReader(`{"input":"hi"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestHandleChat_MissingAgentID(t *testing.T) {
	h := NewHandler(nil)

	req := httptest.NewRequest("POST", "/agui/chat/", strings.NewReader(`{"input":"hi"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 400 or 404", w.Code)
	}
}

func TestParseManifest(t *testing.T) {
	data := []byte(`{"version":"1.0","agents":[{"id":"a1","name":"Test"}]}`)
	m, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("ParseManifest: %v", err)
	}
	if m.Version != "1.0" {
		t.Errorf("version = %q, want 1.0", m.Version)
	}
	if len(m.Agents) != 1 {
		t.Errorf("agents = %d, want 1", len(m.Agents))
	}
}

func TestParseManifest_Invalid(t *testing.T) {
	_, err := ParseManifest([]byte(`{invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestGenerateManifest(t *testing.T) {
	agents := []agent.Agent{&mockAgent{id: "a1", role: "Helper"}}
	data, err := GenerateManifest(agents)
	if err != nil {
		t.Fatalf("GenerateManifest: %v", err)
	}

	m, err := ParseManifest(data)
	if err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if len(m.Agents) != 1 {
		t.Errorf("agents = %d, want 1", len(m.Agents))
	}
}

func TestGenerateMarkdown(t *testing.T) {
	agents := []agent.Agent{
		&mockAgent{id: "a1", role: "Helper"},
		&mockAgent{id: "a2", role: "Analyst"},
	}

	md := GenerateMarkdown(agents)
	if !strings.Contains(md, "# AGENTS.md") {
		t.Error("markdown should contain header")
	}
	if !strings.Contains(md, "## a1") {
		t.Error("markdown should contain agent a1")
	}
	if !strings.Contains(md, "**Role**: Helper") {
		t.Error("markdown should contain role")
	}
}

func TestStreamToUIEvents(t *testing.T) {
	stream := func(yield func(agent.Event, error) bool) {
		yield(agent.Event{Type: agent.EventText, Text: "hello"}, nil)
		yield(agent.Event{Type: agent.EventDone, Text: ""}, nil)
	}

	var events []UIEvent
	for evt, err := range StreamToUIEvents("test", stream) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		events = append(events, evt)
	}

	if len(events) != 2 {
		t.Errorf("events = %d, want 2", len(events))
	}
	if events[0].AgentID != "test" {
		t.Errorf("AgentID = %q, want test", events[0].AgentID)
	}
}

func TestHandleChat_Success(t *testing.T) {
	agents := []agent.Agent{&mockAgent{id: "a1", role: "Helper"}}
	h := NewHandler(agents)

	req := httptest.NewRequest("POST", "/agui/chat/a1", strings.NewReader(`{"input":"hi"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	// httptest.ResponseRecorder implements http.Flusher, so SSE path is used.
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "data: ") {
		t.Errorf("expected SSE data frame, got %q", body)
	}
	if !strings.Contains(body, "streamed: hi") {
		t.Errorf("expected streamed output, got %q", body)
	}
}

func TestHandleChat_StreamError(t *testing.T) {
	agents := []agent.Agent{&mockAgent{id: "a1", role: "Helper", streamErr: errors.New("boom\nsecret")}}
	h := NewHandler(agents)

	req := httptest.NewRequest("POST", "/agui/chat/a1", strings.NewReader(`{"input":"hi"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (SSE header already sent)", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"type":"error"`) {
		t.Errorf("expected error event, got %q", body)
	}
	if strings.Contains(body, "boom") {
		t.Errorf("error body should not leak provider error: %q", body)
	}
}

func TestHandleChat_InvalidBody(t *testing.T) {
	agents := []agent.Agent{&mockAgent{id: "a1", role: "Helper"}}
	h := NewHandler(agents)

	req := httptest.NewRequest("POST", "/agui/chat/a1", strings.NewReader(`{not-json`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleChat_WrongMethod(t *testing.T) {
	agents := []agent.Agent{&mockAgent{id: "a1", role: "Helper"}}
	h := NewHandler(agents)

	req := httptest.NewRequest("GET", "/agui/chat/a1", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestHandleChat_WrongMethod_NoOracle(t *testing.T) {
	// Both existing and non-existing agents should return 405 for wrong method,
	// so clients cannot distinguish registered vs unregistered agents.
	agents := []agent.Agent{&mockAgent{id: "real", role: "Helper"}}
	h := NewHandler(agents)

	for _, id := range []string{"real", "ghost"} {
		req := httptest.NewRequest("PUT", "/agui/chat/"+id, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("id=%s: status = %d, want 405", id, w.Code)
		}
	}
}

func TestHandleAgents_WrongMethod(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest("POST", "/agui/agents", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestHandleManifest_WrongMethod(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest("DELETE", "/agui/manifest", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestServeHTTP_NotFound(t *testing.T) {
	h := NewHandler(nil)
	req := httptest.NewRequest("GET", "/agui/unknown", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestGenerateMarkdown_WithGoal(t *testing.T) {
	agents := []agent.Agent{&mockAgent{id: "a1", role: "Helper", goal: "assist users"}}
	md := GenerateMarkdown(agents)
	if !strings.Contains(md, "**Goal**: assist users") {
		t.Errorf("markdown should contain goal: %s", md)
	}
}

func TestStreamToUIEvents_Error(t *testing.T) {
	boomErr := errors.New("stream boom")
	stream := func(yield func(agent.Event, error) bool) {
		yield(agent.Event{}, boomErr)
	}

	var gotErr error
	for _, err := range StreamToUIEvents("a1", stream) {
		if err != nil {
			gotErr = err
			break
		}
	}
	if gotErr == nil {
		t.Error("expected error from stream")
	}
}

func TestSanitizeLogValue(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"hello", "hello"},
		{"line1\nline2", "line1 line2"},
		{"tab\there", "tab here"},
		{"carriage\rreturn", "carriage return"},
		{"ctrl\x01char", "ctrl?char"},
		{strings.Repeat("x", 300), strings.Repeat("x", 256)},
	}
	for _, c := range cases {
		if got := sanitizeLogValue(c.in); got != c.want {
			t.Errorf("sanitizeLogValue(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestOptions(t *testing.T) {
	agents := []agent.Agent{&mockAgent{id: "a1", role: "Test"}}
	h := NewHandler(agents, WithBasePath("/custom"), WithVersion("3.0"))

	if h.opts.basePath != "/custom" {
		t.Errorf("basePath = %q, want /custom", h.opts.basePath)
	}
	if h.opts.version != "3.0" {
		t.Errorf("version = %q, want 3.0", h.opts.version)
	}
}
