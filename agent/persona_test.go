package agent

import (
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/schema"
)

func assertPersonaMessage(t *testing.T, msg *schema.SystemMessage, wantParts []string, dontWant []string) {
	t.Helper()
	if msg.GetRole() != schema.RoleSystem {
		t.Errorf("role = %s, want %s", msg.GetRole(), schema.RoleSystem)
	}
	text := msg.Text()
	for _, part := range wantParts {
		if !strings.Contains(text, part) {
			t.Errorf("text should contain %q, got:\n%s", part, text)
		}
	}
	for _, part := range dontWant {
		if strings.Contains(text, part) {
			t.Errorf("text should NOT contain %q, got:\n%s", part, text)
		}
	}
}

func TestPersona_ToSystemMessage(t *testing.T) {
	t.Run("empty persona returns nil", func(t *testing.T) {
		msg := (Persona{}).ToSystemMessage()
		if msg != nil {
			t.Fatalf("expected nil, got %v", msg)
		}
	})

	t.Run("role only", func(t *testing.T) {
		p := Persona{Role: "software engineer"}
		msg := p.ToSystemMessage()
		if msg == nil {
			t.Fatal("expected non-nil SystemMessage")
		}
		assertPersonaMessage(t, msg, []string{"You are a software engineer."}, nil)
	})

	t.Run("goal only", func(t *testing.T) {
		p := Persona{Goal: "help users write clean code"}
		msg := p.ToSystemMessage()
		if msg == nil {
			t.Fatal("expected non-nil SystemMessage")
		}
		assertPersonaMessage(t, msg, []string{"Your goal is to help users write clean code."}, nil)
	})

	t.Run("backstory only", func(t *testing.T) {
		p := Persona{Backstory: "You have 10 years of experience."}
		msg := p.ToSystemMessage()
		if msg == nil {
			t.Fatal("expected non-nil SystemMessage")
		}
		assertPersonaMessage(t, msg, []string{"You have 10 years of experience."}, nil)
	})

	t.Run("traits only", func(t *testing.T) {
		p := Persona{Traits: []string{"concise", "friendly"}}
		msg := p.ToSystemMessage()
		if msg == nil {
			t.Fatal("expected non-nil SystemMessage")
		}
		assertPersonaMessage(t, msg, []string{"Your traits: concise, friendly."}, nil)
	})

	t.Run("all fields", func(t *testing.T) {
		p := Persona{
			Role:      "data scientist",
			Goal:      "analyze data accurately",
			Backstory: "Expert in ML and statistics.",
			Traits:    []string{"analytical", "precise"},
		}
		msg := p.ToSystemMessage()
		if msg == nil {
			t.Fatal("expected non-nil SystemMessage")
		}
		wantParts := []string{
			"You are a data scientist.",
			"Your goal is to analyze data accurately.",
			"Expert in ML and statistics.",
			"Your traits: analytical, precise.",
		}
		assertPersonaMessage(t, msg, wantParts, nil)
	})
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
