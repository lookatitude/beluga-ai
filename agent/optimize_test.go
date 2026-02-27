package agent

import (
	"context"
	"fmt"
	"iter"
	"strings"
	"testing"

	"github.com/lookatitude/beluga-ai/optimize"
	"github.com/lookatitude/beluga-ai/tool"
)

// --- Mocks ---

// optMockAgent implements Agent for optimization testing.
type optMockAgent struct {
	id      string
	persona Persona
	tools   []tool.Tool
	result  string
	err     error
}

func (m *optMockAgent) ID() string                { return m.id }
func (m *optMockAgent) Persona() Persona          { return m.persona }
func (m *optMockAgent) Tools() []tool.Tool        { return m.tools }
func (m *optMockAgent) Children() []Agent         { return nil }

func (m *optMockAgent) Invoke(ctx context.Context, input string, opts ...Option) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.result, nil
}

func (m *optMockAgent) Stream(ctx context.Context, input string, opts ...Option) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		if m.err != nil {
			yield(Event{Type: EventError, AgentID: m.id}, m.err)
			return
		}
		if !yield(Event{Type: EventText, Text: m.result, AgentID: m.id}, nil) {
			return
		}
		yield(Event{Type: EventDone, AgentID: m.id}, nil)
	}
}

// mockTool implements tool.Tool for testing.
type mockTool struct {
	name string
	desc string
}

func (t *mockTool) Name() string                  { return t.name }
func (t *mockTool) Description() string            { return t.desc }
func (t *mockTool) InputSchema() map[string]any    { return map[string]any{"type": "object"} }
func (t *mockTool) Execute(_ context.Context, _ map[string]any) (*tool.Result, error) {
	return &tool.Result{}, nil
}

// mockOptimizer implements optimize.Optimizer for testing.
type mockOptimizer struct {
	compileFn func(ctx context.Context, program optimize.Program, opts optimize.CompileOptions) (optimize.Program, error)
}

func (m *mockOptimizer) Compile(ctx context.Context, program optimize.Program, opts optimize.CompileOptions) (optimize.Program, error) {
	if m.compileFn != nil {
		return m.compileFn(ctx, program, opts)
	}
	// Default: return program with demos from first trainset example
	if len(opts.Trainset) > 0 {
		return program.WithDemos(opts.Trainset), nil
	}
	return program, nil
}

// --- Tests ---

func TestAgentProgram_Run(t *testing.T) {
	tests := []struct {
		name     string
		agent    Agent
		inputs   map[string]interface{}
		wantOut  string
		wantErr  bool
	}{
		{
			name: "basic input key",
			agent: &optMockAgent{
				id:     "test-agent",
				result: "hello world",
			},
			inputs:  map[string]interface{}{"input": "hi"},
			wantOut: "hello world",
		},
		{
			name: "multiple input fields joined",
			agent: &optMockAgent{
				id:     "test-agent",
				result: "combined response",
			},
			inputs:  map[string]interface{}{"question": "what?"},
			wantOut: "combined response",
		},
		{
			name: "agent error propagated",
			agent: &optMockAgent{
				id:  "test-agent",
				err: fmt.Errorf("agent failed"),
			},
			inputs:  map[string]interface{}{"input": "hi"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program := NewAgentProgram(tt.agent)
			pred, err := program.Run(context.Background(), tt.inputs)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			output, ok := pred.Outputs["output"].(string)
			if !ok {
				t.Fatal("Run() output not a string")
			}
			if output != tt.wantOut {
				t.Errorf("Run() output = %q, want %q", output, tt.wantOut)
			}
		})
	}
}

func TestAgentProgram_WithDemos(t *testing.T) {
	agent := &optMockAgent{id: "test", result: "ok"}
	program := NewAgentProgram(agent)

	demos := []optimize.Example{
		{
			Inputs:  map[string]interface{}{"input": "hello"},
			Outputs: map[string]interface{}{"output": "world"},
		},
	}

	withDemos := program.WithDemos(demos)
	if withDemos == program {
		t.Error("WithDemos() should return a new program")
	}

	// Original should have no demos
	if len(program.demos) != 0 {
		t.Error("original program should have no demos")
	}

	// New should have demos
	apNew, ok := withDemos.(*AgentProgram)
	if !ok {
		t.Fatal("expected *AgentProgram from WithDemos")
	}
	if len(apNew.demos) != 1 {
		t.Errorf("WithDemos() demos count = %d, want 1", len(apNew.demos))
	}
}

func TestAgentProgram_GetSignature(t *testing.T) {
	agent := &optMockAgent{
		id: "test",
		persona: Persona{
			Role: "assistant",
			Goal: "help users",
		},
		tools: []tool.Tool{
			&mockTool{name: "search", desc: "search the web"},
		},
	}

	program := NewAgentProgram(agent)
	sig := program.GetSignature()

	inputFields := sig.GetInputFields()
	if len(inputFields) != 1 || inputFields[0].Name != "input" {
		t.Errorf("GetInputFields() = %v, want single 'input' field", inputFields)
	}

	outputFields := sig.GetOutputFields()
	if len(outputFields) != 1 || outputFields[0].Name != "output" {
		t.Errorf("GetOutputFields() = %v, want single 'output' field", outputFields)
	}
}

func TestAgentSignature_Render(t *testing.T) {
	sig := &agentSignature{
		persona: Persona{
			Role: "assistant",
			Goal: "help users",
		},
	}

	rendered, err := sig.Render(map[string]interface{}{"input": "test question"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(rendered, "assistant") {
		t.Error("Render() should contain role")
	}
	if !strings.Contains(rendered, "help users") {
		t.Error("Render() should contain goal")
	}
	if !strings.Contains(rendered, "test question") {
		t.Error("Render() should contain input")
	}
}

func TestAgentSignature_RenderWithDemos(t *testing.T) {
	sig := &agentSignature{
		persona: Persona{Role: "helper"},
		demos: []optimize.Example{
			{
				Inputs:  map[string]interface{}{"input": "hi"},
				Outputs: map[string]interface{}{"output": "hello"},
			},
		},
	}

	rendered, err := sig.Render(map[string]interface{}{"input": "test"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(rendered, "Examples:") {
		t.Error("Render() should contain examples section")
	}
	if !strings.Contains(rendered, "hi") {
		t.Error("Render() should contain demo input")
	}
	if !strings.Contains(rendered, "hello") {
		t.Error("Render() should contain demo output")
	}
}

func TestAgentSignature_Parse(t *testing.T) {
	sig := &agentSignature{}
	result, err := sig.Parse("some response")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if result["output"] != "some response" {
		t.Errorf("Parse() = %v, want output='some response'", result)
	}
}

func TestPersonaWithDemos(t *testing.T) {
	base := Persona{
		Role:      "assistant",
		Goal:      "help",
		Backstory: "You are helpful.",
	}

	t.Run("no demos returns base", func(t *testing.T) {
		result := personaWithDemos(base, nil)
		if result.Backstory != base.Backstory {
			t.Errorf("expected base backstory, got %q", result.Backstory)
		}
	})

	t.Run("demos appended to backstory", func(t *testing.T) {
		demos := []optimize.Example{
			{
				Inputs:  map[string]interface{}{"input": "Q1"},
				Outputs: map[string]interface{}{"output": "A1"},
			},
		}
		result := personaWithDemos(base, demos)
		if !strings.Contains(result.Backstory, "You are helpful.") {
			t.Error("should preserve original backstory")
		}
		if !strings.Contains(result.Backstory, "Example 1") {
			t.Error("should contain example header")
		}
		if !strings.Contains(result.Backstory, "Q1") {
			t.Error("should contain demo input")
		}
		if !strings.Contains(result.Backstory, "A1") {
			t.Error("should contain demo output")
		}
		// Verify other fields preserved
		if result.Role != base.Role {
			t.Errorf("Role = %q, want %q", result.Role, base.Role)
		}
		if result.Goal != base.Goal {
			t.Errorf("Goal = %q, want %q", result.Goal, base.Goal)
		}
	})
}

func TestOptimizationOptions(t *testing.T) {
	trainset := []optimize.Example{
		{Inputs: map[string]interface{}{"input": "q"}, Outputs: map[string]interface{}{"output": "a"}},
	}
	metric := optimize.MetricFunc(func(e optimize.Example, p optimize.Prediction, tr *optimize.Trace) float64 {
		return 1.0
	})
	budget := optimize.CostBudget{MaxDollars: 10.0, MaxIterations: 100}

	cfg := defaultConfig()
	WithOptimizer("bootstrapfewshot", optimize.OptimizerConfig{})(& cfg)
	WithTrainset(trainset)(&cfg)
	WithMetric(metric)(&cfg)
	WithOptimizationBudget(budget)(&cfg)
	WithOptimizationWorkers(4)(&cfg)
	WithOptimizationSeed(42)(&cfg)

	optCfg, err := extractOptimizeConfig(cfg.metadata)
	if err != nil {
		t.Fatalf("extractOptimizeConfig() error = %v", err)
	}

	if optCfg.optimizerName != "bootstrapfewshot" {
		t.Errorf("optimizerName = %q, want 'bootstrapfewshot'", optCfg.optimizerName)
	}
	if len(optCfg.trainset) != 1 {
		t.Errorf("trainset length = %d, want 1", len(optCfg.trainset))
	}
	if optCfg.budget == nil || optCfg.budget.MaxDollars != 10.0 {
		t.Errorf("budget = %v, want MaxDollars=10.0", optCfg.budget)
	}
	if optCfg.numWorkers != 4 {
		t.Errorf("numWorkers = %d, want 4", optCfg.numWorkers)
	}
	if optCfg.seed != 42 {
		t.Errorf("seed = %d, want 42", optCfg.seed)
	}
}

func TestExtractOptimizeConfig_Errors(t *testing.T) {
	tests := []struct {
		name    string
		meta    map[string]any
		wantMsg string
	}{
		{
			name:    "nil metadata",
			meta:    nil,
			wantMsg: "no optimization configuration",
		},
		{
			name:    "no optimizer",
			meta:    map[string]any{},
			wantMsg: "no optimizer specified",
		},
		{
			name: "no trainset",
			meta: map[string]any{
				"_optimize_name": "test",
			},
			wantMsg: "no training data",
		},
		{
			name: "no metric",
			meta: map[string]any{
				"_optimize_name":     "test",
				"_optimize_trainset": []optimize.Example{{Inputs: map[string]interface{}{"a": "b"}}},
			},
			wantMsg: "no metric specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := extractOptimizeConfig(tt.meta)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestOptimizedAgent_Interface(t *testing.T) {
	base := &optMockAgent{
		id:      "test-agent",
		persona: Persona{Role: "helper"},
		tools:   []tool.Tool{&mockTool{name: "search"}},
		result:  "optimized response",
	}

	optimized := &OptimizedAgent{
		base: &BaseAgent{id: "test-agent", config: agentConfig{
			persona: base.persona,
			tools:   base.tools,
		}},
		program: NewAgentProgram(base),
	}

	// Verify it satisfies Agent interface
	var _ Agent = optimized

	if optimized.ID() != "test-agent-optimized" {
		t.Errorf("ID() = %q, want 'test-agent-optimized'", optimized.ID())
	}

	result, err := optimized.Invoke(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if result != "optimized response" {
		t.Errorf("Invoke() = %q, want 'optimized response'", result)
	}
}

func TestOptimizedAgent_Stream(t *testing.T) {
	base := &optMockAgent{
		id:     "test-agent",
		result: "streamed output",
	}

	optimized := &OptimizedAgent{
		base: &BaseAgent{id: "test-agent", config: agentConfig{
			persona: base.persona,
		}},
		program: NewAgentProgram(base),
	}

	var texts []string
	var gotDone bool
	for event, err := range optimized.Stream(context.Background(), "test") {
		if err != nil {
			t.Fatalf("Stream() error = %v", err)
		}
		switch event.Type {
		case EventText:
			texts = append(texts, event.Text)
		case EventDone:
			gotDone = true
		}
	}

	if len(texts) != 1 || texts[0] != "streamed output" {
		t.Errorf("Stream() texts = %v, want ['streamed output']", texts)
	}
	if !gotDone {
		t.Error("Stream() should emit EventDone")
	}
}

func TestBaseAgent_Optimize_Integration(t *testing.T) {
	// Register a mock optimizer
	optimize.RegisterOptimizer("mock-test", func(cfg optimize.OptimizerConfig) (optimize.Optimizer, error) {
		return &mockOptimizer{
			compileFn: func(ctx context.Context, program optimize.Program, opts optimize.CompileOptions) (optimize.Program, error) {
				// Verify we received the trainset and metric
				if len(opts.Trainset) == 0 {
					return nil, fmt.Errorf("expected trainset")
				}
				if opts.Metric == nil {
					return nil, fmt.Errorf("expected metric")
				}
				// Return program with demos
				return program.WithDemos(opts.Trainset), nil
			},
		}, nil
	})

	trainset := []optimize.Example{
		{
			Inputs:  map[string]interface{}{"input": "What is Go?"},
			Outputs: map[string]interface{}{"output": "Go is a programming language."},
		},
	}

	metric := optimize.MetricFunc(func(e optimize.Example, p optimize.Prediction, tr *optimize.Trace) float64 {
		if _, ok := p.Outputs["output"]; ok {
			return 1.0
		}
		return 0.0
	})

	// We can't use a full BaseAgent.Optimize() here because it needs a real
	// LLM for the planner. Instead, test the pieces individually.

	// Test that extractOptimizeConfig works with agent metadata
	agent := New("integration-test",
		WithPersona(Persona{Role: "assistant", Goal: "answer questions"}),
		WithOptimizer("mock-test", optimize.OptimizerConfig{}),
		WithTrainset(trainset),
		WithMetric(metric),
		WithOptimizationBudget(optimize.CostBudget{MaxIterations: 10}),
	)

	optCfg, err := extractOptimizeConfig(agent.config.metadata)
	if err != nil {
		t.Fatalf("extractOptimizeConfig() error = %v", err)
	}

	// Create optimizer from registry
	optimizer, err := optimize.NewOptimizer(optCfg.optimizerName, optCfg.optimizerCfg)
	if err != nil {
		t.Fatalf("NewOptimizer() error = %v", err)
	}

	// Wrap agent as program and optimize
	program := NewAgentProgram(agent)
	compiled, err := optimizer.Compile(context.Background(), program, optimize.CompileOptions{
		Trainset: optCfg.trainset,
		Metric:   optCfg.metric,
		MaxCost:  optCfg.budget,
	})
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	// Verify the compiled program has demos
	compiledAP, ok := compiled.(*AgentProgram)
	if !ok {
		t.Fatal("expected *AgentProgram from Compile")
	}
	if len(compiledAP.demos) != 1 {
		t.Errorf("compiled demos = %d, want 1", len(compiledAP.demos))
	}

	// Verify signature is derived from persona
	sig := compiledAP.GetSignature()
	rendered, err := sig.Render(map[string]interface{}{"input": "test"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(rendered, "assistant") {
		t.Error("signature should contain agent role")
	}
}

func TestValsetOption(t *testing.T) {
	valset := []optimize.Example{
		{Inputs: map[string]interface{}{"input": "val"}, Outputs: map[string]interface{}{"output": "result"}},
	}

	cfg := defaultConfig()
	WithOptimizer("test", optimize.OptimizerConfig{})(&cfg)
	WithTrainset([]optimize.Example{{Inputs: map[string]interface{}{"input": "t"}}})(&cfg)
	WithMetric(optimize.MetricFunc(func(e optimize.Example, p optimize.Prediction, tr *optimize.Trace) float64 { return 1.0 }))(&cfg)
	WithValset(valset)(&cfg)

	optCfg, err := extractOptimizeConfig(cfg.metadata)
	if err != nil {
		t.Fatalf("extractOptimizeConfig() error = %v", err)
	}
	if len(optCfg.valset) != 1 {
		t.Errorf("valset length = %d, want 1", len(optCfg.valset))
	}
}
