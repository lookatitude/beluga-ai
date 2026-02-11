package voice

import (
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	s := NewSession("test-123")
	if s.ID != "test-123" {
		t.Errorf("ID = %q, want %q", s.ID, "test-123")
	}
	if s.State != StateIdle {
		t.Errorf("State = %q, want %q", s.State, StateIdle)
	}
	if s.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if s.Metadata == nil {
		t.Error("Metadata should be initialized")
	}
	if len(s.Turns) != 0 {
		t.Errorf("Turns = %d, want 0", len(s.Turns))
	}
}

func TestSessionTransitions(t *testing.T) {
	tests := []struct {
		name    string
		from    SessionState
		to      SessionState
		wantErr bool
	}{
		{"idle to listening", StateIdle, StateListening, false},
		{"listening to speaking", StateListening, StateSpeaking, false},
		{"speaking to listening", StateSpeaking, StateListening, false},
		{"speaking to idle", StateSpeaking, StateIdle, false},
		{"listening to idle", StateListening, StateIdle, false},
		{"idle to idle", StateIdle, StateIdle, false},
		{"idle to speaking", StateIdle, StateSpeaking, true},
		{"listening to listening", StateListening, StateListening, true},
		{"speaking to speaking", StateSpeaking, StateSpeaking, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &VoiceSession{State: tt.from}
			err := s.Transition(tt.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("Transition(%q→%q) error = %v, wantErr %v", tt.from, tt.to, err, tt.wantErr)
			}
			if err == nil && s.State != tt.to {
				t.Errorf("State = %q, want %q", s.State, tt.to)
			}
		})
	}
}

func TestSessionCurrentState(t *testing.T) {
	s := NewSession("test")
	if s.CurrentState() != StateIdle {
		t.Errorf("CurrentState() = %q, want %q", s.CurrentState(), StateIdle)
	}
	_ = s.Transition(StateListening)
	if s.CurrentState() != StateListening {
		t.Errorf("CurrentState() = %q, want %q", s.CurrentState(), StateListening)
	}
}

func TestSessionAddTurn(t *testing.T) {
	s := NewSession("test")

	turn := Turn{
		ID:        "turn-1",
		UserText:  "hello",
		AgentText: "hi there",
		StartTime: time.Now(),
		EndTime:   time.Now(),
	}
	s.AddTurn(turn)

	if s.TurnCount() != 1 {
		t.Errorf("TurnCount() = %d, want 1", s.TurnCount())
	}

	last := s.LastTurn()
	if last == nil {
		t.Fatal("LastTurn() returned nil")
	}
	if last.ID != "turn-1" {
		t.Errorf("LastTurn().ID = %q, want %q", last.ID, "turn-1")
	}
	if last.UserText != "hello" {
		t.Errorf("LastTurn().UserText = %q, want %q", last.UserText, "hello")
	}
}

func TestSessionLastTurnEmpty(t *testing.T) {
	s := NewSession("test")
	if s.LastTurn() != nil {
		t.Error("LastTurn() should return nil for empty session")
	}
}

func TestSessionTurnWithToolCalls(t *testing.T) {
	s := NewSession("test")
	turn := Turn{
		ID:        "turn-1",
		ToolCalls: []string{"call-1", "call-2"},
		StartTime: time.Now(),
	}
	s.AddTurn(turn)

	last := s.LastTurn()
	if len(last.ToolCalls) != 2 {
		t.Errorf("ToolCalls = %d, want 2", len(last.ToolCalls))
	}
}

func TestValidTransitionUnknownFromState(t *testing.T) {
	// An unknown "from" state should reject all transitions (except to idle).
	unknownState := SessionState("unknown")

	// unknown → listening should fail.
	s := &VoiceSession{State: unknownState}
	err := s.Transition(StateListening)
	if err == nil {
		t.Error("expected error for unknown→listening transition")
	}

	// unknown → speaking should fail.
	s2 := &VoiceSession{State: unknownState}
	err2 := s2.Transition(StateSpeaking)
	if err2 == nil {
		t.Error("expected error for unknown→speaking transition")
	}

	// unknown → idle should succeed (any→idle is always allowed).
	s3 := &VoiceSession{State: unknownState}
	err3 := s3.Transition(StateIdle)
	if err3 != nil {
		t.Errorf("unknown→idle should succeed, got error: %v", err3)
	}
}

func TestValidTransitionSameState(t *testing.T) {
	// speaking → speaking should be invalid.
	s := &VoiceSession{State: StateSpeaking}
	err := s.Transition(StateSpeaking)
	if err == nil {
		t.Error("expected error for speaking→speaking")
	}

	// listening → listening should be invalid.
	s2 := &VoiceSession{State: StateListening}
	err2 := s2.Transition(StateListening)
	if err2 == nil {
		t.Error("expected error for listening→listening")
	}
}
