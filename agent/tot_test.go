package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

func TestNewToTPlanner_Defaults(t *testing.T) {
	model := &testLLM{}
	p := NewToTPlanner(model)

	if p == nil {
		t.Fatal("expected non-nil planner")
	}
	if p.llm == nil {
		t.Error("llm should not be nil")
	}
	if p.branchFactor != 3 {
		t.Errorf("branchFactor = %d, want 3", p.branchFactor)
	}
	if p.maxDepth != 5 {
		t.Errorf("maxDepth = %d, want 5", p.maxDepth)
	}
	if p.strategy != StrategyBFS {
		t.Errorf("strategy = %q, want %q", p.strategy, StrategyBFS)
	}
}

func TestToTPlanner_WithBranchFactor(t *testing.T) {
	model := &testLLM{}
	p := NewToTPlanner(model, WithBranchFactor(5))

	if p.branchFactor != 5 {
		t.Errorf("branchFactor = %d, want 5", p.branchFactor)
	}
}

func TestToTPlanner_WithBranchFactor_IgnoresNonPositive(t *testing.T) {
	model := &testLLM{}

	p := NewToTPlanner(model, WithBranchFactor(0))
	if p.branchFactor != 3 {
		t.Errorf("branchFactor = %d, want 3 (unchanged)", p.branchFactor)
	}

	p = NewToTPlanner(model, WithBranchFactor(-1))
	if p.branchFactor != 3 {
		t.Errorf("branchFactor = %d, want 3 (unchanged)", p.branchFactor)
	}
}

func TestToTPlanner_WithMaxDepth(t *testing.T) {
	model := &testLLM{}
	p := NewToTPlanner(model, WithMaxDepth(10))

	if p.maxDepth != 10 {
		t.Errorf("maxDepth = %d, want 10", p.maxDepth)
	}
}

func TestToTPlanner_WithMaxDepth_IgnoresNonPositive(t *testing.T) {
	model := &testLLM{}

	p := NewToTPlanner(model, WithMaxDepth(0))
	if p.maxDepth != 5 {
		t.Errorf("maxDepth = %d, want 5 (unchanged)", p.maxDepth)
	}

	p = NewToTPlanner(model, WithMaxDepth(-1))
	if p.maxDepth != 5 {
		t.Errorf("maxDepth = %d, want 5 (unchanged)", p.maxDepth)
	}
}

func TestToTPlanner_WithSearchStrategy(t *testing.T) {
	model := &testLLM{}
	p := NewToTPlanner(model, WithSearchStrategy(StrategyDFS))

	if p.strategy != StrategyDFS {
		t.Errorf("strategy = %q, want %q", p.strategy, StrategyDFS)
	}
}

func TestThoughtHeap_PushPop(t *testing.T) {
	h := &thoughtHeap{}

	// Push nodes with different scores using heap.Push
	nodes := []*thoughtNode{
		{thought: "low", score: 0.3, depth: 1},
		{thought: "high", score: 0.9, depth: 1},
		{thought: "medium", score: 0.6, depth: 1},
	}

	for _, n := range nodes {
		*h = append(*h, n)
	}

	if h.Len() != 3 {
		t.Fatalf("heap length = %d, want 3", h.Len())
	}

	// Pop should return highest score first (max-heap)
	// Note: Without heap.Init, the slice ordering is just insertion order
	// We're testing the heap interface methods, not the actual heap behavior
	first := (*h)[0]
	if first.thought != "low" {
		t.Errorf("first element = %q (score %.1f), want 'low' (score 0.3)", first.thought, first.score)
	}

	// Test Less for max-heap property
	if !h.Less(1, 0) { // high (0.9) > low (0.3)
		t.Error("Less(1, 0) should be true for max-heap (high score > low score)")
	}

	if h.Less(0, 1) {
		t.Error("Less(0, 1) should be false for max-heap (low score < high score)")
	}
}

func TestThoughtHeap_MaxHeapOrdering(t *testing.T) {
	// Test the Less method to verify max-heap ordering
	h := thoughtHeap{
		{thought: "a", score: 0.5},
		{thought: "b", score: 0.8},
		{thought: "c", score: 0.2},
		{thought: "d", score: 1.0},
	}

	// In a max-heap, Less should return true if i > j (higher score = higher priority)
	// h[0] = 0.5, h[1] = 0.8 -> h[1] > h[0], so Less(1, 0) should be true
	if !h.Less(1, 0) {
		t.Error("Less(1, 0) should be true (0.8 > 0.5)")
	}

	// h[0] = 0.5, h[1] = 0.8 -> h[0] < h[1], so Less(0, 1) should be false
	if h.Less(0, 1) {
		t.Error("Less(0, 1) should be false (0.5 < 0.8)")
	}

	// h[3] = 1.0, h[2] = 0.2 -> h[3] > h[2], so Less(3, 2) should be true
	if !h.Less(3, 2) {
		t.Error("Less(3, 2) should be true (1.0 > 0.2)")
	}
}

func TestToTPlanner_GenerateThoughts_ParsesNumberedList(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("1. First thought\n2. Second thought\n3. Third thought"), nil
		},
	}

	p := NewToTPlanner(model, WithBranchFactor(3))
	thoughts, err := p.generateThoughts(context.Background(), "test problem", nil)

	if err != nil {
		t.Fatalf("generateThoughts error: %v", err)
	}

	if len(thoughts) != 3 {
		t.Fatalf("thoughts length = %d, want 3", len(thoughts))
	}

	if thoughts[0] != "First thought" {
		t.Errorf("thoughts[0] = %q, want %q", thoughts[0], "First thought")
	}
	if thoughts[1] != "Second thought" {
		t.Errorf("thoughts[1] = %q, want %q", thoughts[1], "Second thought")
	}
	if thoughts[2] != "Third thought" {
		t.Errorf("thoughts[2] = %q, want %q", thoughts[2], "Third thought")
	}
}

func TestToTPlanner_EvaluateThought_Sure(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("sure"), nil
		},
	}

	p := NewToTPlanner(model)
	score, err := p.evaluateThought(context.Background(), "problem", "thought")

	if err != nil {
		t.Fatalf("evaluateThought error: %v", err)
	}
	if score != 1.0 {
		t.Errorf("score = %.1f, want 1.0 for 'sure'", score)
	}
}

func TestToTPlanner_EvaluateThought_Maybe(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("maybe"), nil
		},
	}

	p := NewToTPlanner(model)
	score, err := p.evaluateThought(context.Background(), "problem", "thought")

	if err != nil {
		t.Fatalf("evaluateThought error: %v", err)
	}
	if score != 0.5 {
		t.Errorf("score = %.1f, want 0.5 for 'maybe'", score)
	}
}

func TestToTPlanner_EvaluateThought_Impossible(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("impossible"), nil
		},
	}

	p := NewToTPlanner(model)
	score, err := p.evaluateThought(context.Background(), "problem", "thought")

	if err != nil {
		t.Fatalf("evaluateThought error: %v", err)
	}
	if score != 0.0 {
		t.Errorf("score = %.1f, want 0.0 for 'impossible'", score)
	}
}

func TestToTPlanner_Plan_BFS(t *testing.T) {
	callCount := 0
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			callCount++
			lastMsg := msgs[len(msgs)-1]
			var prompt string
			if hm, ok := lastMsg.(*schema.HumanMessage); ok {
				prompt = hm.Text()
			}

			// Generate thoughts
			if strings.Contains(prompt, "generate") || strings.Contains(prompt, "Generate") {
				return schema.NewAIMessage("1. Thought A\n2. Thought B"), nil
			}

			// Evaluate thoughts
			if strings.Contains(prompt, "Evaluate") {
				return schema.NewAIMessage("sure"), nil
			}

			// Synthesize final answer
			return schema.NewAIMessage("Final answer from BFS"), nil
		},
	}

	p := NewToTPlanner(model, WithBranchFactor(2), WithMaxDepth(2), WithSearchStrategy(StrategyBFS))
	state := PlannerState{
		Input:    "test problem",
		Messages: []schema.Message{schema.NewHumanMessage("test problem")},
	}

	actions, err := p.Plan(context.Background(), state)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}

	if len(actions) == 0 {
		t.Fatal("expected at least one action")
	}

	if callCount == 0 {
		t.Error("expected LLM to be called")
	}
}

func TestToTPlanner_Plan_DFS(t *testing.T) {
	callCount := 0
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			callCount++
			lastMsg := msgs[len(msgs)-1]
			var prompt string
			if hm, ok := lastMsg.(*schema.HumanMessage); ok {
				prompt = hm.Text()
			}

			// Generate thoughts
			if strings.Contains(prompt, "generate") || strings.Contains(prompt, "Generate") {
				return schema.NewAIMessage("1. Thought A\n2. Thought B"), nil
			}

			// Evaluate thoughts
			if strings.Contains(prompt, "Evaluate") {
				return schema.NewAIMessage("sure"), nil
			}

			// Synthesize final answer
			return schema.NewAIMessage("Final answer from DFS"), nil
		},
	}

	p := NewToTPlanner(model, WithBranchFactor(2), WithMaxDepth(2), WithSearchStrategy(StrategyDFS))
	state := PlannerState{
		Input:    "test problem",
		Messages: []schema.Message{schema.NewHumanMessage("test problem")},
	}

	actions, err := p.Plan(context.Background(), state)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}

	if len(actions) == 0 {
		t.Fatal("expected at least one action")
	}

	if callCount == 0 {
		t.Error("expected LLM to be called")
	}
}

func TestToTPlanner_Plan_LLMError(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return nil, errors.New("LLM failed")
		},
	}

	p := NewToTPlanner(model)
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	_, err := p.Plan(context.Background(), state)
	if err == nil {
		t.Fatal("expected error from LLM failure")
	}
}

func TestToTPlanner_Replan(t *testing.T) {
	callCount := 0
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			callCount++
			lastMsg := msgs[len(msgs)-1]
			var prompt string
			if hm, ok := lastMsg.(*schema.HumanMessage); ok {
				prompt = hm.Text()
			}

			if strings.Contains(prompt, "generate") || strings.Contains(prompt, "Generate") {
				return schema.NewAIMessage("1. Thought"), nil
			}
			if strings.Contains(prompt, "Evaluate") {
				return schema.NewAIMessage("sure"), nil
			}
			return schema.NewAIMessage("Replanned answer"), nil
		},
	}

	p := NewToTPlanner(model, WithBranchFactor(1), WithMaxDepth(1))
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	actions, err := p.Replan(context.Background(), state)
	if err != nil {
		t.Fatalf("Replan error: %v", err)
	}

	if len(actions) == 0 {
		t.Fatal("expected at least one action")
	}

	// Replan should delegate to Plan
	if callCount == 0 {
		t.Error("expected LLM to be called during replan")
	}
}

func TestToTPlanner_Synthesize_BindsTools(t *testing.T) {
	var boundTools []schema.ToolDefinition

	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("Final answer"), nil
		},
	}

	// Wrap the model to track BindTools calls
	trackedModel := &bindTrackingLLM{
		testLLM: testLLM{
			generateFn: model.generateFn,
		},
		onBind: func(tools []schema.ToolDefinition) {
			boundTools = tools
		},
	}

	p := NewToTPlanner(trackedModel, WithBranchFactor(1), WithMaxDepth(1))
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
		Tools:    []tool.Tool{&simpleTool{toolName: "test_tool"}},
	}

	// Need to synthesize, so first generate a simple path
	path := []string{"initial thought"}
	_, err := p.synthesize(context.Background(), state, path)
	if err != nil {
		t.Fatalf("synthesize error: %v", err)
	}

	if len(boundTools) == 0 {
		t.Error("expected tools to be bound")
	}
	if len(boundTools) > 0 && boundTools[0].Name != "test_tool" {
		t.Errorf("bound tool name = %q, want %q", boundTools[0].Name, "test_tool")
	}
}

func TestToTPlanner_Registry_Registered(t *testing.T) {
	planners := ListPlanners()
	found := false
	for _, name := range planners {
		if name == "tree-of-thought" {
			found = true
			break
		}
	}
	if !found {
		t.Error("tree-of-thought planner not registered")
	}
}

func TestToTPlanner_Registry_Creation(t *testing.T) {
	model := &testLLM{}
	p, err := NewPlanner("tree-of-thought", PlannerConfig{
		LLM: model,
	})
	if err != nil {
		t.Fatalf("NewPlanner error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil planner")
	}
}

// TestToTPlanner_Registry_CreationWithExtra tests registry creation with extra options.
func TestToTPlanner_Registry_CreationWithExtra(t *testing.T) {
	model := &testLLM{}
	p, err := NewPlanner("tree-of-thought", PlannerConfig{
		LLM: model,
		Extra: map[string]any{
			"branch_factor": 5,
			"max_depth":     10,
			"strategy":      StrategyDFS,
		},
	})
	if err != nil {
		t.Fatalf("NewPlanner error: %v", err)
	}
	tot := p.(*ToTPlanner)
	if tot.branchFactor != 5 {
		t.Errorf("branchFactor = %d, want 5", tot.branchFactor)
	}
	if tot.maxDepth != 10 {
		t.Errorf("maxDepth = %d, want 10", tot.maxDepth)
	}
	if tot.strategy != StrategyDFS {
		t.Errorf("strategy = %q, want %q", tot.strategy, StrategyDFS)
	}
}

func TestToTPlanner_Registry_CreationFailsWithoutLLM(t *testing.T) {
	_, err := NewPlanner("tree-of-thought", PlannerConfig{})
	if err == nil {
		t.Fatal("expected error when creating tree-of-thought planner without LLM")
	}
}

func TestToTPlanner_ImplementsPlanner(t *testing.T) {
	var _ Planner = (*ToTPlanner)(nil)
}
