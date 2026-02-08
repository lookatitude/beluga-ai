package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

func TestNewSelfDiscoverPlanner(t *testing.T) {
	model := &testLLM{}
	p := NewSelfDiscoverPlanner(model)

	if p == nil {
		t.Fatal("expected non-nil planner")
	}
	if p.llm == nil {
		t.Error("llm should not be nil")
	}
	if p.modules == nil {
		t.Error("modules should not be nil")
	}
	if len(p.modules) != len(DefaultReasoningModules) {
		t.Errorf("modules length = %d, want %d (default)", len(p.modules), len(DefaultReasoningModules))
	}
}

func TestNewSelfDiscoverPlanner_WithReasoningModules(t *testing.T) {
	customModules := []ReasoningModule{
		{Name: "custom1", Description: "Custom module 1", Template: "Template 1"},
		{Name: "custom2", Description: "Custom module 2", Template: "Template 2"},
	}

	model := &testLLM{}
	p := NewSelfDiscoverPlanner(model, WithReasoningModules(customModules))

	if len(p.modules) != 2 {
		t.Fatalf("modules length = %d, want 2", len(p.modules))
	}
	if p.modules[0].Name != "custom1" {
		t.Errorf("modules[0].Name = %q, want %q", p.modules[0].Name, "custom1")
	}
	if p.modules[1].Name != "custom2" {
		t.Errorf("modules[1].Name = %q, want %q", p.modules[1].Name, "custom2")
	}
}

func TestDefaultReasoningModules(t *testing.T) {
	expectedModules := map[string]bool{
		"critical_thinking":   true,
		"decomposition":       true,
		"analogical_reasoning": true,
		"causal_reasoning":    true,
		"constraint_analysis": true,
		"abstraction":         true,
		"step_by_step":        true,
		"hypothesis_testing":  true,
	}

	if len(DefaultReasoningModules) != len(expectedModules) {
		t.Errorf("DefaultReasoningModules length = %d, want %d", len(DefaultReasoningModules), len(expectedModules))
	}

	for _, module := range DefaultReasoningModules {
		if !expectedModules[module.Name] {
			t.Errorf("unexpected module: %q", module.Name)
		}
		if module.Description == "" {
			t.Errorf("module %q has empty description", module.Name)
		}
		if module.Template == "" {
			t.Errorf("module %q has empty template", module.Name)
		}
	}
}

func TestSelfDiscoverPlanner_Plan_ThreePhases(t *testing.T) {
	callCount := 0
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			callCount++
			switch callCount {
			case 1: // SELECT phase
				return schema.NewAIMessage("decomposition\ncritical_thinking"), nil
			case 2: // ADAPT phase
				return schema.NewAIMessage("1. Break down the problem\n2. Evaluate critically"), nil
			case 3: // IMPLEMENT phase
				return schema.NewAIMessage("Final answer based on reasoning"), nil
			default:
				return nil, errors.New("too many calls")
			}
		},
	}

	p := NewSelfDiscoverPlanner(model)
	state := PlannerState{
		Input:    "Solve this problem",
		Messages: []schema.Message{schema.NewHumanMessage("Solve this problem")},
	}

	actions, err := p.Plan(context.Background(), state)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}

	if callCount != 3 {
		t.Errorf("LLM call count = %d, want 3 (select + adapt + implement)", callCount)
	}

	if len(actions) == 0 {
		t.Fatal("expected at least one action")
	}
}

func TestSelfDiscoverPlanner_Plan_SelectParsesModuleNames(t *testing.T) {
	callCount := 0
	var selectedModules []string

	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			callCount++
			switch callCount {
			case 1: // SELECT phase - return module names in various formats
				return schema.NewAIMessage("1. decomposition\n- critical_thinking\nabstraction"), nil
			case 2: // ADAPT phase - capture context to verify parsing
				// Check that the selected modules are in the prompt
				lastMsg := msgs[len(msgs)-1]
				var prompt string
				if hm, ok := lastMsg.(*schema.HumanMessage); ok {
					prompt = hm.Text()
				}
				if strings.Contains(prompt, "decomposition") {
					selectedModules = append(selectedModules, "decomposition")
				}
				if strings.Contains(prompt, "critical_thinking") {
					selectedModules = append(selectedModules, "critical_thinking")
				}
				if strings.Contains(prompt, "abstraction") {
					selectedModules = append(selectedModules, "abstraction")
				}
				return schema.NewAIMessage("Adapted structure"), nil
			case 3: // IMPLEMENT phase
				return schema.NewAIMessage("Final answer"), nil
			default:
				return nil, errors.New("too many calls")
			}
		},
	}

	p := NewSelfDiscoverPlanner(model)
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	_, err := p.Plan(context.Background(), state)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}

	// Verify that the selected modules were parsed and passed to adapt phase
	if len(selectedModules) < 2 {
		t.Errorf("selectedModules = %v, expected at least 2 modules", selectedModules)
	}
}

func TestSelfDiscoverPlanner_Plan_LLMError(t *testing.T) {
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return nil, errors.New("LLM failed")
		},
	}

	p := NewSelfDiscoverPlanner(model)
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	_, err := p.Plan(context.Background(), state)
	if err == nil {
		t.Fatal("expected error from LLM failure")
	}
}

func TestSelfDiscoverPlanner_Replan_WithCachedStructure(t *testing.T) {
	callCount := 0
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			callCount++
			// Should only be called once for implement phase when structure is cached
			return schema.NewAIMessage("Response using cached structure"), nil
		},
	}

	p := NewSelfDiscoverPlanner(model)
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
		Metadata: map[string]any{
			"self_discover_structure": "1. Cached step 1\n2. Cached step 2",
		},
	}

	actions, err := p.Replan(context.Background(), state)
	if err != nil {
		t.Fatalf("Replan error: %v", err)
	}

	if callCount != 1 {
		t.Errorf("LLM call count = %d, want 1 (only implement, reusing cached structure)", callCount)
	}

	if len(actions) == 0 {
		t.Fatal("expected at least one action")
	}
}

func TestSelfDiscoverPlanner_Replan_WithoutCachedStructure(t *testing.T) {
	callCount := 0
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			callCount++
			switch callCount {
			case 1: // SELECT
				return schema.NewAIMessage("decomposition"), nil
			case 2: // ADAPT
				return schema.NewAIMessage("Adapted structure"), nil
			case 3: // IMPLEMENT
				return schema.NewAIMessage("Final answer"), nil
			default:
				return nil, errors.New("too many calls")
			}
		},
	}

	p := NewSelfDiscoverPlanner(model)
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
		Metadata: map[string]any{}, // No cached structure
	}

	actions, err := p.Replan(context.Background(), state)
	if err != nil {
		t.Fatalf("Replan error: %v", err)
	}

	if callCount != 3 {
		t.Errorf("LLM call count = %d, want 3 (full plan without cached structure)", callCount)
	}

	if len(actions) == 0 {
		t.Fatal("expected at least one action")
	}
}

func TestSelfDiscoverPlanner_Registry_Registered(t *testing.T) {
	planners := ListPlanners()
	found := false
	for _, name := range planners {
		if name == "self-discover" {
			found = true
			break
		}
	}
	if !found {
		t.Error("self-discover planner not registered")
	}
}

func TestSelfDiscoverPlanner_Registry_Creation(t *testing.T) {
	model := &testLLM{}
	p, err := NewPlanner("self-discover", PlannerConfig{
		LLM: model,
	})
	if err != nil {
		t.Fatalf("NewPlanner error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil planner")
	}
}

func TestSelfDiscoverPlanner_Registry_CreationFailsWithoutLLM(t *testing.T) {
	_, err := NewPlanner("self-discover", PlannerConfig{})
	if err == nil {
		t.Fatal("expected error when creating self-discover planner without LLM")
	}
}

func TestSelfDiscoverPlanner_ImplementsPlanner(t *testing.T) {
	var _ Planner = (*SelfDiscoverPlanner)(nil)
}

func TestSelfDiscoverPlanner_Plan_StoresStructureInMetadata(t *testing.T) {
	callCount := 0
	adaptedStructure := "Step 1: Decompose\nStep 2: Analyze"

	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			callCount++
			switch callCount {
			case 1: // SELECT
				return schema.NewAIMessage("decomposition"), nil
			case 2: // ADAPT
				return schema.NewAIMessage(adaptedStructure), nil
			case 3: // IMPLEMENT
				return schema.NewAIMessage("Final answer"), nil
			default:
				return nil, errors.New("too many calls")
			}
		},
	}

	p := NewSelfDiscoverPlanner(model)
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	actions, err := p.Plan(context.Background(), state)
	if err != nil {
		t.Fatalf("Plan error: %v", err)
	}

	if len(actions) == 0 {
		t.Fatal("expected at least one action")
	}

	// Check that the first action has the structure stored in metadata
	if actions[0].Metadata == nil {
		t.Fatal("expected metadata in first action")
	}

	structure, ok := actions[0].Metadata["self_discover_structure"].(string)
	if !ok {
		t.Fatal("expected self_discover_structure in metadata")
	}

	if structure != adaptedStructure {
		t.Errorf("stored structure = %q, want %q", structure, adaptedStructure)
	}
}
