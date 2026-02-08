package agent

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/tool"
)

// simpleTool is a minimal tool.Tool for testing.
type simpleTool struct {
	toolName string
}

func (s *simpleTool) Name() string                { return s.toolName }
func (s *simpleTool) Description() string         { return "test tool" }
func (s *simpleTool) InputSchema() map[string]any { return nil }
func (s *simpleTool) Execute(_ context.Context, _ map[string]any) (*tool.Result, error) {
	return tool.TextResult("ok"), nil
}

var _ tool.Tool = (*simpleTool)(nil)

func TestBuildCard_BasicAgent(t *testing.T) {
	a := &mockAgent{id: "test-agent"}
	card := BuildCard(a)

	if card.Name != "test-agent" {
		t.Errorf("Name = %q, want %q", card.Name, "test-agent")
	}
	if card.Description != "" {
		t.Errorf("Description = %q, want empty", card.Description)
	}
	if len(card.Skills) != 0 {
		t.Errorf("Skills = %v, want empty", card.Skills)
	}
}

func TestBuildCard_WithGoalPersona(t *testing.T) {
	a := New("agent-with-goal",
		WithPersona(Persona{
			Role: "researcher",
			Goal: "find answers",
		}),
	)
	card := BuildCard(a)

	if card.Name != "agent-with-goal" {
		t.Errorf("Name = %q, want %q", card.Name, "agent-with-goal")
	}
	// Goal takes priority over Role for description.
	if card.Description != "find answers" {
		t.Errorf("Description = %q, want %q", card.Description, "find answers")
	}
}

func TestBuildCard_WithRoleOnly(t *testing.T) {
	a := New("agent-with-role",
		WithPersona(Persona{
			Role: "data analyst",
		}),
	)
	card := BuildCard(a)

	if card.Description != "data analyst" {
		t.Errorf("Description = %q, want %q", card.Description, "data analyst")
	}
}

func TestBuildCard_WithTools(t *testing.T) {
	tools := []tool.Tool{
		&simpleTool{toolName: "search"},
		&simpleTool{toolName: "calculate"},
	}
	a := New("tool-agent",
		WithTools(tools),
		WithPersona(Persona{Goal: "help users"}),
	)
	card := BuildCard(a)

	if len(card.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(card.Skills))
	}
	if card.Skills[0] != "search" {
		t.Errorf("Skills[0] = %q, want %q", card.Skills[0], "search")
	}
	if card.Skills[1] != "calculate" {
		t.Errorf("Skills[1] = %q, want %q", card.Skills[1], "calculate")
	}
}

func TestBuildCard_WithHandoffs(t *testing.T) {
	target := &mockAgent{id: "helper"}
	a := New("main-agent",
		WithHandoffs([]Handoff{HandoffTo(target, "Transfer to helper")}),
	)
	card := BuildCard(a)

	// Handoffs become tools, so they appear as skills.
	if len(card.Skills) != 1 {
		t.Fatalf("expected 1 skill (handoff tool), got %d", len(card.Skills))
	}
	if card.Skills[0] != "transfer_to_helper" {
		t.Errorf("Skills[0] = %q, want %q", card.Skills[0], "transfer_to_helper")
	}
}

func TestBuildCard_EmptyPersona(t *testing.T) {
	a := New("empty-persona")
	card := BuildCard(a)

	if card.Description != "" {
		t.Errorf("Description = %q, want empty for empty persona", card.Description)
	}
}

func TestAgentCard_Fields(t *testing.T) {
	card := AgentCard{
		Name:        "test",
		Description: "desc",
		URL:         "http://example.com",
		Skills:      []string{"a", "b"},
		Protocols:   []string{"a2a"},
	}
	if card.Name != "test" {
		t.Errorf("Name = %q, want %q", card.Name, "test")
	}
	if card.URL != "http://example.com" {
		t.Errorf("URL = %q, want %q", card.URL, "http://example.com")
	}
	if len(card.Protocols) != 1 || card.Protocols[0] != "a2a" {
		t.Errorf("Protocols = %v, want [a2a]", card.Protocols)
	}
}
