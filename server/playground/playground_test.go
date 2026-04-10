package playground

import (
	"context"
	"encoding/json"
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
	id string
}

func (m *mockAgent) ID() string              { return m.id }
func (m *mockAgent) Persona() agent.Persona  { return agent.Persona{Role: "test"} }
func (m *mockAgent) Tools() []tool.Tool      { return nil }
func (m *mockAgent) Children() []agent.Agent { return nil }
func (m *mockAgent) Invoke(_ context.Context, input string, _ ...agent.Option) (string, error) {
	return "response to: " + input, nil
}
func (m *mockAgent) Stream(_ context.Context, input string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		yield(agent.Event{Type: agent.EventText, Text: "hello from " + m.id}, nil)
	}
}

var _ agent.Agent = (*mockAgent)(nil)

func TestStaticSelector(t *testing.T) {
	a1 := &mockAgent{id: "agent-1"}
	a2 := &mockAgent{id: "agent-2"}
	sel := NewStaticSelector(a1, a2)

	ctx := context.Background()

	ids := sel.List(ctx)
	if len(ids) != 2 {
		t.Errorf("List: got %d agents, want 2", len(ids))
	}
	// Should be sorted.
	if ids[0] != "agent-1" || ids[1] != "agent-2" {
		t.Errorf("List: got %v, want [agent-1 agent-2]", ids)
	}

	a := sel.Get(ctx, "agent-1")
	if a == nil || a.ID() != "agent-1" {
		t.Error("Get(agent-1): expected agent-1")
	}

	a = sel.Get(ctx, "nonexistent")
	if a != nil {
		t.Error("Get(nonexistent): expected nil")
	}
}

func TestHandleUI(t *testing.T) {
	sel := NewStaticSelector(&mockAgent{id: "test"})
	h := NewHandler(sel, WithTitle("Test Playground"))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	if !strings.Contains(w.Body.String(), "Test Playground") {
		t.Error("HTML should contain the title")
	}
}

func TestHandleListAgents(t *testing.T) {
	sel := NewStaticSelector(&mockAgent{id: "a1"}, &mockAgent{id: "a2"})
	h := NewHandler(sel)

	req := httptest.NewRequest("GET", "/agents", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var resp map[string][]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp["agents"]) != 2 {
		t.Errorf("agents = %v, want 2 entries", resp["agents"])
	}
}

func TestHandleChat_MissingFields(t *testing.T) {
	sel := NewStaticSelector(&mockAgent{id: "a1"})
	h := NewHandler(sel)

	req := httptest.NewRequest("POST", "/chat", strings.NewReader(`{"agent_id":"","input":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestHandleChat_AgentNotFound(t *testing.T) {
	sel := NewStaticSelector(&mockAgent{id: "a1"})
	h := NewHandler(sel)

	req := httptest.NewRequest("POST", "/chat", strings.NewReader(`{"agent_id":"nonexistent","input":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestNewHandler_Options(t *testing.T) {
	sel := NewStaticSelector()
	h := NewHandler(sel, WithTitle("Custom"), WithBasePath("/custom"))
	if h.opts.title != "Custom" {
		t.Errorf("title = %q, want Custom", h.opts.title)
	}
	if h.opts.path != "/custom" {
		t.Errorf("path = %q, want /custom", h.opts.path)
	}
}
