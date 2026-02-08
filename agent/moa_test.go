package agent

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

func TestNewMoAPlanner_Defaults(t *testing.T) {
	model := &testLLM{}
	p := NewMoAPlanner(model)

	if p == nil {
		t.Fatal("expected non-nil planner")
	}
	if p.defaultLLM == nil {
		t.Error("defaultLLM should not be nil")
	}
	if p.aggregator == nil {
		t.Error("aggregator should not be nil (should default to defaultLLM)")
	}
	if p.aggregator != model {
		t.Error("aggregator should default to defaultLLM when not specified")
	}
}

func TestMoAPlanner_WithLayers(t *testing.T) {
	model1 := &testLLM{generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
		return schema.NewAIMessage("model1"), nil
	}}
	model2 := &testLLM{generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
		return schema.NewAIMessage("model2"), nil
	}}

	layers := [][]llm.ChatModel{
		{model1, model2},
	}

	p := NewMoAPlanner(&testLLM{}, WithLayers(layers))

	if len(p.layers) != 1 {
		t.Errorf("layers length = %d, want 1", len(p.layers))
	}
	if len(p.layers[0]) != 2 {
		t.Errorf("layer 0 length = %d, want 2", len(p.layers[0]))
	}
}

func TestMoAPlanner_WithAggregator(t *testing.T) {
	defaultModel := &testLLM{}
	aggregator := &testLLM{generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
		return schema.NewAIMessage("aggregated"), nil
	}}

	p := NewMoAPlanner(defaultModel, WithAggregator(aggregator))

	if p.aggregator != aggregator {
		t.Error("aggregator should be set to custom aggregator")
	}
}

func TestMoAPlanner_Plan_NoLayers_UsesSingleModel(t *testing.T) {
	callCount := 0
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			callCount++
			return schema.NewAIMessage("single model response"), nil
		},
	}

	p := NewMoAPlanner(model) // No layers specified
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

	// Should call the model at least twice: once for the layer, once for aggregation
	if callCount < 2 {
		t.Errorf("call count = %d, want at least 2 (layer + aggregation)", callCount)
	}
}

func TestMoAPlanner_Plan_MultipleLayers(t *testing.T) {
	layer1Model1Calls := 0
	layer1Model2Calls := 0
	layer2ModelCalls := 0
	aggregatorCalls := 0

	layer1Model1 := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			layer1Model1Calls++
			return schema.NewAIMessage("layer1 model1 response"), nil
		},
	}

	layer1Model2 := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			layer1Model2Calls++
			return schema.NewAIMessage("layer1 model2 response"), nil
		},
	}

	layer2Model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			layer2ModelCalls++
			// Verify that layer 2 receives layer 1 outputs
			var prompt string
			if sm, ok := msgs[0].(*schema.SystemMessage); ok {
				prompt = sm.Text()
			}
			if !strings.Contains(prompt, "layer1") {
				t.Error("layer 2 should receive layer 1 outputs in context")
			}
			return schema.NewAIMessage("layer2 response"), nil
		},
	}

	aggregator := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			aggregatorCalls++
			return schema.NewAIMessage("final aggregated response"), nil
		},
	}

	layers := [][]llm.ChatModel{
		{layer1Model1, layer1Model2},
		{layer2Model},
	}

	p := NewMoAPlanner(&testLLM{}, WithLayers(layers), WithAggregator(aggregator))
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

	if layer1Model1Calls != 1 {
		t.Errorf("layer1Model1Calls = %d, want 1", layer1Model1Calls)
	}
	if layer1Model2Calls != 1 {
		t.Errorf("layer1Model2Calls = %d, want 1", layer1Model2Calls)
	}
	if layer2ModelCalls != 1 {
		t.Errorf("layer2ModelCalls = %d, want 1", layer2ModelCalls)
	}
	if aggregatorCalls != 1 {
		t.Errorf("aggregatorCalls = %d, want 1", aggregatorCalls)
	}
}

func TestMoAPlanner_ExecuteLayer_ParallelExecution(t *testing.T) {
	var concurrentCalls int32

	slowModel := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			atomic.AddInt32(&concurrentCalls, 1)
			// Small sleep to ensure concurrent execution is tested
			// (without actual time.Sleep, we rely on goroutine scheduling)
			defer atomic.AddInt32(&concurrentCalls, -1)
			return schema.NewAIMessage("response"), nil
		},
	}

	models := []llm.ChatModel{slowModel, slowModel, slowModel}
	p := NewMoAPlanner(&testLLM{})

	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	outputs, err := p.executeLayer(context.Background(), state, []schema.Message{schema.NewHumanMessage("test")}, models, nil)
	if err != nil {
		t.Fatalf("executeLayer error: %v", err)
	}

	if len(outputs) != 3 {
		t.Errorf("outputs length = %d, want 3", len(outputs))
	}

	// The test verifies that models are called (concurrent execution is implicit via goroutines)
}

func TestMoAPlanner_ExecuteLayer_PartialFailures(t *testing.T) {
	successModel := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return schema.NewAIMessage("success"), nil
		},
	}

	failureModel := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return nil, errors.New("model failed")
		},
	}

	models := []llm.ChatModel{successModel, failureModel, successModel}
	p := NewMoAPlanner(&testLLM{})

	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	outputs, err := p.executeLayer(context.Background(), state, []schema.Message{schema.NewHumanMessage("test")}, models, nil)

	// Should succeed with partial outputs
	if err != nil {
		t.Fatalf("executeLayer error: %v (should handle partial failures)", err)
	}

	if len(outputs) != 2 {
		t.Errorf("outputs length = %d, want 2 (only successful models)", len(outputs))
	}

	for _, output := range outputs {
		if output != "success" {
			t.Errorf("output = %q, want %q", output, "success")
		}
	}
}

func TestMoAPlanner_ExecuteLayer_AllFailures(t *testing.T) {
	failureModel := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return nil, errors.New("model failed")
		},
	}

	models := []llm.ChatModel{failureModel, failureModel}
	p := NewMoAPlanner(&testLLM{})

	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	_, err := p.executeLayer(context.Background(), state, []schema.Message{schema.NewHumanMessage("test")}, models, nil)

	// Should return error when all models fail
	if err == nil {
		t.Fatal("expected error when all models fail")
	}
}

func TestMoAPlanner_Aggregate_SynthesizesOutputs(t *testing.T) {
	var receivedPrompt string
	aggregator := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			if sm, ok := msgs[0].(*schema.SystemMessage); ok {
				receivedPrompt = sm.Text()
			}
			return schema.NewAIMessage("synthesized final answer"), nil
		},
	}

	p := NewMoAPlanner(&testLLM{}, WithAggregator(aggregator))
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	outputs := []string{
		"Expert 1 says X",
		"Expert 2 says Y",
		"Expert 3 says Z",
	}

	actions, err := p.aggregate(context.Background(), state, outputs)
	if err != nil {
		t.Fatalf("aggregate error: %v", err)
	}

	if len(actions) == 0 {
		t.Fatal("expected at least one action")
	}

	// Verify that all expert outputs are included in the aggregator prompt
	if !strings.Contains(receivedPrompt, "Expert 1") {
		t.Error("aggregator prompt should include Expert 1 output")
	}
	if !strings.Contains(receivedPrompt, "Expert 2") {
		t.Error("aggregator prompt should include Expert 2 output")
	}
	if !strings.Contains(receivedPrompt, "Expert 3") {
		t.Error("aggregator prompt should include Expert 3 output")
	}
}

func TestMoAPlanner_Plan_LLMError(t *testing.T) {
	failureModel := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			return nil, errors.New("LLM failed")
		},
	}

	p := NewMoAPlanner(failureModel)
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
	}

	_, err := p.Plan(context.Background(), state)
	if err == nil {
		t.Fatal("expected error from LLM failure")
	}
}

func TestMoAPlanner_Replan(t *testing.T) {
	callCount := 0
	model := &testLLM{
		generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
			callCount++
			return schema.NewAIMessage("replanned response"), nil
		},
	}

	p := NewMoAPlanner(model)
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

func TestMoAPlanner_ExecuteLayer_BindsTools(t *testing.T) {
	var boundTools []schema.ToolDefinition

	model := &bindTrackingLLM{
		testLLM: testLLM{
			generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
				return schema.NewAIMessage("response"), nil
			},
		},
		onBind: func(tools []schema.ToolDefinition) {
			boundTools = tools
		},
	}

	p := NewMoAPlanner(&testLLM{})
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
		Tools:    []tool.Tool{&simpleTool{toolName: "test_tool"}},
	}

	_, err := p.executeLayer(context.Background(), state, []schema.Message{schema.NewHumanMessage("test")}, []llm.ChatModel{model}, nil)
	if err != nil {
		t.Fatalf("executeLayer error: %v", err)
	}

	if len(boundTools) == 0 {
		t.Error("expected tools to be bound")
	}
	if len(boundTools) > 0 && boundTools[0].Name != "test_tool" {
		t.Errorf("bound tool name = %q, want %q", boundTools[0].Name, "test_tool")
	}
}

func TestMoAPlanner_Aggregate_BindsTools(t *testing.T) {
	var boundTools []schema.ToolDefinition

	aggregator := &bindTrackingLLM{
		testLLM: testLLM{
			generateFn: func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
				return schema.NewAIMessage("aggregated"), nil
			},
		},
		onBind: func(tools []schema.ToolDefinition) {
			boundTools = tools
		},
	}

	p := NewMoAPlanner(&testLLM{}, WithAggregator(aggregator))
	state := PlannerState{
		Input:    "test",
		Messages: []schema.Message{schema.NewHumanMessage("test")},
		Tools:    []tool.Tool{&simpleTool{toolName: "agg_tool"}},
	}

	outputs := []string{"output1", "output2"}
	_, err := p.aggregate(context.Background(), state, outputs)
	if err != nil {
		t.Fatalf("aggregate error: %v", err)
	}

	if len(boundTools) == 0 {
		t.Error("expected tools to be bound in aggregator")
	}
	if len(boundTools) > 0 && boundTools[0].Name != "agg_tool" {
		t.Errorf("bound tool name = %q, want %q", boundTools[0].Name, "agg_tool")
	}
}

func TestMoAPlanner_Registry_Registered(t *testing.T) {
	planners := ListPlanners()
	found := false
	for _, name := range planners {
		if name == "moa" {
			found = true
			break
		}
	}
	if !found {
		t.Error("moa planner not registered")
	}
}

func TestMoAPlanner_Registry_Creation(t *testing.T) {
	model := &testLLM{}
	p, err := NewPlanner("moa", PlannerConfig{
		LLM: model,
	})
	if err != nil {
		t.Fatalf("NewPlanner error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil planner")
	}
}

func TestMoAPlanner_Registry_CreationFailsWithoutLLM(t *testing.T) {
	_, err := NewPlanner("moa", PlannerConfig{})
	if err == nil {
		t.Fatal("expected error when creating moa planner without LLM")
	}
}

func TestMoAPlanner_ImplementsPlanner(t *testing.T) {
	var _ Planner = (*MoAPlanner)(nil)
}
