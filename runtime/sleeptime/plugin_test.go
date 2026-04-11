package sleeptime

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/runtime"
	"github.com/lookatitude/beluga-ai/schema"
)

func TestSleeptimePlugin_Name(t *testing.T) {
	s := NewScheduler(&alwaysIdleDetector{})
	p := NewSleeptimePlugin(s)

	if got := p.Name(); got != "sleeptime" {
		t.Errorf("Name() = %q, want %q", got, "sleeptime")
	}
}

func TestSleeptimePlugin_BeforeTurn_Wakes(t *testing.T) {
	det := NewInactivityDetector(1 * time.Second)
	// Make detector idle by backdating lastSeen.
	det.lastSeen = time.Now().Add(-5 * time.Second)

	s := NewScheduler(det)
	p := NewSleeptimePlugin(s)

	if !det.IsIdle() {
		t.Fatal("expected detector to be idle before BeforeTurn")
	}

	session := runtime.NewSession("sess-1", "agent-1")
	msg := schema.NewHumanMessage("hello")

	got, err := p.BeforeTurn(context.Background(), session, msg)
	if err != nil {
		t.Fatalf("BeforeTurn() error = %v", err)
	}

	// Message should pass through unmodified.
	if got.GetRole() != msg.GetRole() {
		t.Errorf("BeforeTurn() role = %v, want %v", got.GetRole(), msg.GetRole())
	}

	// Detector should no longer be idle.
	if det.IsIdle() {
		t.Error("expected detector to not be idle after BeforeTurn")
	}
}

func TestSleeptimePlugin_AfterTurn_UpdatesState(t *testing.T) {
	s := NewScheduler(&alwaysIdleDetector{})
	p := NewSleeptimePlugin(s)

	session := runtime.NewSession("sess-1", "agent-1")
	session.State["key"] = "value"
	session.Turns = make([]schema.Turn, 5)

	events := []agent.Event{{Type: agent.EventText, Text: "hi"}}

	got, err := p.AfterTurn(context.Background(), session, events)
	if err != nil {
		t.Fatalf("AfterTurn() error = %v", err)
	}

	// Events should pass through unmodified.
	if len(got) != 1 || got[0].Text != "hi" {
		t.Errorf("AfterTurn() events = %v, want original", got)
	}

	// Check scheduler state was updated.
	s.mu.Lock()
	state := s.state
	s.mu.Unlock()

	if state.SessionID != "sess-1" {
		t.Errorf("state.SessionID = %q, want %q", state.SessionID, "sess-1")
	}
	if state.AgentID != "agent-1" {
		t.Errorf("state.AgentID = %q, want %q", state.AgentID, "agent-1")
	}
	if state.TurnCount != 5 {
		t.Errorf("state.TurnCount = %d, want %d", state.TurnCount, 5)
	}
	if state.Metadata["key"] != "value" {
		t.Errorf("state.Metadata[key] = %v, want %q", state.Metadata["key"], "value")
	}
}

func TestSleeptimePlugin_OnError_Passthrough(t *testing.T) {
	s := NewScheduler(&alwaysIdleDetector{})
	p := NewSleeptimePlugin(s)

	testErr := context.DeadlineExceeded
	got := p.OnError(context.Background(), testErr)
	if got != testErr {
		t.Errorf("OnError() = %v, want %v", got, testErr)
	}
}

func TestSleeptimePlugin_AfterTurn_NilState(t *testing.T) {
	s := NewScheduler(&alwaysIdleDetector{})
	p := NewSleeptimePlugin(s)

	session := runtime.NewSession("sess-2", "agent-2")
	session.State = nil

	_, err := p.AfterTurn(context.Background(), session, nil)
	if err != nil {
		t.Fatalf("AfterTurn() error = %v", err)
	}

	s.mu.Lock()
	state := s.state
	s.mu.Unlock()

	if state.Metadata != nil {
		t.Errorf("state.Metadata = %v, want nil", state.Metadata)
	}
}

// Verify compile-time interface check.
var _ runtime.Plugin = (*SleeptimePlugin)(nil)
