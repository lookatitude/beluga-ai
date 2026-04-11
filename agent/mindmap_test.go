package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

func TestMindMapPlanner_CompileCheck(t *testing.T) {
	var _ Planner = (*MindMapPlanner)(nil)
}

func TestNewMindMapPlanner_Defaults(t *testing.T) {
	model := &testLLM{}
	p := NewMindMapPlanner(model)

	if p.maxNodes != 20 {
		t.Errorf("default maxNodes = %d, want 20", p.maxNodes)
	}
	if p.coherenceThreshold != 0.5 {
		t.Errorf("default coherenceThreshold = %f, want 0.5", p.coherenceThreshold)
	}
	if p.graph == nil {
		t.Error("graph should be initialized")
	}
}

func TestNewMindMapPlanner_Options(t *testing.T) {
	model := &testLLM{}
	p := NewMindMapPlanner(model,
		WithMaxNodes(50),
		WithCoherenceThreshold(0.7),
	)

	if p.maxNodes != 50 {
		t.Errorf("maxNodes = %d, want 50", p.maxNodes)
	}
	if p.coherenceThreshold != 0.7 {
		t.Errorf("coherenceThreshold = %f, want 0.7", p.coherenceThreshold)
	}
}

func TestWithMaxNodes_Invalid(t *testing.T) {
	model := &testLLM{}
	p := NewMindMapPlanner(model, WithMaxNodes(-5))
	if p.maxNodes != 20 {
		t.Errorf("negative maxNodes should keep default, got %d", p.maxNodes)
	}

	p = NewMindMapPlanner(model, WithMaxNodes(0))
	if p.maxNodes != 20 {
		t.Errorf("zero maxNodes should keep default, got %d", p.maxNodes)
	}
}

func TestWithCoherenceThreshold_Clamp(t *testing.T) {
	model := &testLLM{}
	p := NewMindMapPlanner(model, WithCoherenceThreshold(1.5))
	if p.coherenceThreshold != 1.0 {
		t.Errorf("threshold should be clamped to 1.0, got %f", p.coherenceThreshold)
	}

	p = NewMindMapPlanner(model, WithCoherenceThreshold(-0.5))
	if p.coherenceThreshold != 0.0 {
		t.Errorf("threshold should be clamped to 0.0, got %f", p.coherenceThreshold)
	}
}

func TestMindMapPlanner_Plan(t *testing.T) {
	model := &testLLM{
		generateFn: func(_ context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			// Check if this is the populate call (system message about extracting reasoning).
			for _, m := range msgs {
				if sm, ok := m.(*schema.SystemMessage); ok {
					if strings.Contains(sm.Text(), "Extract structured reasoning") {
						return schema.NewAIMessage(
							"claim|The earth is round|0.9|0|\n" +
								"evidence|Satellite photos|0.8|1|supports\n" +
								"question|What about flat earth claims?|0.3|1|contradicts\n",
						), nil
					}
				}
			}
			// Synthesis call.
			return schema.NewAIMessage("The earth is definitely round based on evidence."), nil
		},
	}

	p := NewMindMapPlanner(model)
	actions, err := p.Plan(context.Background(), PlannerState{
		Input:    "Is the earth round?",
		Messages: []schema.Message{schema.NewHumanMessage("Is the earth round?")},
	})

	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(actions) == 0 {
		t.Fatal("expected at least one action")
	}
	if actions[0].Type != ActionFinish {
		t.Errorf("expected ActionFinish, got %s", actions[0].Type)
	}

	// Verify graph was populated.
	g := p.Graph()
	if g.NodeCount() == 0 {
		t.Error("graph should have nodes after Plan")
	}
}

func TestMindMapPlanner_Plan_LLMError(t *testing.T) {
	model := &testLLM{
		generateFn: func(_ context.Context, _ []schema.Message) (*schema.AIMessage, error) {
			return nil, errors.New("LLM unavailable")
		},
	}

	p := NewMindMapPlanner(model)
	_, err := p.Plan(context.Background(), PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	})

	if err == nil {
		t.Fatal("expected error from LLM failure")
	}
	if !strings.Contains(err.Error(), "mindmap plan") {
		t.Errorf("error should be wrapped with mindmap plan context, got: %v", err)
	}
}

func TestMindMapPlanner_Replan(t *testing.T) {
	callCount := 0
	model := &testLLM{
		generateFn: func(_ context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			callCount++
			for _, m := range msgs {
				if sm, ok := m.(*schema.SystemMessage); ok {
					if strings.Contains(sm.Text(), "Extract structured reasoning") {
						return schema.NewAIMessage("claim|Updated conclusion|0.9|0|\n"), nil
					}
				}
			}
			return schema.NewAIMessage("Updated answer"), nil
		},
	}

	p := NewMindMapPlanner(model)

	// Seed the graph via Plan first.
	_, err := p.Plan(context.Background(), PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	})
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}

	// Replan with observations.
	actions, err := p.Replan(context.Background(), PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
		Observations: []Observation{
			{
				Action: Action{Type: ActionTool},
				Result: tool.TextResult("tool returned useful data"),
			},
		},
		Iteration: 1,
	})

	if err != nil {
		t.Fatalf("Replan error: %v", err)
	}
	if len(actions) == 0 {
		t.Fatal("expected at least one action from Replan")
	}

	// Graph should have more nodes after replan (observation + new reasoning).
	g := p.Graph()
	if g.NodeCount() < 2 {
		t.Errorf("expected at least 2 nodes after replan, got %d", g.NodeCount())
	}
}

func TestMindMapPlanner_Replan_CoherenceCheck(t *testing.T) {
	callIdx := 0
	model := &testLLM{
		generateFn: func(_ context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			callIdx++
			for _, m := range msgs {
				if sm, ok := m.(*schema.SystemMessage); ok {
					if strings.Contains(sm.Text(), "Resolve contradictions") {
						return schema.NewAIMessage("CONCLUSION|Both views have merit|0.9\n"), nil
					}
					if strings.Contains(sm.Text(), "Extract structured reasoning") {
						return schema.NewAIMessage(
							"claim|A is true|0.9|0|\n" +
								"claim|A is false|0.8|1|contradicts\n",
						), nil
					}
				}
			}
			return schema.NewAIMessage("answer"), nil
		},
	}

	// Use a very high coherence threshold to trigger contradiction resolution.
	p := NewMindMapPlanner(model, WithCoherenceThreshold(0.99))

	_, err := p.Plan(context.Background(), PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	})
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}

	// Replan should detect low coherence and try to resolve.
	_, err = p.Replan(context.Background(), PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	})
	if err != nil {
		t.Fatalf("Replan error: %v", err)
	}
}

func TestMindMapPlanner_Plan_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	model := &testLLM{
		generateFn: func(ctx context.Context, _ []schema.Message) (*schema.AIMessage, error) {
			return nil, ctx.Err()
		},
	}

	p := NewMindMapPlanner(model)
	_, err := p.Plan(ctx, PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	})

	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestMindMapPlanner_MaxNodes(t *testing.T) {
	model := &testLLM{
		generateFn: func(_ context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			for _, m := range msgs {
				if sm, ok := m.(*schema.SystemMessage); ok {
					if strings.Contains(sm.Text(), "Extract structured reasoning") {
						// Try to add many nodes.
						var lines []string
						for i := 0; i < 20; i++ {
							lines = append(lines, "claim|node content|0.5|0|")
						}
						return schema.NewAIMessage(strings.Join(lines, "\n")), nil
					}
				}
			}
			return schema.NewAIMessage("answer"), nil
		},
	}

	p := NewMindMapPlanner(model, WithMaxNodes(5))
	_, err := p.Plan(context.Background(), PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	})
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}

	if p.Graph().NodeCount() > 5 {
		t.Errorf("expected at most 5 nodes, got %d", p.Graph().NodeCount())
	}
}

func TestMindMapPlanner_Registry(t *testing.T) {
	planners := ListPlanners()
	found := false
	for _, name := range planners {
		if name == "mindmap" {
			found = true
			break
		}
	}
	if !found {
		t.Error("mindmap planner not found in registry")
	}
}

func TestMindMapPlanner_RegistryCreation(t *testing.T) {
	model := &testLLM{}

	// Should fail without LLM.
	_, err := NewPlanner("mindmap", PlannerConfig{})
	if err == nil {
		t.Error("expected error without LLM")
	}

	// Should succeed with LLM.
	p, err := NewPlanner("mindmap", PlannerConfig{
		LLM: model,
		Extra: map[string]any{
			"max_nodes":           10,
			"coherence_threshold": 0.6,
		},
	})
	if err != nil {
		t.Fatalf("NewPlanner error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil planner")
	}
}

func TestMindMapPlanner_ParseAndAddNodes(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantNodes int
	}{
		{
			name:      "valid structured output",
			input:     "claim|Test claim|0.8|0|\nevidence|Test evidence|0.7|1|supports\n",
			wantNodes: 2,
		},
		{
			name:      "empty lines skipped",
			input:     "\n\nclaim|Something|0.5|0|\n\n",
			wantNodes: 1,
		},
		{
			name:      "malformed lines skipped",
			input:     "this is not valid\nclaim|Valid|0.5|0|\nbad",
			wantNodes: 1,
		},
		{
			name:      "empty content skipped",
			input:     "claim||0.5|0|\nclaim|Valid|0.5|0|",
			wantNodes: 1,
		},
		{
			name:      "all types",
			input:     "claim|A|0.8|0|\nevidence|B|0.7|1|supports\nquestion|C|0.5|0|\nconclusion|D|0.9|1|derives_from\n",
			wantNodes: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &testLLM{}
			p := NewMindMapPlanner(model)

			err := p.parseAndAddNodes(tt.input)
			if err != nil {
				t.Fatalf("parseAndAddNodes error: %v", err)
			}
			if p.graph.NodeCount() != tt.wantNodes {
				t.Errorf("got %d nodes, want %d", p.graph.NodeCount(), tt.wantNodes)
			}
		})
	}
}

func TestMindMapPlanner_ParseNodeType(t *testing.T) {
	tests := []struct {
		input string
		want  NodeType
	}{
		{"claim", NodeClaim},
		{"CLAIM", NodeClaim},
		{"evidence", NodeEvidence},
		{"Evidence", NodeEvidence},
		{"question", NodeQuestion},
		{"conclusion", NodeConclusion},
		{"unknown", NodeClaim},
		{"", NodeClaim},
	}

	for _, tt := range tests {
		got := parseNodeType(tt.input)
		if got != tt.want {
			t.Errorf("parseNodeType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMindMapPlanner_ParseEdgeType(t *testing.T) {
	tests := []struct {
		input string
		want  EdgeType
	}{
		{"supports", EdgeSupports},
		{"contradicts", EdgeContradicts},
		{"derives_from", EdgeDerivesFrom},
		{"refines", EdgeRefines},
		{"unknown", ""},
		{"", ""},
	}

	for _, tt := range tests {
		got := parseEdgeType(tt.input)
		if got != tt.want {
			t.Errorf("parseEdgeType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMindMapPlanner_GraphSummary(t *testing.T) {
	model := &testLLM{}
	p := NewMindMapPlanner(model)

	// Empty graph.
	if p.graphSummary() != "" {
		t.Error("expected empty summary for empty graph")
	}

	// Add a node.
	p.graph.AddNode(NodeClaim, "test claim", 0.8, nil)
	summary := p.graphSummary()
	if !strings.Contains(summary, "test claim") {
		t.Error("summary should contain node content")
	}
	if !strings.Contains(summary, "[claim]") {
		t.Error("summary should contain node type")
	}
}

func TestMindMapPlanner_Plan_WithTools(t *testing.T) {
	model := &testLLM{
		generateFn: func(_ context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			for _, m := range msgs {
				if sm, ok := m.(*schema.SystemMessage); ok {
					if strings.Contains(sm.Text(), "Extract structured reasoning") {
						return schema.NewAIMessage("claim|Use tool|0.8|0|\n"), nil
					}
				}
			}
			// Return a tool call response.
			return &schema.AIMessage{
				ToolCalls: []schema.ToolCall{
					{ID: "tc1", Name: "search", Arguments: `{"q":"test"}`},
				},
			}, nil
		},
	}

	p := NewMindMapPlanner(model)
	actions, err := p.Plan(context.Background(), PlannerState{
		Input:    "search for something",
		Messages: []schema.Message{schema.NewHumanMessage("search for something")},
		Tools:    []tool.Tool{&simpleTool{toolName: "search"}},
	})

	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}
	if len(actions) == 0 {
		t.Fatal("expected at least one action")
	}
	if actions[0].Type != ActionTool {
		t.Errorf("expected ActionTool, got %s", actions[0].Type)
	}
}
