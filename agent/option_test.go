package agent

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/tool"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.maxIterations != 10 {
		t.Errorf("maxIterations = %d, want 10", cfg.maxIterations)
	}
	if cfg.timeout != 5*time.Minute {
		t.Errorf("timeout = %v, want %v", cfg.timeout, 5*time.Minute)
	}
	if cfg.plannerName != "react" {
		t.Errorf("plannerName = %q, want %q", cfg.plannerName, "react")
	}
	if cfg.llm != nil {
		t.Error("llm should be nil by default")
	}
	if cfg.planner != nil {
		t.Error("planner should be nil by default")
	}
	if cfg.tools != nil {
		t.Error("tools should be nil by default")
	}
	if cfg.handoffs != nil {
		t.Error("handoffs should be nil by default")
	}
	if cfg.memory != nil {
		t.Error("memory should be nil by default")
	}
	if cfg.children != nil {
		t.Error("children should be nil by default")
	}
}

func TestWithLLM(t *testing.T) {
	cfg := defaultConfig()
	// Using nil here just to test the option applies.
	WithLLM(nil)(&cfg)
	if cfg.llm != nil {
		t.Error("expected nil LLM when set to nil")
	}
}

func TestWithTools(t *testing.T) {
	cfg := defaultConfig()
	tools := []tool.Tool{&simpleTool{toolName: "test"}}
	WithTools(tools)(&cfg)
	if len(cfg.tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(cfg.tools))
	}
	if cfg.tools[0].Name() != "test" {
		t.Errorf("tool name = %q, want %q", cfg.tools[0].Name(), "test")
	}
}

func TestWithPersona(t *testing.T) {
	cfg := defaultConfig()
	p := Persona{Role: "tester", Goal: "test things"}
	WithPersona(p)(&cfg)
	if cfg.persona.Role != "tester" {
		t.Errorf("persona.Role = %q, want %q", cfg.persona.Role, "tester")
	}
	if cfg.persona.Goal != "test things" {
		t.Errorf("persona.Goal = %q, want %q", cfg.persona.Goal, "test things")
	}
}

func TestWithMaxIterations(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want int
	}{
		{name: "positive", n: 5, want: 5},
		{name: "zero ignored", n: 0, want: 10},
		{name: "negative ignored", n: -1, want: 10},
		{name: "one", n: 1, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultConfig()
			WithMaxIterations(tt.n)(&cfg)
			if cfg.maxIterations != tt.want {
				t.Errorf("maxIterations = %d, want %d", cfg.maxIterations, tt.want)
			}
		})
	}
}

func TestWithTimeout(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want time.Duration
	}{
		{name: "positive", d: 30 * time.Second, want: 30 * time.Second},
		{name: "zero ignored", d: 0, want: 5 * time.Minute},
		{name: "negative ignored", d: -1, want: 5 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaultConfig()
			WithTimeout(tt.d)(&cfg)
			if cfg.timeout != tt.want {
				t.Errorf("timeout = %v, want %v", cfg.timeout, tt.want)
			}
		})
	}
}

func TestWithHooks(t *testing.T) {
	cfg := defaultConfig()
	called := false
	h := Hooks{
		OnStart: func(ctx context.Context, input string) error {
			called = true
			return nil
		},
	}
	WithHooks(h)(&cfg)

	if cfg.hooks.OnStart == nil {
		t.Fatal("hooks.OnStart should not be nil")
	}
	_ = cfg.hooks.OnStart(context.Background(), "test")
	if !called {
		t.Error("hooks.OnStart was not called")
	}
}

func TestWithHandoffs(t *testing.T) {
	cfg := defaultConfig()
	target := &mockAgent{id: "helper"}
	handoffs := []Handoff{HandoffTo(target, "help")}
	WithHandoffs(handoffs)(&cfg)

	if len(cfg.handoffs) != 1 {
		t.Fatalf("expected 1 handoff, got %d", len(cfg.handoffs))
	}
	if cfg.handoffs[0].TargetAgent.ID() != "helper" {
		t.Errorf("handoff target = %q, want %q", cfg.handoffs[0].TargetAgent.ID(), "helper")
	}
}

func TestWithMemory(t *testing.T) {
	cfg := defaultConfig()
	m := &mockMemory{}
	WithMemory(m)(&cfg)
	if cfg.memory == nil {
		t.Error("memory should not be nil")
	}
}

func TestWithPlanner(t *testing.T) {
	cfg := defaultConfig()
	p := &mockPlanner{name: "custom"}
	WithPlanner(p)(&cfg)
	if cfg.planner == nil {
		t.Error("planner should not be nil")
	}
}

func TestWithPlannerName(t *testing.T) {
	cfg := defaultConfig()
	WithPlannerName("reflexion")(&cfg)
	if cfg.plannerName != "reflexion" {
		t.Errorf("plannerName = %q, want %q", cfg.plannerName, "reflexion")
	}
}

func TestWithChildren(t *testing.T) {
	cfg := defaultConfig()
	children := []Agent{&mockAgent{id: "child1"}, &mockAgent{id: "child2"}}
	WithChildren(children)(&cfg)

	if len(cfg.children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(cfg.children))
	}
	if cfg.children[0].ID() != "child1" {
		t.Errorf("children[0].ID() = %q, want %q", cfg.children[0].ID(), "child1")
	}
}

func TestWithMetadata(t *testing.T) {
	cfg := defaultConfig()
	meta := map[string]any{"key": "value", "count": 42}
	WithMetadata(meta)(&cfg)

	if cfg.metadata["key"] != "value" {
		t.Errorf("metadata[key] = %v, want %q", cfg.metadata["key"], "value")
	}
	if cfg.metadata["count"] != 42 {
		t.Errorf("metadata[count] = %v, want 42", cfg.metadata["count"])
	}
}

// mockMemory implements Memory for testing.
type mockMemory struct{}

func (m *mockMemory) Save(_ context.Context, _ string, _ []any) error { return nil }
func (m *mockMemory) Load(_ context.Context, _ string) ([]any, error) { return nil, nil }
