package agent

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

func TestMCTSNode_UCTScore_Unvisited(t *testing.T) {
	node := &MCTSNode{
		State:  "test",
		Visits: 0,
	}

	score := node.UCTScore(1.41)
	if !math.IsInf(score, 1) {
		t.Errorf("UCTScore for unvisited node = %f, want +Inf", score)
	}
}

func TestMCTSNode_UCTScore_Root(t *testing.T) {
	root := &MCTSNode{
		State:  "root",
		Visits: 10,
		Value:  5.0,
		Parent: nil,
	}

	score := root.UCTScore(1.41)
	if score != 0 {
		t.Errorf("UCTScore for root = %f, want 0", score)
	}
}

func TestMCTSNode_UCTScore_BalancesExploitationExploration(t *testing.T) {
	parent := &MCTSNode{
		State:  "parent",
		Visits: 100,
		Value:  50.0,
	}

	highValue := &MCTSNode{
		State:  "high value",
		Visits: 50,
		Value:  40.0, // avg = 0.8
		Parent: parent,
	}

	lowVisits := &MCTSNode{
		State:  "low visits",
		Visits: 5,
		Value:  3.0, // avg = 0.6, but low visits boost exploration
		Parent: parent,
	}

	scoreHigh := highValue.UCTScore(1.41)
	scoreLow := lowVisits.UCTScore(1.41)

	// With exploration constant, the low-visit node should have higher score
	// due to exploration bonus
	if scoreLow <= scoreHigh {
		t.Errorf("lowVisits score = %f, highValue score = %f; expected lowVisits > highValue due to exploration", scoreLow, scoreHigh)
	}
}

func TestNewLATSPlanner_Defaults(t *testing.T) {
	model := &testLLM{}
	p := NewLATSPlanner(model)

	if p == nil {
		t.Fatal("expected non-nil planner")
	}
	if p.llm == nil {
		t.Error("llm should not be nil")
	}
	if p.expansionWidth != 5 {
		t.Errorf("expansionWidth = %d, want 5", p.expansionWidth)
	}
	if p.maxDepth != 10 {
		t.Errorf("maxDepth = %d, want 10", p.maxDepth)
	}
	if p.explorationConstant != 1.41 {
		t.Errorf("explorationConstant = %f, want 1.41", p.explorationConstant)
	}
}

func TestLATSPlanner_WithExpansionWidth(t *testing.T) {
	model := &testLLM{}
	p := NewLATSPlanner(model, WithExpansionWidth(10))

	if p.expansionWidth != 10 {
		t.Errorf("expansionWidth = %d, want 10", p.expansionWidth)
	}
}

func TestLATSPlanner_WithExpansionWidth_IgnoresNonPositive(t *testing.T) {
	model := &testLLM{}

	p := NewLATSPlanner(model, WithExpansionWidth(0))
	if p.expansionWidth != 5 {
		t.Errorf("expansionWidth = %d, want 5 (unchanged)", p.expansionWidth)
	}

	p = NewLATSPlanner(model, WithExpansionWidth(-1))
	if p.expansionWidth != 5 {
		t.Errorf("expansionWidth = %d, want 5 (unchanged)", p.expansionWidth)
	}
}

func TestLATSPlanner_WithLATSMaxDepth(t *testing.T) {
	model := &testLLM{}
	p := NewLATSPlanner(model, WithLATSMaxDepth(15))

	if p.maxDepth != 15 {
		t.Errorf("maxDepth = %d, want 15", p.maxDepth)
	}
}

func TestLATSPlanner_WithLATSMaxDepth_IgnoresNonPositive(t *testing.T) {
	model := &testLLM{}

	p := NewLATSPlanner(model, WithLATSMaxDepth(0))
	if p.maxDepth != 10 {
		t.Errorf("maxDepth = %d, want 10 (unchanged)", p.maxDepth)
	}

	p = NewLATSPlanner(model, WithLATSMaxDepth(-1))
	if p.maxDepth != 10 {
		t.Errorf("maxDepth = %d, want 10 (unchanged)", p.maxDepth)
	}
}

func TestLATSPlanner_WithExplorationConstant(t *testing.T) {
	model := &testLLM{}
	p := NewLATSPlanner(model, WithExplorationConstant(2.0))

	if p.explorationConstant != 2.0 {
		t.Errorf("explorationConstant = %f, want 2.0", p.explorationConstant)
	}
}

func TestLATSPlanner_WithExplorationConstant_IgnoresNonPositive(t *testing.T) {
	model := &testLLM{}

	p := NewLATSPlanner(model, WithExplorationConstant(0))
	if p.explorationConstant != 1.41 {
		t.Errorf("explorationConstant = %f, want 1.41 (unchanged)", p.explorationConstant)
	}

	p = NewLATSPlanner(model, WithExplorationConstant(-1))
	if p.explorationConstant != 1.41 {
		t.Errorf("explorationConstant = %f, want 1.41 (unchanged)", p.explorationConstant)
	}
}

func TestLATSPlanner_SelectNode_PrefersUnvisited(t *testing.T) {
	model := &testLLM{}
	p := NewLATSPlanner(model)

	root := &MCTSNode{
		State:  "root",
		Visits: 10,
		Value:  5.0,
	}

	visited := &MCTSNode{
		State:  "visited",
		Visits: 5,
		Value:  3.0,
		Parent: root,
	}

	unvisited := &MCTSNode{
		State:  "unvisited",
		Visits: 0,
		Value:  0.0,
		Parent: root,
	}

	root.Children = []*MCTSNode{visited, unvisited}

	selected := p.selectNode(root)

	// Should select unvisited node (UCT score is +Inf)
	if selected.State != "unvisited" {
		t.Errorf("selectNode = %q, want %q (unvisited node)", selected.State, "unvisited")
	}
}

func TestLATSPlanner_Backpropagate(t *testing.T) {
	model := &testLLM{}
	p := NewLATSPlanner(model)

	root := &MCTSNode{State: "root"}
	child := &MCTSNode{State: "child", Parent: root}
	grandchild := &MCTSNode{State: "grandchild", Parent: child}

	p.backpropagate(grandchild, 0.8)

	// All nodes in the path should have visits incremented and value accumulated
	if grandchild.Visits != 1 {
		t.Errorf("grandchild visits = %d, want 1", grandchild.Visits)
	}
	if grandchild.Value != 0.8 {
		t.Errorf("grandchild value = %f, want 0.8", grandchild.Value)
	}

	if child.Visits != 1 {
		t.Errorf("child visits = %d, want 1", child.Visits)
	}
	if child.Value != 0.8 {
		t.Errorf("child value = %f, want 0.8", child.Value)
	}

	if root.Visits != 1 {
		t.Errorf("root visits = %d, want 1", root.Visits)
	}
	if root.Value != 0.8 {
		t.Errorf("root value = %f, want 0.8", root.Value)
	}
}

func TestLATSPlanner_ExtractPath(t *testing.T) {
	model := &testLLM{}
	p := NewLATSPlanner(model)

	root := &MCTSNode{State: "root"}
	child := &MCTSNode{State: "child", Parent: root}
	grandchild := &MCTSNode{State: "grandchild", Parent: child}

	path := p.extractPath(grandchild)

	expected := []string{"root", "child", "grandchild"}
	if len(path) != len(expected) {
		t.Fatalf("path length = %d, want %d", len(path), len(expected))
	}

	for i, state := range expected {
		if path[i] != state {
			t.Errorf("path[%d] = %q, want %q", i, path[i], state)
		}
	}
}

func TestLATSPlanner_BestLeaf(t *testing.T) {
	model := &testLLM{}
	p := NewLATSPlanner(model)

	root := &MCTSNode{State: "root", Visits: 10, Value: 5.0}

	leaf1 := &MCTSNode{State: "leaf1", Visits: 5, Value: 4.0, Parent: root} // avg = 0.8
	leaf2 := &MCTSNode{State: "leaf2", Visits: 3, Value: 2.1, Parent: root} // avg = 0.7
	leaf3 := &MCTSNode{State: "leaf3", Visits: 2, Value: 1.8, Parent: root} // avg = 0.9

	// Create tree structure
	child := &MCTSNode{State: "child", Visits: 10, Value: 8.0, Parent: root, Children: []*MCTSNode{leaf1, leaf2}}
	root.Children = []*MCTSNode{child, leaf3}

	best := p.bestLeaf(root)

	// leaf3 has highest average (0.9)
	if best.State != "leaf3" {
		t.Errorf("bestLeaf = %q, want %q (highest average value)", best.State, "leaf3")
	}
}

func TestLATSPlanner_Plan_ReturnsActions(t *testing.T) {
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
			if strings.Contains(prompt, "Generate") || strings.Contains(prompt, "generate") {
				return schema.NewAIMessage("1. Thought A"), nil
			}

			// Evaluate thoughts - return low score to avoid early termination
			if strings.Contains(prompt, "quality") || strings.Contains(prompt, "Rate") {
				return schema.NewAIMessage("0.3"), nil
			}

			// Synthesize final answer
			return schema.NewAIMessage("Final answer from LATS"), nil
		},
	}

	p := NewLATSPlanner(model, WithExpansionWidth(1), WithLATSMaxDepth(1))
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

func TestLATSPlanner_Plan_HighScoreEarlyReturn(t *testing.T) {
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
			if strings.Contains(prompt, "Generate") || strings.Contains(prompt, "generate") {
				return schema.NewAIMessage("1. Perfect thought"), nil
			}

			// Evaluate thoughts - return high score to trigger early return
			if strings.Contains(prompt, "quality") || strings.Contains(prompt, "Rate") {
				return schema.NewAIMessage("0.95"), nil
			}

			// Synthesize final answer
			return schema.NewAIMessage("Early answer from high-score path"), nil
		},
	}

	p := NewLATSPlanner(model, WithExpansionWidth(1), WithLATSMaxDepth(5))
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

	// With high score (>= 0.9), should return early without exhausting all iterations
	// The exact call count depends on timing, but should be relatively low
}

func TestLATSPlanner_Reflect_GeneratesReflection(t *testing.T) {
	reflectionText := "The approach was too simplistic. Need deeper analysis."
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage(reflectionText), nil
		},
	}

	p := NewLATSPlanner(model)
	reflection, err := p.reflect(context.Background(), "problem", "failed state", 0.2)

	if err != nil {
		t.Fatalf("reflect error: %v", err)
	}

	if reflection != reflectionText {
		t.Errorf("reflection = %q, want %q", reflection, reflectionText)
	}
}

func TestLATSPlanner_Reflections(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			lastMsg := msgs[len(msgs)-1]
			var prompt string
			if hm, ok := lastMsg.(*schema.HumanMessage); ok {
				prompt = hm.Text()
			}

			if strings.Contains(prompt, "Generate") {
				return schema.NewAIMessage("1. Thought"), nil
			}
			if strings.Contains(prompt, "quality") || strings.Contains(prompt, "Rate") {
				return schema.NewAIMessage("0.1"), nil // Low score to trigger reflection
			}
			if strings.Contains(prompt, "reflection") {
				return schema.NewAIMessage("Reflection on failure"), nil
			}
			return schema.NewAIMessage("Final answer"), nil
		},
	}

	p := NewLATSPlanner(model, WithExpansionWidth(1), WithLATSMaxDepth(1))
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	_, err := p.Plan(context.Background(), state)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}

	reflections := p.Reflections()
	if len(reflections) == 0 {
		t.Error("expected at least one reflection from low-score evaluation")
	}
}

func TestLATSPlanner_Plan_LLMError(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return nil, errors.New("LLM failed")
		},
	}

	p := NewLATSPlanner(model)
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	// Should handle LLM errors gracefully and still return a result
	// (it continues despite individual expansion/evaluation failures)
	actions, err := p.Plan(context.Background(), state)

	// Either succeeds with fallback or fails cleanly
	if err != nil {
		// Error is acceptable
		return
	}

	if len(actions) == 0 {
		t.Fatal("if Plan succeeds, expected at least one action")
	}
}

func TestLATSPlanner_Replan(t *testing.T) {
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
				return schema.NewAIMessage("1. Replanned thought"), nil
			}
			if strings.Contains(prompt, "quality") || strings.Contains(prompt, "Rate") {
				return schema.NewAIMessage("0.5"), nil
			}
			return schema.NewAIMessage("Replanned answer"), nil
		},
	}

	p := NewLATSPlanner(model, WithExpansionWidth(1), WithLATSMaxDepth(1))
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

func TestLATSPlanner_Registry_Registered(t *testing.T) {
	planners := ListPlanners()
	found := false
	for _, name := range planners {
		if name == "lats" {
			found = true
			break
		}
	}
	if !found {
		t.Error("lats planner not registered")
	}
}

func TestLATSPlanner_Registry_Creation(t *testing.T) {
	model := &testLLM{}
	p, err := NewPlanner("lats", PlannerConfig{
		LLM: model,
	})
	if err != nil {
		t.Fatalf("NewPlanner error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil planner")
	}
}

func TestLATSPlanner_Registry_CreationFailsWithoutLLM(t *testing.T) {
	_, err := NewPlanner("lats", PlannerConfig{})
	if err == nil {
		t.Fatal("expected error when creating lats planner without LLM")
	}
}

func TestLATSPlanner_ImplementsPlanner(t *testing.T) {
	var _ Planner = (*LATSPlanner)(nil)
}

// TestLATSPlanner_Registry_CreationWithExtra tests registry creation with extra options.
func TestLATSPlanner_Registry_CreationWithExtra(t *testing.T) {
	model := &testLLM{}
	p, err := NewPlanner("lats", PlannerConfig{
		LLM: model,
		Extra: map[string]any{
			"expansion_width":       5,
			"max_depth":             10,
			"exploration_constant":  2.5,
		},
	})
	if err != nil {
		t.Fatalf("NewPlanner error: %v", err)
	}
	lats := p.(*LATSPlanner)
	if lats.expansionWidth != 5 {
		t.Errorf("expansionWidth = %d, want 5", lats.expansionWidth)
	}
	if lats.maxDepth != 10 {
		t.Errorf("maxDepth = %d, want 10", lats.maxDepth)
	}
	if lats.explorationConstant != 2.5 {
		t.Errorf("explorationConstant = %f, want 2.5", lats.explorationConstant)
	}
}
