package schema

import (
	"testing"
	"time"
)

func TestSession_Fields(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name       string
		session    Session
		wantID     string
		wantTurns  int
		wantState  bool
	}{
		{
			name: "fully_populated",
			session: Session{
				ID: "sess-1",
				Turns: []Turn{
					{
						Input:     NewHumanMessage("hello"),
						Output:    NewAIMessage("hi there"),
						Timestamp: now,
					},
				},
				State:     map[string]any{"step": 1},
				CreatedAt: now.Add(-time.Hour),
				UpdatedAt: now,
			},
			wantID:    "sess-1",
			wantTurns: 1,
			wantState: true,
		},
		{
			name: "empty_session",
			session: Session{
				ID:        "sess-2",
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantID:    "sess-2",
			wantTurns: 0,
			wantState: false,
		},
		{
			name: "multiple_turns",
			session: Session{
				ID: "sess-3",
				Turns: []Turn{
					{Input: NewHumanMessage("first"), Output: NewAIMessage("resp1"), Timestamp: now},
					{Input: NewHumanMessage("second"), Output: NewAIMessage("resp2"), Timestamp: now},
					{Input: NewHumanMessage("third"), Output: NewAIMessage("resp3"), Timestamp: now},
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
			wantID:    "sess-3",
			wantTurns: 3,
			wantState: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.session.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", tt.session.ID, tt.wantID)
			}
			if len(tt.session.Turns) != tt.wantTurns {
				t.Errorf("len(Turns) = %d, want %d", len(tt.session.Turns), tt.wantTurns)
			}
			hasState := tt.session.State != nil
			if hasState != tt.wantState {
				t.Errorf("has State = %v, want %v", hasState, tt.wantState)
			}
		})
	}
}

func TestSession_ZeroValue(t *testing.T) {
	var s Session
	if s.ID != "" {
		t.Errorf("zero ID = %q, want empty", s.ID)
	}
	if s.Turns != nil {
		t.Errorf("zero Turns = %v, want nil", s.Turns)
	}
	if s.State != nil {
		t.Errorf("zero State = %v, want nil", s.State)
	}
	if !s.CreatedAt.IsZero() {
		t.Errorf("zero CreatedAt = %v, want zero", s.CreatedAt)
	}
	if !s.UpdatedAt.IsZero() {
		t.Errorf("zero UpdatedAt = %v, want zero", s.UpdatedAt)
	}
}

func TestSession_Timestamps(t *testing.T) {
	created := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC)

	s := Session{
		ID:        "sess-time",
		CreatedAt: created,
		UpdatedAt: updated,
	}

	if !s.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt = %v, want %v", s.CreatedAt, created)
	}
	if !s.UpdatedAt.Equal(updated) {
		t.Errorf("UpdatedAt = %v, want %v", s.UpdatedAt, updated)
	}
	if !s.UpdatedAt.After(s.CreatedAt) {
		t.Error("UpdatedAt should be after CreatedAt")
	}
}

func TestSession_State(t *testing.T) {
	s := Session{
		ID: "sess-state",
		State: map[string]any{
			"step":        3,
			"agent":       "researcher",
			"accumulated": []string{"fact1", "fact2"},
		},
	}

	if v, ok := s.State["step"].(int); !ok || v != 3 {
		t.Errorf("State[\"step\"] = %v, want 3", s.State["step"])
	}
	if v, ok := s.State["agent"].(string); !ok || v != "researcher" {
		t.Errorf("State[\"agent\"] = %v, want %q", s.State["agent"], "researcher")
	}

	// State is mutable
	s.State["step"] = 4
	if s.State["step"] != 4 {
		t.Errorf("State[\"step\"] after mutation = %v, want 4", s.State["step"])
	}
}

func TestSession_TurnAccess(t *testing.T) {
	now := time.Now()
	s := Session{
		ID: "sess-turns",
		Turns: []Turn{
			{
				Input:     NewHumanMessage("What is Go?"),
				Output:    NewAIMessage("Go is a programming language."),
				Timestamp: now,
				Metadata:  map[string]any{"latency_ms": 150},
			},
		},
	}

	if len(s.Turns) != 1 {
		t.Fatalf("len(Turns) = %d, want 1", len(s.Turns))
	}

	turn := s.Turns[0]
	if turn.Input.GetRole() != RoleHuman {
		t.Errorf("Turn.Input.GetRole() = %q, want %q", turn.Input.GetRole(), RoleHuman)
	}
	if turn.Output.GetRole() != RoleAI {
		t.Errorf("Turn.Output.GetRole() = %q, want %q", turn.Output.GetRole(), RoleAI)
	}
}

func TestTurn_Fields(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		turn      Turn
		wantMeta  bool
	}{
		{
			name: "with_metadata",
			turn: Turn{
				Input:     NewHumanMessage("hi"),
				Output:    NewAIMessage("hello"),
				Timestamp: now,
				Metadata:  map[string]any{"tool_count": 2},
			},
			wantMeta: true,
		},
		{
			name: "without_metadata",
			turn: Turn{
				Input:     NewHumanMessage("bye"),
				Output:    NewAIMessage("goodbye"),
				Timestamp: now,
			},
			wantMeta: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.turn.Input == nil {
				t.Error("Input is nil")
			}
			if tt.turn.Output == nil {
				t.Error("Output is nil")
			}
			if tt.turn.Timestamp.IsZero() {
				t.Error("Timestamp is zero")
			}
			hasMeta := tt.turn.Metadata != nil
			if hasMeta != tt.wantMeta {
				t.Errorf("has Metadata = %v, want %v", hasMeta, tt.wantMeta)
			}
		})
	}
}

func TestTurn_ZeroValue(t *testing.T) {
	var turn Turn
	if turn.Input != nil {
		t.Errorf("zero Input = %v, want nil", turn.Input)
	}
	if turn.Output != nil {
		t.Errorf("zero Output = %v, want nil", turn.Output)
	}
	if !turn.Timestamp.IsZero() {
		t.Errorf("zero Timestamp = %v, want zero", turn.Timestamp)
	}
	if turn.Metadata != nil {
		t.Errorf("zero Metadata = %v, want nil", turn.Metadata)
	}
}

func TestTurn_MessageContent(t *testing.T) {
	turn := Turn{
		Input:     NewHumanMessage("What is 2+2?"),
		Output:    NewAIMessage("4"),
		Timestamp: time.Now(),
	}

	// Access input text through Message interface
	inputParts := turn.Input.GetContent()
	if len(inputParts) != 1 {
		t.Fatalf("len(Input.GetContent()) = %d, want 1", len(inputParts))
	}
	if tp, ok := inputParts[0].(TextPart); !ok || tp.Text != "What is 2+2?" {
		t.Errorf("Input content = %v, want TextPart with %q", inputParts[0], "What is 2+2?")
	}

	// Access output text through Message interface
	outputParts := turn.Output.GetContent()
	if len(outputParts) != 1 {
		t.Fatalf("len(Output.GetContent()) = %d, want 1", len(outputParts))
	}
	if tp, ok := outputParts[0].(TextPart); !ok || tp.Text != "4" {
		t.Errorf("Output content = %v, want TextPart with %q", outputParts[0], "4")
	}
}

func TestTurn_Metadata(t *testing.T) {
	turn := Turn{
		Input:     NewHumanMessage("test"),
		Output:    NewAIMessage("response"),
		Timestamp: time.Now(),
		Metadata: map[string]any{
			"duration_ms": 250,
			"model":       "gpt-4o",
			"retries":     0,
		},
	}

	if v, ok := turn.Metadata["duration_ms"].(int); !ok || v != 250 {
		t.Errorf("Metadata[\"duration_ms\"] = %v, want 250", turn.Metadata["duration_ms"])
	}
	if v, ok := turn.Metadata["model"].(string); !ok || v != "gpt-4o" {
		t.Errorf("Metadata[\"model\"] = %v, want %q", turn.Metadata["model"], "gpt-4o")
	}
}

func TestSession_AppendTurn(t *testing.T) {
	now := time.Now()
	s := Session{
		ID:        "sess-append",
		Turns:     nil,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Append turns manually (no method, just slice append)
	s.Turns = append(s.Turns, Turn{
		Input:     NewHumanMessage("first"),
		Output:    NewAIMessage("resp1"),
		Timestamp: now,
	})
	if len(s.Turns) != 1 {
		t.Fatalf("len(Turns) = %d, want 1", len(s.Turns))
	}

	s.Turns = append(s.Turns, Turn{
		Input:     NewHumanMessage("second"),
		Output:    NewAIMessage("resp2"),
		Timestamp: now.Add(time.Minute),
	})
	if len(s.Turns) != 2 {
		t.Fatalf("len(Turns) = %d, want 2", len(s.Turns))
	}

	// Verify order is preserved
	if s.Turns[0].Input.GetContent()[0].(TextPart).Text != "first" {
		t.Error("Turns[0] input should be 'first'")
	}
	if s.Turns[1].Input.GetContent()[0].(TextPart).Text != "second" {
		t.Error("Turns[1] input should be 'second'")
	}
}
