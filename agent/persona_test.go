package agent

import (
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

func TestPersona_ToSystemMessage(t *testing.T) {
	tests := []struct {
		name       string
		persona    Persona
		wantNil    bool
		wantParts  []string
		dontWant   []string
	}{
		{
			name:    "empty persona returns nil",
			persona: Persona{},
			wantNil: true,
		},
		{
			name: "role only",
			persona: Persona{
				Role: "software engineer",
			},
			wantParts: []string{"You are a software engineer."},
		},
		{
			name: "goal only",
			persona: Persona{
				Goal: "help users write clean code",
			},
			wantParts: []string{"Your goal is to help users write clean code."},
		},
		{
			name: "backstory only",
			persona: Persona{
				Backstory: "You have 10 years of experience.",
			},
			wantParts: []string{"You have 10 years of experience."},
		},
		{
			name: "traits only",
			persona: Persona{
				Traits: []string{"concise", "friendly"},
			},
			wantParts: []string{"Your traits: concise, friendly."},
		},
		{
			name: "all fields",
			persona: Persona{
				Role:      "data scientist",
				Goal:      "analyze data accurately",
				Backstory: "Expert in ML and statistics.",
				Traits:    []string{"analytical", "precise"},
			},
			wantParts: []string{
				"You are a data scientist.",
				"Your goal is to analyze data accurately.",
				"Expert in ML and statistics.",
				"Your traits: analytical, precise.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.persona.ToSystemMessage()

			if tt.wantNil {
				if msg != nil {
					t.Fatalf("expected nil, got %v", msg)
				}
				return
			}

			if msg == nil {
				t.Fatal("expected non-nil SystemMessage")
			}

			if msg.GetRole() != schema.RoleSystem {
				t.Errorf("role = %s, want %s", msg.GetRole(), schema.RoleSystem)
			}

			text := msg.Text()
			for _, part := range tt.wantParts {
				if !strings.Contains(text, part) {
					t.Errorf("text should contain %q, got:\n%s", part, text)
				}
			}
			for _, part := range tt.dontWant {
				if strings.Contains(text, part) {
					t.Errorf("text should NOT contain %q, got:\n%s", part, text)
				}
			}
		})
	}
}

func TestPersona_IsEmpty(t *testing.T) {
	tests := []struct {
		name    string
		persona Persona
		want    bool
	}{
		{name: "fully empty", persona: Persona{}, want: true},
		{name: "has role", persona: Persona{Role: "test"}, want: false},
		{name: "has goal", persona: Persona{Goal: "test"}, want: false},
		{name: "has backstory", persona: Persona{Backstory: "test"}, want: false},
		{name: "has traits", persona: Persona{Traits: []string{"a"}}, want: false},
		{name: "empty traits slice", persona: Persona{Traits: []string{}}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.persona.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}
