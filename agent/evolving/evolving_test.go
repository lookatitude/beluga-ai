package evolving

import (
	"context"
	"iter"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

type mockAgent struct {
	id string
}

func (m *mockAgent) ID() string                          { return m.id }
func (m *mockAgent) Persona() agent.Persona              { return agent.Persona{Role: "test"} }
func (m *mockAgent) Tools() []tool.Tool                  { return nil }
func (m *mockAgent) Children() []agent.Agent             { return nil }
func (m *mockAgent) Invoke(_ context.Context, input string, _ ...agent.Option) (string, error) {
	return "response: " + input, nil
}
func (m *mockAgent) Stream(_ context.Context, input string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		yield(agent.Event{Type: agent.EventText, Text: "streamed: " + input}, nil)
	}
}

var _ agent.Agent = (*mockAgent)(nil)

func TestEvolvingAgent_Invoke(t *testing.T) {
	inner := &mockAgent{id: "test"}
	ea := New(inner, WithLearnEveryN(100)) // Don't trigger learn during test.

	result, err := ea.Invoke(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	if result != "response: hello" {
		t.Errorf("result = %q, want %q", result, "response: hello")
	}

	if ea.InteractionCount() != 1 {
		t.Errorf("InteractionCount = %d, want 1", ea.InteractionCount())
	}
}

func TestEvolvingAgent_Stream(t *testing.T) {
	inner := &mockAgent{id: "test"}
	ea := New(inner, WithLearnEveryN(100))

	var text string
	for evt, err := range ea.Stream(context.Background(), "hello") {
		if err != nil {
			t.Fatalf("Stream error: %v", err)
		}
		text += evt.Text
	}

	if text != "streamed: hello" {
		t.Errorf("text = %q, want %q", text, "streamed: hello")
	}
	if ea.InteractionCount() != 1 {
		t.Errorf("InteractionCount = %d, want 1", ea.InteractionCount())
	}
}

func TestEvolvingAgent_DelegatesIdentity(t *testing.T) {
	inner := &mockAgent{id: "delegate-test"}
	ea := New(inner)

	if ea.ID() != "delegate-test" {
		t.Errorf("ID = %q, want delegate-test", ea.ID())
	}
	if ea.Persona().Role != "test" {
		t.Errorf("Persona.Role = %q, want test", ea.Persona().Role)
	}
	if ea.Tools() != nil {
		t.Errorf("Tools = %v, want nil", ea.Tools())
	}
	if ea.Children() != nil {
		t.Errorf("Children = %v, want nil", ea.Children())
	}
}

func TestEvolvingAgent_MaxMemory(t *testing.T) {
	inner := &mockAgent{id: "test"}
	ea := New(inner, WithMaxMemory(5), WithLearnEveryN(1000))

	for i := 0; i < 10; i++ {
		ea.Invoke(context.Background(), "hello")
	}

	ea.mu.Lock()
	count := len(ea.interactions)
	ea.mu.Unlock()

	if count > 5 {
		t.Errorf("interactions = %d, want <= 5", count)
	}
}

func TestSimpleDistiller(t *testing.T) {
	distiller := &SimpleDistiller{}
	interactions := []Interaction{
		{Input: "hi", Output: "hello", Success: true, Duration: 100 * time.Millisecond},
		{Input: "bye", Output: "goodbye", Success: true, Duration: 200 * time.Millisecond},
		{Input: "err", Output: "", Success: false, Duration: 50 * time.Millisecond},
	}

	experiences, err := distiller.Distill(context.Background(), interactions)
	if err != nil {
		t.Fatalf("Distill: %v", err)
	}

	if len(experiences) == 0 {
		t.Fatal("expected at least one experience")
	}

	var hasReliability, hasPerformance bool
	for _, exp := range experiences {
		if exp.Category == "reliability" {
			hasReliability = true
		}
		if exp.Category == "performance" {
			hasPerformance = true
		}
	}

	if !hasReliability {
		t.Error("expected reliability experience")
	}
	if !hasPerformance {
		t.Error("expected performance experience")
	}
}

func TestSimpleDistiller_Empty(t *testing.T) {
	distiller := &SimpleDistiller{}
	experiences, err := distiller.Distill(context.Background(), nil)
	if err != nil {
		t.Fatalf("Distill: %v", err)
	}
	if experiences != nil {
		t.Errorf("expected nil for empty input, got %v", experiences)
	}
}

func TestFrequencyOptimizer(t *testing.T) {
	optimizer := &FrequencyOptimizer{}
	experiences := []Experience{
		{Category: "reliability", Confidence: 0.5},
		{Category: "performance", Confidence: 0.8},
	}

	suggestions, err := optimizer.Optimize(context.Background(), experiences)
	if err != nil {
		t.Fatalf("Optimize: %v", err)
	}

	if len(suggestions) == 0 {
		t.Fatal("expected at least one suggestion")
	}

	// Should be sorted by priority (descending).
	for i := 1; i < len(suggestions); i++ {
		if suggestions[i].Priority > suggestions[i-1].Priority {
			t.Error("suggestions should be sorted by priority descending")
		}
	}
}

func TestFrequencyOptimizer_Empty(t *testing.T) {
	optimizer := &FrequencyOptimizer{}
	suggestions, err := optimizer.Optimize(context.Background(), nil)
	if err != nil {
		t.Fatalf("Optimize: %v", err)
	}
	if suggestions != nil {
		t.Errorf("expected nil for empty input")
	}
}
