package evolving

import (
	"context"
	"errors"
	"iter"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

type mockAgent struct {
	id string
}

func (m *mockAgent) ID() string              { return m.id }
func (m *mockAgent) Persona() agent.Persona  { return agent.Persona{Role: "test"} }
func (m *mockAgent) Tools() []tool.Tool      { return nil }
func (m *mockAgent) Children() []agent.Agent { return nil }
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

// stubDistiller is a test distiller that records calls and can return errors.
type stubDistiller struct {
	mu    sync.Mutex
	calls int
	done  chan struct{}
	err   error
	out   []Experience
}

func (s *stubDistiller) Distill(_ context.Context, _ []Interaction) ([]Experience, error) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	if s.done != nil {
		defer func() {
			select {
			case s.done <- struct{}{}:
			default:
			}
		}()
	}
	return s.out, s.err
}

// stubOptimizer is a test optimizer.
type stubOptimizer struct {
	mu    sync.Mutex
	calls int
	err   error
	out   []Suggestion
}

func (s *stubOptimizer) Optimize(_ context.Context, _ []Experience) ([]Suggestion, error) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	return s.out, s.err
}

func TestWithOptimizerAndDistiller(t *testing.T) {
	d := &stubDistiller{
		out:  []Experience{{Category: "test", Pattern: "p", Confidence: 0.5}},
		done: make(chan struct{}, 1),
	}
	o := &stubOptimizer{
		out: []Suggestion{{Type: "t", Description: "d", Priority: 5, Confidence: 0.7}},
	}
	inner := &mockAgent{id: "test"}
	ea := New(inner, WithOptimizer(o), WithDistiller(d), WithLearnEveryN(1))

	if _, err := ea.Invoke(context.Background(), "hello"); err != nil {
		t.Fatalf("Invoke: %v", err)
	}

	// Wait for async learn to complete.
	select {
	case <-d.done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for learn")
	}
	// Give optimizer time to run after distiller signals.
	time.Sleep(50 * time.Millisecond)

	exps := ea.Experiences()
	if len(exps) != 1 || exps[0].Category != "test" {
		t.Errorf("Experiences = %v, want 1 with category 'test'", exps)
	}
	sugs := ea.Suggestions()
	if len(sugs) != 1 || sugs[0].Type != "t" {
		t.Errorf("Suggestions = %v, want 1 with type 't'", sugs)
	}
}

func TestLearn_DistillerError(t *testing.T) {
	d := &stubDistiller{
		err:  errors.New("distill failed"),
		done: make(chan struct{}, 1),
	}
	o := &stubOptimizer{}
	inner := &mockAgent{id: "test"}
	ea := New(inner, WithOptimizer(o), WithDistiller(d), WithLearnEveryN(1))

	if _, err := ea.Invoke(context.Background(), "hi"); err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	select {
	case <-d.done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	time.Sleep(50 * time.Millisecond)

	o.mu.Lock()
	calls := o.calls
	o.mu.Unlock()
	if calls != 0 {
		t.Errorf("optimizer should not be called when distiller errors, got calls=%d", calls)
	}
	if exps := ea.Experiences(); len(exps) != 0 {
		t.Errorf("expected no experiences on distiller error, got %v", exps)
	}
}

func TestLearn_OptimizerError(t *testing.T) {
	d := &stubDistiller{
		out:  []Experience{{Category: "x"}},
		done: make(chan struct{}, 1),
	}
	o := &stubOptimizer{err: errors.New("optimize failed")}
	inner := &mockAgent{id: "test"}
	ea := New(inner, WithOptimizer(o), WithDistiller(d), WithLearnEveryN(1))

	if _, err := ea.Invoke(context.Background(), "hi"); err != nil {
		t.Fatalf("Invoke: %v", err)
	}
	select {
	case <-d.done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
	time.Sleep(50 * time.Millisecond)

	if sugs := ea.Suggestions(); len(sugs) != 0 {
		t.Errorf("expected no suggestions on optimizer error, got %v", sugs)
	}
}

// errorStreamAgent returns an error from its stream.
type errorStreamAgent struct{ mockAgent }

func (e *errorStreamAgent) Stream(_ context.Context, _ string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		yield(agent.Event{Type: agent.EventText, Text: "partial"}, nil)
		yield(agent.Event{}, errors.New("stream failure"))
	}
}

func TestEvolvingAgent_StreamError(t *testing.T) {
	inner := &errorStreamAgent{mockAgent: mockAgent{id: "err"}}
	ea := New(inner, WithLearnEveryN(1000))

	var gotErr bool
	for _, err := range ea.Stream(context.Background(), "hello") {
		if err != nil {
			gotErr = true
		}
	}
	if !gotErr {
		t.Error("expected stream error")
	}
	if ea.InteractionCount() != 1 {
		t.Errorf("InteractionCount = %d, want 1", ea.InteractionCount())
	}
}

// earlyStopAgent yields many events; consumer stops early.
func TestEvolvingAgent_StreamEarlyStop(t *testing.T) {
	inner := &mockAgent{id: "test"}
	ea := New(inner, WithLearnEveryN(1000))

	next, stop := iter.Pull2(ea.Stream(context.Background(), "hello"))
	defer stop()
	_, _, ok := next()
	if !ok {
		t.Fatal("expected at least one event")
	}
	stop()
}

func TestExperiencesAndSuggestions_EmptyReturnsEmptySlice(t *testing.T) {
	inner := &mockAgent{id: "test"}
	ea := New(inner, WithLearnEveryN(1000))

	if exps := ea.Experiences(); len(exps) != 0 {
		t.Errorf("expected empty experiences, got %v", exps)
	}
	if sugs := ea.Suggestions(); len(sugs) != 0 {
		t.Errorf("expected empty suggestions, got %v", sugs)
	}
}
