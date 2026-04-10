package context

import (
	gocontext "context"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

func TestDefaultPipeline_Execute(t *testing.T) {
	pipeline := NewPipeline(
		WithStep(&RecencyBooster{boost: 0.1}),
		WithStep(&RelevanceRanker{}),
	)

	input := PipelineInput{
		Query: "test query",
		Messages: []schema.Message{
			schema.NewHumanMessage("hello"),
			schema.NewAIMessage("hi there"),
		},
		MaxTokens: 1000,
	}

	output, err := pipeline.Execute(gocontext.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(output.Items) == 0 {
		t.Error("expected items in output")
	}
	if len(output.StepsExecuted) != 2 {
		t.Errorf("StepsExecuted = %d, want 2", len(output.StepsExecuted))
	}
	if output.TokenEstimate <= 0 {
		t.Errorf("TokenEstimate = %d, want > 0", output.TokenEstimate)
	}
}

func TestDefaultPipeline_ContextCancellation(t *testing.T) {
	ctx, cancel := gocontext.WithCancel(gocontext.Background())
	cancel()

	pipeline := NewPipeline(WithStep(&RelevanceRanker{}))
	_, err := pipeline.Execute(ctx, PipelineInput{MaxTokens: 100})
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestRelevanceRanker(t *testing.T) {
	ranker := &RelevanceRanker{}
	items := []ContextItem{
		{ID: "1", Score: 0.3},
		{ID: "2", Score: 0.9},
		{ID: "3", Score: 0.5},
	}

	result, err := ranker.Process(gocontext.Background(), items)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	if result[0].ID != "2" {
		t.Errorf("first item = %q, want 2 (highest score)", result[0].ID)
	}
	if result[2].ID != "1" {
		t.Errorf("last item = %q, want 1 (lowest score)", result[2].ID)
	}
}

func TestTokenBudgetFilter(t *testing.T) {
	filter := NewTokenBudgetFilter(10) // Very small budget.
	items := []ContextItem{
		{ID: "1", Content: "short"},                                  // ~1 token
		{ID: "2", Content: "this is a much longer piece of content"}, // ~8 tokens
		{ID: "3", Content: "another longer content here"},            // ~6 tokens
	}

	result, err := filter.Process(gocontext.Background(), items)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	totalTokens := 0
	for _, item := range result {
		tokens := len(item.Content) / 4
		if tokens == 0 {
			tokens = 1
		}
		totalTokens += tokens
	}

	if totalTokens > 10 {
		t.Errorf("total tokens = %d, exceeds budget of 10", totalTokens)
	}
}

func TestDuplicateFilter(t *testing.T) {
	filter := &DuplicateFilter{}
	items := []ContextItem{
		{ID: "1", Content: "hello"},
		{ID: "2", Content: "world"},
		{ID: "3", Content: "hello"}, // duplicate
	}

	result, err := filter.Process(gocontext.Background(), items)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("items = %d, want 2 (duplicate removed)", len(result))
	}
}

func TestRecencyBooster(t *testing.T) {
	booster := NewRecencyBooster(0.5)
	items := []ContextItem{
		{ID: "1", Score: 0.5},
		{ID: "2", Score: 0.5},
		{ID: "3", Score: 0.5},
	}

	result, err := booster.Process(gocontext.Background(), items)
	if err != nil {
		t.Fatalf("Process: %v", err)
	}

	// Last item should have highest score (most boosted).
	if result[2].Score <= result[0].Score {
		t.Errorf("last item score (%f) should be > first item score (%f)", result[2].Score, result[0].Score)
	}
}

func TestPipelineBuilder(t *testing.T) {
	pipeline := NewBuilder().
		WithRetrieve(&DuplicateFilter{}). // Using as placeholder steps.
		WithRank(&RelevanceRanker{}).
		WithFilter(NewTokenBudgetFilter(1000)).
		WithStructure(&RecencyBooster{boost: 0.1}).
		SetMaxTokens(2000).
		Build()

	input := PipelineInput{
		Query:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("hello")},
	}

	output, err := pipeline.Execute(gocontext.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if len(output.StepsExecuted) != 4 {
		t.Errorf("StepsExecuted = %d, want 4", len(output.StepsExecuted))
	}
}

func TestStepRegistry(t *testing.T) {
	names := ListSteps()
	wantAll := []string{"relevance_rank", "token_budget_filter", "duplicate_filter", "recency_boost"}
	if len(names) < len(wantAll) {
		t.Errorf("expected at least %d registered steps, got %d: %v", len(wantAll), len(names), names)
	}
	present := make(map[string]bool, len(names))
	for _, n := range names {
		present[n] = true
	}
	for _, want := range wantAll {
		if !present[want] {
			t.Errorf("expected step %q to be registered, got %v", want, names)
		}
	}

	for _, want := range wantAll {
		step, err := NewStep(want)
		if err != nil {
			t.Fatalf("NewStep(%q): %v", want, err)
		}
		if step.Name() != want {
			t.Errorf("Name = %q, want %q", step.Name(), want)
		}
	}

	_, err := NewStep("nonexistent")
	if err == nil {
		t.Error("expected error for unknown step")
	}
}

func TestPipelineOutput_MessagesReflectFiltering(t *testing.T) {
	// Drop the first message via a custom step; ensure output.Messages
	// no longer includes it.
	dropFirst := stepFunc{name: "drop_first", fn: func(items []ContextItem) []ContextItem {
		if len(items) == 0 {
			return items
		}
		return items[1:]
	}}

	pipeline := NewPipeline(WithStep(dropFirst))
	input := PipelineInput{
		Messages: []schema.Message{
			schema.NewHumanMessage("first"),
			schema.NewAIMessage("second"),
			schema.NewHumanMessage("third"),
		},
		MaxTokens: 1000,
	}

	output, err := pipeline.Execute(gocontext.Background(), input)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(output.Messages) != 2 {
		t.Fatalf("output.Messages = %d, want 2 (first dropped)", len(output.Messages))
	}
	if got := textOf(output.Messages[0]); got != "second" {
		t.Errorf("output.Messages[0] = %q, want %q", got, "second")
	}
	if got := textOf(output.Messages[1]); got != "third" {
		t.Errorf("output.Messages[1] = %q, want %q", got, "third")
	}
}

type stepFunc struct {
	name string
	fn   func([]ContextItem) []ContextItem
}

func (s stepFunc) Name() string { return s.name }
func (s stepFunc) Process(_ gocontext.Context, items []ContextItem) ([]ContextItem, error) {
	return s.fn(items), nil
}

func textOf(m schema.Message) string {
	out := ""
	for _, p := range m.GetContent() {
		if tp, ok := p.(schema.TextPart); ok {
			out += tp.Text
		}
	}
	return out
}

func TestStepNames(t *testing.T) {
	steps := []ContextStep{
		&RelevanceRanker{},
		NewTokenBudgetFilter(100),
		&DuplicateFilter{},
		NewRecencyBooster(0.1),
	}

	expectedNames := []string{"relevance_rank", "token_budget_filter", "duplicate_filter", "recency_boost"}

	for i, step := range steps {
		if step.Name() != expectedNames[i] {
			t.Errorf("step %d Name = %q, want %q", i, step.Name(), expectedNames[i])
		}
	}
}
