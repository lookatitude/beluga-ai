package agent

import (
	"context"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

func TestRegisterPlannerAndNew(t *testing.T) {
	// Save and restore registry state.
	plannerMu.Lock()
	orig := make(map[string]PlannerFactory, len(plannerRegistry))
	for k, v := range plannerRegistry {
		orig[k] = v
	}
	plannerMu.Unlock()
	t.Cleanup(func() {
		plannerMu.Lock()
		plannerRegistry = orig
		plannerMu.Unlock()
	})

	plannerMu.Lock()
	plannerRegistry = make(map[string]PlannerFactory)
	plannerMu.Unlock()

	RegisterPlanner("test-planner", func(cfg PlannerConfig) (Planner, error) {
		return &mockPlanner{name: "test-planner"}, nil
	})

	p, err := NewPlanner("test-planner", PlannerConfig{})
	if err != nil {
		t.Fatalf("NewPlanner error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil planner")
	}
}

func TestNewPlanner_NotRegistered(t *testing.T) {
	plannerMu.Lock()
	orig := make(map[string]PlannerFactory, len(plannerRegistry))
	for k, v := range plannerRegistry {
		orig[k] = v
	}
	plannerMu.Unlock()
	t.Cleanup(func() {
		plannerMu.Lock()
		plannerRegistry = orig
		plannerMu.Unlock()
	})

	plannerMu.Lock()
	plannerRegistry = make(map[string]PlannerFactory)
	plannerMu.Unlock()

	_, err := NewPlanner("nonexistent", PlannerConfig{})
	if err == nil {
		t.Fatal("expected error for unregistered planner")
	}
}

func TestListPlanners(t *testing.T) {
	plannerMu.Lock()
	orig := make(map[string]PlannerFactory, len(plannerRegistry))
	for k, v := range plannerRegistry {
		orig[k] = v
	}
	plannerMu.Unlock()
	t.Cleanup(func() {
		plannerMu.Lock()
		plannerRegistry = orig
		plannerMu.Unlock()
	})

	plannerMu.Lock()
	plannerRegistry = make(map[string]PlannerFactory)
	plannerMu.Unlock()

	RegisterPlanner("zebra", func(cfg PlannerConfig) (Planner, error) { return nil, nil })
	RegisterPlanner("alpha", func(cfg PlannerConfig) (Planner, error) { return nil, nil })

	names := ListPlanners()
	if len(names) != 2 {
		t.Fatalf("expected 2 planners, got %d", len(names))
	}
	if names[0] != "alpha" || names[1] != "zebra" {
		t.Errorf("expected sorted [alpha, zebra], got %v", names)
	}
}

func TestListPlanners_Empty(t *testing.T) {
	plannerMu.Lock()
	orig := make(map[string]PlannerFactory, len(plannerRegistry))
	for k, v := range plannerRegistry {
		orig[k] = v
	}
	plannerMu.Unlock()
	t.Cleanup(func() {
		plannerMu.Lock()
		plannerRegistry = orig
		plannerMu.Unlock()
	})

	plannerMu.Lock()
	plannerRegistry = make(map[string]PlannerFactory)
	plannerMu.Unlock()

	names := ListPlanners()
	if len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}
}

func TestRegisterPlanner_Overwrite(t *testing.T) {
	plannerMu.Lock()
	orig := make(map[string]PlannerFactory, len(plannerRegistry))
	for k, v := range plannerRegistry {
		orig[k] = v
	}
	plannerMu.Unlock()
	t.Cleanup(func() {
		plannerMu.Lock()
		plannerRegistry = orig
		plannerMu.Unlock()
	})

	plannerMu.Lock()
	plannerRegistry = make(map[string]PlannerFactory)
	plannerMu.Unlock()

	RegisterPlanner("dup", func(cfg PlannerConfig) (Planner, error) {
		return &mockPlanner{name: "first"}, nil
	})
	RegisterPlanner("dup", func(cfg PlannerConfig) (Planner, error) {
		return &mockPlanner{name: "second"}, nil
	})

	p, err := NewPlanner("dup", PlannerConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mp := p.(*mockPlanner)
	if mp.name != "second" {
		t.Errorf("expected overwritten planner, got %q", mp.name)
	}
}

// mockPlanner implements Planner for testing.
type mockPlanner struct {
	name string
}

func (m *mockPlanner) Plan(ctx context.Context, state PlannerState) ([]Action, error) {
	return []Action{{Type: ActionFinish, Message: "done"}}, nil
}

func (m *mockPlanner) Replan(ctx context.Context, state PlannerState) ([]Action, error) {
	return m.Plan(ctx, state)
}

// mockAgent implements Agent for testing.
type mockAgent struct {
	id string
}

func (a *mockAgent) ID() string                                { return a.id }
func (a *mockAgent) Persona() Persona                          { return Persona{} }
func (a *mockAgent) Tools() []tool.Tool                        { return nil }
func (a *mockAgent) Children() []Agent                         { return nil }
func (a *mockAgent) Invoke(ctx context.Context, input string, opts ...Option) (string, error) {
	return "mock:" + input, nil
}
func (a *mockAgent) Stream(ctx context.Context, input string, opts ...Option) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		yield(Event{Type: EventText, Text: "mock:" + input, AgentID: a.id}, nil)
		yield(Event{Type: EventDone, AgentID: a.id}, nil)
	}
}

// streamMockAgent is a mock agent that returns its id and result from Invoke.
type streamMockAgent struct {
	id     string
	result string
}

func (a *streamMockAgent) ID() string                 { return a.id }
func (a *streamMockAgent) Persona() Persona           { return Persona{} }
func (a *streamMockAgent) Tools() []tool.Tool         { return nil }
func (a *streamMockAgent) Children() []Agent          { return nil }
func (a *streamMockAgent) Invoke(ctx context.Context, input string, opts ...Option) (string, error) {
	return a.result, nil
}
func (a *streamMockAgent) Stream(ctx context.Context, input string, opts ...Option) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		yield(Event{Type: EventText, Text: a.result, AgentID: a.id}, nil)
		yield(Event{Type: EventDone, AgentID: a.id}, nil)
	}
}

// Ensure mockAgent has the right return type by using it to populate an Agent slice.
var _ Agent = (*mockAgent)(nil)
var _ Agent = (*streamMockAgent)(nil)

// A simple stub for tests that need schema.Message verification.
func msgText(m schema.Message) string {
	for _, p := range m.GetContent() {
		if tp, ok := p.(schema.TextPart); ok {
			return tp.Text
		}
	}
	return ""
}
