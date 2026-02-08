package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

func TestNewThoughtGraph(t *testing.T) {
	g := NewThoughtGraph()

	if g == nil {
		t.Fatal("expected non-nil graph")
	}
	if g.Nodes == nil {
		t.Fatal("Nodes map should not be nil")
	}
	if len(g.Nodes) != 0 {
		t.Errorf("new graph should be empty, got %d nodes", len(g.Nodes))
	}
}

func TestThoughtGraph_AddNode_ReturnsUniqueIDs(t *testing.T) {
	g := NewThoughtGraph()

	id1 := g.AddNode("thought 1", nil)
	id2 := g.AddNode("thought 2", nil)
	id3 := g.AddNode("thought 3", nil)

	if id1 == id2 || id2 == id3 || id1 == id3 {
		t.Errorf("expected unique IDs, got: %q, %q, %q", id1, id2, id3)
	}

	if len(g.Nodes) != 3 {
		t.Errorf("graph should have 3 nodes, got %d", len(g.Nodes))
	}
}

func TestThoughtGraph_AddNode_UpdatesParentChildren(t *testing.T) {
	g := NewThoughtGraph()

	parentID := g.AddNode("parent", nil)
	childID := g.AddNode("child", []string{parentID})

	parent, ok := g.GetNode(parentID)
	if !ok {
		t.Fatal("parent node not found")
	}

	if len(parent.Children) != 1 {
		t.Fatalf("parent should have 1 child, got %d", len(parent.Children))
	}
	if parent.Children[0] != childID {
		t.Errorf("parent's child = %q, want %q", parent.Children[0], childID)
	}

	child, ok := g.GetNode(childID)
	if !ok {
		t.Fatal("child node not found")
	}
	if len(child.Parents) != 1 {
		t.Fatalf("child should have 1 parent, got %d", len(child.Parents))
	}
	if child.Parents[0] != parentID {
		t.Errorf("child's parent = %q, want %q", child.Parents[0], parentID)
	}
}

func TestThoughtGraph_GetNode_Existing(t *testing.T) {
	g := NewThoughtGraph()
	id := g.AddNode("test content", nil)

	node, ok := g.GetNode(id)
	if !ok {
		t.Fatal("GetNode should return true for existing node")
	}
	if node.Content != "test content" {
		t.Errorf("node content = %q, want %q", node.Content, "test content")
	}
}

func TestThoughtGraph_GetNode_NonExisting(t *testing.T) {
	g := NewThoughtGraph()

	_, ok := g.GetNode("nonexistent")
	if ok {
		t.Error("GetNode should return false for non-existing node")
	}
}

func TestThoughtGraph_LeafNodes(t *testing.T) {
	g := NewThoughtGraph()

	root := g.AddNode("root", nil)
	child1 := g.AddNode("child1", []string{root})
	child2 := g.AddNode("child2", []string{root})
	g.AddNode("grandchild", []string{child1})

	leaves := g.LeafNodes()

	// child2 and grandchild are leaves (no children)
	if len(leaves) != 2 {
		t.Fatalf("expected 2 leaf nodes, got %d", len(leaves))
	}

	leafIDs := make(map[string]bool)
	for _, leaf := range leaves {
		leafIDs[leaf.ID] = true
	}

	if !leafIDs[child2] {
		t.Error("child2 should be a leaf node")
	}
	if !leafIDs["thought_4"] { // grandchild is the 4th node
		t.Error("grandchild should be a leaf node")
	}
}

func TestNewDefaultController_Defaults(t *testing.T) {
	c := NewDefaultController()

	if c.generateCount != 3 {
		t.Errorf("generateCount = %d, want 3", c.generateCount)
	}
	if !c.mergeEnabled {
		t.Error("mergeEnabled should be true by default")
	}
	if c.maxIterations != 3 {
		t.Errorf("maxIterations = %d, want 3", c.maxIterations)
	}
}

func TestDefaultController_WithGenerateCount(t *testing.T) {
	c := NewDefaultController(WithGenerateCount(5))

	if c.generateCount != 5 {
		t.Errorf("generateCount = %d, want 5", c.generateCount)
	}
}

func TestDefaultController_WithGenerateCount_IgnoresNonPositive(t *testing.T) {
	c := NewDefaultController(WithGenerateCount(0))
	if c.generateCount != 3 {
		t.Errorf("generateCount = %d, want 3 (unchanged)", c.generateCount)
	}

	c = NewDefaultController(WithGenerateCount(-1))
	if c.generateCount != 3 {
		t.Errorf("generateCount = %d, want 3 (unchanged)", c.generateCount)
	}
}

func TestDefaultController_WithMergeEnabled(t *testing.T) {
	c := NewDefaultController(WithMergeEnabled(false))

	if c.mergeEnabled {
		t.Error("mergeEnabled should be false")
	}
}

func TestDefaultController_WithControllerMaxIterations(t *testing.T) {
	c := NewDefaultController(WithControllerMaxIterations(5))

	if c.maxIterations != 5 {
		t.Errorf("maxIterations = %d, want 5", c.maxIterations)
	}
}

func TestDefaultController_WithControllerMaxIterations_IgnoresNonPositive(t *testing.T) {
	c := NewDefaultController(WithControllerMaxIterations(0))
	if c.maxIterations != 3 {
		t.Errorf("maxIterations = %d, want 3 (unchanged)", c.maxIterations)
	}
}

func TestDefaultController_NextOperation_Sequence(t *testing.T) {
	c := NewDefaultController(WithControllerMaxIterations(2), WithMergeEnabled(true))
	g := NewThoughtGraph()

	ctx := context.Background()

	// Iteration 1: should generate
	op1, err := c.NextOperation(ctx, g)
	if err != nil {
		t.Fatalf("NextOperation error: %v", err)
	}
	if op1.Type != OpGenerate {
		t.Errorf("iteration 1: operation type = %q, want %q", op1.Type, OpGenerate)
	}

	// Add some nodes to simulate generation
	g.AddNode("input", nil)
	g.AddNode("thought1", []string{"thought_1"})
	g.AddNode("thought2", []string{"thought_1"})

	// Iteration 2: should still generate
	op2, err := c.NextOperation(ctx, g)
	if err != nil {
		t.Fatalf("NextOperation error: %v", err)
	}
	if op2.Type != OpGenerate {
		t.Errorf("iteration 2: operation type = %q, want %q", op2.Type, OpGenerate)
	}

	// Iteration 3: should merge (iteration == maxIterations + 1)
	op3, err := c.NextOperation(ctx, g)
	if err != nil {
		t.Fatalf("NextOperation error: %v", err)
	}
	if op3.Type != OpMerge {
		t.Errorf("iteration 3: operation type = %q, want %q", op3.Type, OpMerge)
	}

	// Iteration 4: should aggregate (there are still leaves)
	op4, err := c.NextOperation(ctx, g)
	if err != nil {
		t.Fatalf("NextOperation error: %v", err)
	}
	if op4.Type != OpAggregate {
		t.Errorf("iteration 4: operation type = %q, want %q", op4.Type, OpAggregate)
	}

	// Note: Controller continues to return aggregate as long as there are leaves.
	// It only returns nil when there are no leaves left.
}

func TestNewGoTPlanner_Defaults(t *testing.T) {
	model := &testLLM{}
	p := NewGoTPlanner(model)

	if p == nil {
		t.Fatal("expected non-nil planner")
	}
	if p.llm == nil {
		t.Error("llm should not be nil")
	}
	if p.controller == nil {
		t.Error("controller should not be nil (should use default)")
	}
	if p.maxOperations != 10 {
		t.Errorf("maxOperations = %d, want 10", p.maxOperations)
	}
}

func TestGoTPlanner_WithController(t *testing.T) {
	model := &testLLM{}
	customController := NewDefaultController(WithGenerateCount(7))
	p := NewGoTPlanner(model, WithController(customController))

	if p.controller != customController {
		t.Error("controller should be the custom controller")
	}
}

func TestGoTPlanner_WithMaxOperations(t *testing.T) {
	model := &testLLM{}
	p := NewGoTPlanner(model, WithMaxOperations(20))

	if p.maxOperations != 20 {
		t.Errorf("maxOperations = %d, want 20", p.maxOperations)
	}
}

func TestGoTPlanner_WithMaxOperations_IgnoresNonPositive(t *testing.T) {
	model := &testLLM{}

	p := NewGoTPlanner(model, WithMaxOperations(0))
	if p.maxOperations != 10 {
		t.Errorf("maxOperations = %d, want 10 (unchanged)", p.maxOperations)
	}

	p = NewGoTPlanner(model, WithMaxOperations(-1))
	if p.maxOperations != 10 {
		t.Errorf("maxOperations = %d, want 10 (unchanged)", p.maxOperations)
	}
}

func TestGoTPlanner_Plan_ExecutesOperations(t *testing.T) {
	callCount := 0
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			callCount++
			return schema.NewAIMessage("1. Generated thought\n2. Another thought"), nil
		},
	}

	// Use a simple controller that generates once then aggregates
	ctrl := &funcController{
		nextFn: func(g *ThoughtGraph) (*Operation, error) {
			leaves := g.LeafNodes()
			if len(leaves) == 1 {
				// First call, generate
				return &Operation{Type: OpGenerate, Args: map[string]any{"count": 2}}, nil
			} else if len(leaves) > 1 {
				// Second call, aggregate
				var ids []string
				for _, leaf := range leaves {
					ids = append(ids, leaf.ID)
				}
				return &Operation{Type: OpAggregate, NodeIDs: ids}, nil
			}
			return nil, nil
		},
	}

	p := NewGoTPlanner(model, WithController(ctrl))
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

func TestGoTPlanner_Plan_LLMError(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return nil, errors.New("LLM failed")
		},
	}

	p := NewGoTPlanner(model)
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	_, err := p.Plan(context.Background(), state)
	if err == nil {
		t.Fatal("expected error from LLM failure")
	}
}

func TestParseNumberedList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxItems int
		want     []string
	}{
		{
			name:     "simple numbered list",
			input:    "1. First\n2. Second\n3. Third",
			maxItems: 10,
			want:     []string{"First", "Second", "Third"},
		},
		{
			name:     "with extra whitespace",
			input:    "  1. First  \n  2. Second  ",
			maxItems: 10,
			want:     []string{"First", "Second"},
		},
		{
			name:     "respects max items",
			input:    "1. First\n2. Second\n3. Third\n4. Fourth",
			maxItems: 2,
			want:     []string{"First", "Second"},
		},
		{
			name:     "handles empty lines",
			input:    "1. First\n\n2. Second\n\n",
			maxItems: 10,
			want:     []string{"First", "Second"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseNumberedList(tt.input, tt.maxItems)
			if len(got) != len(tt.want) {
				t.Fatalf("length = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("item %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGoTPlanner_ExecuteOperation_OpMerge(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			lastMsg := msgs[len(msgs)-1]
			var prompt string
			if hm, ok := lastMsg.(*schema.HumanMessage); ok {
				prompt = hm.Text()
			}
			if strings.Contains(prompt, "Synthesize") {
				return schema.NewAIMessage("Merged thought"), nil
			}
			return schema.NewAIMessage("default"), nil
		},
	}

	p := NewGoTPlanner(model)
	g := NewThoughtGraph()

	node1 := g.AddNode("thought 1", nil)
	node2 := g.AddNode("thought 2", nil)

	op := &Operation{
		Type:    OpMerge,
		NodeIDs: []string{node1, node2},
	}

	err := p.executeOperation(context.Background(), "test problem", g, op)
	if err != nil {
		t.Fatalf("executeOperation error: %v", err)
	}

	// Should have added a merged node
	if len(g.Nodes) != 3 {
		t.Errorf("expected 3 nodes after merge, got %d", len(g.Nodes))
	}
}

func TestGoTPlanner_ExecuteOperation_OpSplit(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("1. Sub-thought A\n2. Sub-thought B"), nil
		},
	}

	p := NewGoTPlanner(model)
	g := NewThoughtGraph()

	nodeID := g.AddNode("parent thought", nil)

	op := &Operation{
		Type:    OpSplit,
		NodeIDs: []string{nodeID},
	}

	err := p.executeOperation(context.Background(), "test problem", g, op)
	if err != nil {
		t.Fatalf("executeOperation error: %v", err)
	}

	// Should have split into sub-thoughts
	if len(g.Nodes) < 2 {
		t.Errorf("expected at least 2 nodes after split, got %d", len(g.Nodes))
	}
}

func TestGoTPlanner_ExecuteOperation_OpLoop(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("Refined thought"), nil
		},
	}

	p := NewGoTPlanner(model)
	g := NewThoughtGraph()

	nodeID := g.AddNode("original thought", nil)

	op := &Operation{
		Type:    OpLoop,
		NodeIDs: []string{nodeID},
	}

	err := p.executeOperation(context.Background(), "test problem", g, op)
	if err != nil {
		t.Fatalf("executeOperation error: %v", err)
	}

	// Should have added a refined node
	if len(g.Nodes) != 2 {
		t.Errorf("expected 2 nodes after loop refinement, got %d", len(g.Nodes))
	}
}

func TestGoTPlanner_ExecuteOperation_OpAggregate(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("Aggregated result"), nil
		},
	}

	p := NewGoTPlanner(model)
	g := NewThoughtGraph()

	node1 := g.AddNode("thought 1", nil)
	node2 := g.AddNode("thought 2", nil)

	op := &Operation{
		Type:    OpAggregate,
		NodeIDs: []string{node1, node2},
	}

	err := p.executeOperation(context.Background(), "test problem", g, op)
	if err != nil {
		t.Fatalf("executeOperation error: %v", err)
	}

	// Should have added an aggregated node
	if len(g.Nodes) != 3 {
		t.Errorf("expected 3 nodes after aggregate, got %d", len(g.Nodes))
	}
}

func TestGoTPlanner_Replan(t *testing.T) {
	callCount := 0
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			callCount++
			return schema.NewAIMessage("1. Thought"), nil
		},
	}

	p := NewGoTPlanner(model)
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

func TestGoTPlanner_Registry_Registered(t *testing.T) {
	planners := ListPlanners()
	found := false
	for _, name := range planners {
		if name == "graph-of-thought" {
			found = true
			break
		}
	}
	if !found {
		t.Error("graph-of-thought planner not registered")
	}
}

func TestGoTPlanner_Registry_Creation(t *testing.T) {
	model := &testLLM{}
	p, err := NewPlanner("graph-of-thought", PlannerConfig{
		LLM: model,
	})
	if err != nil {
		t.Fatalf("NewPlanner error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil planner")
	}
}

func TestGoTPlanner_ImplementsPlanner(t *testing.T) {
	var _ Planner = (*GoTPlanner)(nil)
}

func TestController_ImplementedByDefaultController(t *testing.T) {
	var _ Controller = (*DefaultController)(nil)
}

// funcController is a test controller that uses a function
type funcController struct {
	nextFn func(g *ThoughtGraph) (*Operation, error)
}

func (c *funcController) NextOperation(ctx context.Context, graph *ThoughtGraph) (*Operation, error) {
	if c.nextFn != nil {
		return c.nextFn(graph)
	}
	return nil, nil
}
