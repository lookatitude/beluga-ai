package agent

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

func TestNewReflexionPlanner_Defaults(t *testing.T) {
	model := &testLLM{}
	p := NewReflexionPlanner(model)

	if p.threshold != 0.7 {
		t.Errorf("threshold = %f, want 0.7", p.threshold)
	}
	if p.maxReflections != 3 {
		t.Errorf("maxReflections = %d, want 3", p.maxReflections)
	}
	if p.evaluator == nil {
		t.Error("evaluator should default to actor")
	}
}

func TestNewReflexionPlanner_WithOptions(t *testing.T) {
	actor := &testLLM{}
	evaluator := &testLLM{}
	p := NewReflexionPlanner(actor,
		WithEvaluator(evaluator),
		WithThreshold(0.9),
		WithMaxReflections(5),
	)

	if p.threshold != 0.9 {
		t.Errorf("threshold = %f, want 0.9", p.threshold)
	}
	if p.maxReflections != 5 {
		t.Errorf("maxReflections = %d, want 5", p.maxReflections)
	}
}

func TestWithMaxReflections_IgnoresNonPositive(t *testing.T) {
	actor := &testLLM{}
	p := NewReflexionPlanner(actor, WithMaxReflections(0))
	if p.maxReflections != 3 {
		t.Errorf("maxReflections = %d, want 3 (default)", p.maxReflections)
	}

	p = NewReflexionPlanner(actor, WithMaxReflections(-1))
	if p.maxReflections != 3 {
		t.Errorf("maxReflections = %d, want 3 (default)", p.maxReflections)
	}
}

func TestReflexionPlanner_Plan_HighScore(t *testing.T) {
	// Actor returns a response, evaluator scores it highly.
	actor := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("great answer"), nil
		},
	}
	evaluator := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("0.9"), nil
		},
	}

	p := NewReflexionPlanner(actor, WithEvaluator(evaluator), WithThreshold(0.7))
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	actions, err := p.Plan(context.Background(), state)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Type != ActionFinish {
		t.Errorf("action type = %q, want %q", actions[0].Type, ActionFinish)
	}
	if actions[0].Message != "great answer" {
		t.Errorf("message = %q, want %q", actions[0].Message, "great answer")
	}
}

func TestReflexionPlanner_Plan_LowScore_TriggersReflection(t *testing.T) {
	callCount := 0
	actor := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			callCount++
			if callCount == 1 {
				return schema.NewAIMessage("weak answer"), nil
			}
			return schema.NewAIMessage("improved answer"), nil
		},
	}

	evalCallCount := 0
	evaluator := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			evalCallCount++
			if evalCallCount == 1 {
				// Evaluation: low score.
				return schema.NewAIMessage("0.3"), nil
			}
			// Reflection feedback.
			return schema.NewAIMessage("Improve clarity and add examples"), nil
		},
	}

	p := NewReflexionPlanner(actor, WithEvaluator(evaluator), WithThreshold(0.7))
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	actions, err := p.Plan(context.Background(), state)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	// After reflection, it retries and should return the improved answer.
	if actions[0].Type != ActionFinish {
		t.Errorf("action type = %q, want %q", actions[0].Type, ActionFinish)
	}
	if actions[0].Message != "improved answer" {
		t.Errorf("message = %q, want %q", actions[0].Message, "improved answer")
	}

	// Should have at least 1 reflection recorded.
	if len(p.Reflections()) == 0 {
		t.Error("expected at least 1 reflection")
	}
}

func TestReflexionPlanner_Plan_ToolCallResponse_SkipsEvaluation(t *testing.T) {
	actor := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return &schema.AIMessage{
				ToolCalls: []schema.ToolCall{
					{ID: "call-1", Name: "search", Arguments: `{}`},
				},
			}, nil
		},
	}

	evaluatorCalled := false
	evaluator := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			evaluatorCalled = true
			return schema.NewAIMessage("0.9"), nil
		},
	}

	p := NewReflexionPlanner(actor, WithEvaluator(evaluator))
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
		Tools:    []tool.Tool{&simpleTool{toolName: "search"}},
	}

	actions, err := p.Plan(context.Background(), state)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(actions) != 1 || actions[0].Type != ActionTool {
		t.Errorf("expected tool action, got %v", actions)
	}
	if evaluatorCalled {
		t.Error("evaluator should not be called for tool call responses")
	}
}

func TestReflexionPlanner_Plan_ActorError(t *testing.T) {
	actor := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return nil, errors.New("actor failed")
		},
	}

	p := NewReflexionPlanner(actor)
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	_, err := p.Plan(context.Background(), state)
	if err == nil {
		t.Fatal("expected error from actor")
	}
}

func TestReflexionPlanner_Plan_EvaluationError_AcceptsResponse(t *testing.T) {
	actor := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("response"), nil
		},
	}
	evaluator := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return nil, errors.New("eval failed")
		},
	}

	p := NewReflexionPlanner(actor, WithEvaluator(evaluator))
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	actions, err := p.Plan(context.Background(), state)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	// Should accept the response when evaluation fails.
	if len(actions) != 1 || actions[0].Type != ActionFinish {
		t.Errorf("expected finish action, got %v", actions)
	}
}

func TestReflexionPlanner_Plan_MaxReflections(t *testing.T) {
	actor := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("always weak"), nil
		},
	}

	evalCount := 0
	evaluator := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			evalCount++
			// All eval calls: alternately return low score and reflection.
			if evalCount%2 == 1 {
				return schema.NewAIMessage("0.2"), nil
			}
			return schema.NewAIMessage("needs improvement"), nil
		},
	}

	p := NewReflexionPlanner(actor, WithEvaluator(evaluator), WithMaxReflections(1))

	// First plan: triggers 1 reflection.
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}
	_, _ = p.Plan(context.Background(), state)

	// Second plan: max reflections reached, should accept response.
	actions, err := p.Plan(context.Background(), state)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(actions) == 0 {
		t.Fatal("expected at least 1 action")
	}
	// With max reflections reached, it should still return something.
	if actions[0].Type != ActionFinish {
		t.Errorf("expected finish action after max reflections, got %q", actions[0].Type)
	}
}

func TestReflexionPlanner_Reflections(t *testing.T) {
	p := NewReflexionPlanner(&testLLM{})
	if len(p.Reflections()) != 0 {
		t.Error("expected empty reflections initially")
	}

	p.reflections = []string{"reflection 1", "reflection 2"}
	refs := p.Reflections()
	if len(refs) != 2 {
		t.Fatalf("expected 2 reflections, got %d", len(refs))
	}
}

func TestReflexionPlanner_ResetReflections(t *testing.T) {
	p := NewReflexionPlanner(&testLLM{})
	p.reflections = []string{"a", "b"}
	p.ResetReflections()

	if len(p.Reflections()) != 0 {
		t.Errorf("expected empty reflections after reset, got %d", len(p.Reflections()))
	}
}

func TestReflexionPlanner_Evaluate_ParsesScore(t *testing.T) {
	evaluator := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("0.85"), nil
		},
	}

	p := NewReflexionPlanner(&testLLM{}, WithEvaluator(evaluator))
	score, err := p.evaluate(context.Background(), "input", "response")
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	if score != 0.85 {
		t.Errorf("score = %f, want 0.85", score)
	}
}

func TestReflexionPlanner_Evaluate_InvalidScore(t *testing.T) {
	evaluator := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("not a number"), nil
		},
	}

	p := NewReflexionPlanner(&testLLM{}, WithEvaluator(evaluator))
	score, err := p.evaluate(context.Background(), "input", "response")
	if err != nil {
		t.Fatalf("evaluate error: %v", err)
	}
	// Should default to 0.5 for unparseable scores.
	if score != 0.5 {
		t.Errorf("score = %f, want 0.5 (default)", score)
	}
}

func TestReflexionPlanner_Evaluate_ClampsScore(t *testing.T) {
	tests := []struct {
		name  string
		score string
		want  float64
	}{
		{name: "above 1", score: "1.5", want: 1.0},
		{name: "below 0", score: "-0.5", want: 0.0},
		{name: "normal", score: "0.75", want: 0.75},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evaluator := &testLLM{
				generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
					return schema.NewAIMessage(tt.score), nil
				},
			}
			p := NewReflexionPlanner(&testLLM{}, WithEvaluator(evaluator))
			score, err := p.evaluate(context.Background(), "input", "response")
			if err != nil {
				t.Fatalf("evaluate error: %v", err)
			}
			if score != tt.want {
				t.Errorf("score = %f, want %f", score, tt.want)
			}
		})
	}
}

func TestReflexionPlanner_Reflect(t *testing.T) {
	evaluator := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("Add more detail and examples"), nil
		},
	}

	p := NewReflexionPlanner(&testLLM{}, WithEvaluator(evaluator))
	reflection, err := p.reflect(context.Background(), "input", "response", 0.3)
	if err != nil {
		t.Fatalf("reflect error: %v", err)
	}
	if reflection != "Add more detail and examples" {
		t.Errorf("reflection = %q, want %q", reflection, "Add more detail and examples")
	}
}

func TestReflexionPlanner_RegisteredInRegistry(t *testing.T) {
	names := ListPlanners()
	found := false
	for _, name := range names {
		if name == "reflexion" {
			found = true
			break
		}
	}
	if !found {
		t.Error("reflexion planner not found in registry")
	}
}

func TestReflexionPlanner_CreateFromRegistry(t *testing.T) {
	model := &testLLM{}
	p, err := NewPlanner("reflexion", PlannerConfig{LLM: model})
	if err != nil {
		t.Fatalf("NewPlanner error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil planner")
	}
}

func TestReflexionPlanner_CreateFromRegistry_NoLLM(t *testing.T) {
	_, err := NewPlanner("reflexion", PlannerConfig{})
	if err == nil {
		t.Fatal("expected error when no LLM provided")
	}
}

func TestReflexionPlanner_CreateFromRegistry_WithExtra(t *testing.T) {
	model := &testLLM{}
	p, err := NewPlanner("reflexion", PlannerConfig{
		LLM: model,
		Extra: map[string]any{
			"threshold":       0.9,
			"max_reflections": 5,
		},
	})
	if err != nil {
		t.Fatalf("NewPlanner error: %v", err)
	}
	rp := p.(*ReflexionPlanner)
	if rp.threshold != 0.9 {
		t.Errorf("threshold = %f, want 0.9", rp.threshold)
	}
	if rp.maxReflections != 5 {
		t.Errorf("maxReflections = %d, want 5", rp.maxReflections)
	}
}

func TestReflexionPlanner_ImplementsPlanner(t *testing.T) {
	var _ Planner = (*ReflexionPlanner)(nil)
}

// TestReflexionPlanner_Replan tests Replan() specifically.
func TestReflexionPlanner_Replan(t *testing.T) {
	actor := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("replanned response"), nil
		},
	}
	evaluator := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("0.9"), nil
		},
	}

	p := NewReflexionPlanner(actor, WithEvaluator(evaluator))
	state := PlannerState{
		Input:    "replan test",
		Messages: []schema.Message{schema.NewHumanMessage("replan test")},
	}

	actions, err := p.Replan(context.Background(), state)
	if err != nil {
		t.Fatalf("Replan error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Message != "replanned response" {
		t.Errorf("message = %q, want %q", actions[0].Message, "replanned response")
	}
}

// reflexionTestLLM is a ChatModel for reflexion tests that allows tracking
// separate actor and evaluator calls.
type reflexionTestLLM struct {
	generateFn func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error)
}

func (m *reflexionTestLLM) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	return m.generateFn(ctx, msgs)
}

func (m *reflexionTestLLM) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return func(yield func(schema.StreamChunk, error) bool) {}
}

func (m *reflexionTestLLM) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
	return &reflexionTestLLM{generateFn: m.generateFn}
}

func (m *reflexionTestLLM) ModelID() string { return "reflexion-test" }

var _ llm.ChatModel = (*reflexionTestLLM)(nil)
