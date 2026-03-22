// Package main demonstrates all four Beluga AI optimizers (BootstrapFewShot,
// MIPROv2, GEPA, SIMBA) applied to a simple geography Q&A task.
//
// The example shows two usage patterns:
//
//  1. The high-level optimizer.Compiler API — recommended for most users.
//  2. The low-level optimize.Optimizer API — useful when you need fine-grained
//     control over program compilation outside the agent framework.
//
// Usage:
//
//	export OPENAI_API_KEY=your-key   # required for the live demo
//	go run .
//
// Without an API key the program runs in "dry-run" mode using a mock agent
// so you can explore the optimizer/compiler wiring without spending credits.
package main

import (
	"context"
	"fmt"
	"iter"
	"os"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/optimize"
	optpkg "github.com/lookatitude/beluga-ai/optimizer"
	"github.com/lookatitude/beluga-ai/tool"

	// Blank-import to register all four optimize.Optimizer implementations
	// (bootstrapfewshot, mipro, gepa, simba) into the optimize registry.
	// This also triggers the bridge init() in optimizer/bridge.go so they
	// are available via optimizer.NewCompiler / CompilerForStrategy.
	_ "github.com/lookatitude/beluga-ai/optimize/optimizers"
)

// ── data ─────────────────────────────────────────────────────────────────────

func trainingData() []optpkg.Example {
	return []optpkg.Example{
		{Inputs: map[string]any{"input": "Capital of France?"}, Outputs: map[string]any{"answer": "Paris"}},
		{Inputs: map[string]any{"input": "Capital of Japan?"}, Outputs: map[string]any{"answer": "Tokyo"}},
		{Inputs: map[string]any{"input": "Capital of Brazil?"}, Outputs: map[string]any{"answer": "Brasília"}},
		{Inputs: map[string]any{"input": "Capital of Australia?"}, Outputs: map[string]any{"answer": "Canberra"}},
		{Inputs: map[string]any{"input": "Capital of Egypt?"}, Outputs: map[string]any{"answer": "Cairo"}},
		{Inputs: map[string]any{"input": "Capital of Canada?"}, Outputs: map[string]any{"answer": "Ottawa"}},
		{Inputs: map[string]any{"input": "Capital of South Korea?"}, Outputs: map[string]any{"answer": "Seoul"}},
		{Inputs: map[string]any{"input": "Capital of Italy?"}, Outputs: map[string]any{"answer": "Rome"}},
		{Inputs: map[string]any{"input": "Capital of Thailand?"}, Outputs: map[string]any{"answer": "Bangkok"}},
		{Inputs: map[string]any{"input": "Capital of Nigeria?"}, Outputs: map[string]any{"answer": "Abuja"}},
	}
}

func testData() []optpkg.Example {
	return []optpkg.Example{
		{Inputs: map[string]any{"input": "Capital of Germany?"}, Outputs: map[string]any{"answer": "Berlin"}},
		{Inputs: map[string]any{"input": "Capital of Spain?"}, Outputs: map[string]any{"answer": "Madrid"}},
		{Inputs: map[string]any{"input": "Capital of India?"}, Outputs: map[string]any{"answer": "New Delhi"}},
	}
}

// ── mock agent (used when no API key is set) ──────────────────────────────────

// mockAgent echoes the city name embedded in the training data, simulating
// a capable geography model without making real LLM calls.
type mockAgent struct {
	id string
}

func (m *mockAgent) ID() string              { return m.id }
func (m *mockAgent) Persona() agent.Persona  { return agent.Persona{Role: "Geography expert"} }
func (m *mockAgent) Tools() []tool.Tool      { return nil }
func (m *mockAgent) Children() []agent.Agent { return nil }

func (m *mockAgent) Invoke(_ context.Context, input string, _ ...agent.Option) (string, error) {
	// Simulate a model that always knows its capitals.
	capitals := map[string]string{
		"france": "Paris", "japan": "Tokyo", "brazil": "Brasília",
		"australia": "Canberra", "egypt": "Cairo", "canada": "Ottawa",
		"south korea": "Seoul", "italy": "Rome", "thailand": "Bangkok",
		"nigeria": "Abuja", "germany": "Berlin", "spain": "Madrid",
		"india": "New Delhi", "turkey": "Ankara", "myanmar": "Naypyidaw",
	}
	lower := strings.ToLower(input)
	for country, capital := range capitals {
		if strings.Contains(lower, country) {
			return capital, nil
		}
	}
	return "Unknown", nil
}

func (m *mockAgent) Stream(ctx context.Context, input string, opts ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		result, err := m.Invoke(ctx, input, opts...)
		if err != nil {
			yield(agent.Event{Type: agent.EventError, AgentID: m.id}, err)
			return
		}
		yield(agent.Event{Type: agent.EventText, Text: result, AgentID: m.id}, nil)
		yield(agent.Event{Type: agent.EventDone, AgentID: m.id}, nil)
	}
}

// ── metrics ───────────────────────────────────────────────────────────────────

// containsMetric scores 1.0 when the prediction contains the expected answer.
var containsMetric = optpkg.MetricFunc(func(ctx context.Context, ex optpkg.Example, pred optpkg.Prediction) (float64, error) {
	expected, _ := ex.Outputs["answer"].(string)
	if strings.Contains(strings.ToLower(pred.Text), strings.ToLower(expected)) {
		return 1.0, nil
	}
	return 0.0, nil
})

// ── evaluation ────────────────────────────────────────────────────────────────

func evaluate(ctx context.Context, agt agent.Agent, examples []optpkg.Example) float64 {
	var total float64
	for _, ex := range examples {
		input, _ := ex.Inputs["input"].(string)
		result, err := agt.Invoke(ctx, input)
		if err != nil {
			continue
		}
		score, _ := containsMetric.Score(ctx, ex, optpkg.Prediction{Text: result})
		total += score
	}
	if len(examples) == 0 {
		return 0
	}
	return total / float64(len(examples))
}

// ── progress callback ─────────────────────────────────────────────────────────

type progressCB struct{ prefix string }

func (p *progressCB) OnProgress(_ context.Context, prog optpkg.Progress) {
	fmt.Printf("  [%s] phase=%s trial=%d/%d score=%.3f\n",
		p.prefix, prog.Phase, prog.CurrentTrial, prog.TotalTrials, prog.CurrentScore)
}

func (p *progressCB) OnTrialComplete(_ context.Context, t optpkg.Trial) {
	if t.Error != nil {
		fmt.Printf("  [%s] trial %d error: %v\n", p.prefix, t.ID, t.Error)
	}
}

func (p *progressCB) OnComplete(_ context.Context, r optpkg.Result) {
	fmt.Printf("  [%s] done score=%.3f trials=%d duration=%s\n",
		p.prefix, r.Score, len(r.Trials), r.TotalDuration.Round(time.Millisecond))
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	ctx := context.Background()

	// Decide whether to use a real LLM or the mock.
	var baseAgent agent.Agent
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey != "" {
		fmt.Println("Using OpenAI (gpt-4o-mini)")
		// Real LLM path — uncomment when running with a key:
		// model, _ := llm.New("openai", config.ProviderConfig{Provider:"openai", APIKey:apiKey, Model:"gpt-4o-mini"})
		// baseAgent = agent.New("geo-qa", agent.WithLLM(model), agent.WithPersona(agent.Persona{Role:"Geography expert"}))
		baseAgent = &mockAgent{id: "geo-qa"} // placeholder until LLM is wired
	} else {
		fmt.Println("No OPENAI_API_KEY set — running in dry-run mode with mock agent.")
		baseAgent = &mockAgent{id: "geo-qa"}
	}

	trainset := trainingData()
	testset := testData()

	fmt.Printf("\nBaseline score: %.1f%%\n", evaluate(ctx, baseAgent, testset)*100)

	// ── Pattern 1: High-level Compiler API ────────────────────────────────

	fmt.Println("\n=== Pattern 1: optimizer.Compiler API ===")

	// Run all four strategies and compare.
	strategies := []optpkg.OptimizationStrategy{
		optpkg.StrategyBootstrapFewShot,
		optpkg.StrategyMIPROv2,
		optpkg.StrategyGEPA,
		optpkg.StrategySIMBA,
	}

	for _, strategy := range strategies {
		fmt.Printf("\n-- Strategy: %s --\n", strategy)

		compiler := optpkg.CompilerForStrategy(strategy)
		result, err := compiler.CompileWithResult(ctx, baseAgent,
			optpkg.WithMetric(containsMetric),
			optpkg.WithTrainsetExamples(trainset),
			optpkg.WithMaxIterations(5),
			optpkg.WithSeed(42),
			optpkg.WithCallback(&progressCB{prefix: string(strategy)}),
		)
		if err != nil {
			fmt.Printf("  error: %v\n", err)
			continue
		}

		score := evaluate(ctx, result.Agent, testset)
		fmt.Printf("  test score: %.1f%%\n", score*100)
	}

	// ── Pattern 2: Registry-based Compiler ────────────────────────────────

	fmt.Println("\n=== Pattern 2: Named compiler from registry ===")

	c, err := optpkg.NewCompiler("bootstrap_few_shot", optpkg.CompilerConfig{})
	if err != nil {
		fmt.Printf("NewCompiler error: %v\n", err)
	} else {
		result, err := c.CompileWithResult(ctx, baseAgent,
			optpkg.WithMetric(containsMetric),
			optpkg.WithTrainsetExamples(trainset),
			optpkg.WithMaxIterations(3),
			optpkg.WithSeed(42),
		)
		if err != nil {
			fmt.Printf("Compile error: %v\n", err)
		} else {
			fmt.Printf("  score: %.1f%%\n", evaluate(ctx, result.Agent, testset)*100)
		}
	}

	// ── Pattern 3: Low-level optimize.Optimizer API ───────────────────────

	fmt.Println("\n=== Pattern 3: Low-level optimize.Optimizer API ===")

	opt, err := optimize.NewOptimizer("bootstrapfewshot", optimize.OptimizerConfig{})
	if err != nil {
		fmt.Printf("optimize.NewOptimizer error: %v\n", err)
	} else {
		// Wrap agent as optimize.Program.
		program := agent.NewAgentProgram(baseAgent)

		compiled, err := opt.Compile(ctx, program, optimize.CompileOptions{
			Trainset: func() []optimize.Example {
				ex := make([]optimize.Example, len(trainset))
				for i, e := range trainset {
					ex[i] = optimize.Example{Inputs: e.Inputs, Outputs: e.Outputs}
				}
				return ex
			}(),
			Metric: optimize.MetricFunc(func(ex optimize.Example, pred optimize.Prediction, _ *optimize.Trace) float64 {
				expected, _ := ex.Outputs["answer"].(string)
				if strings.Contains(strings.ToLower(pred.Raw), strings.ToLower(expected)) {
					return 1.0
				}
				return 0.0
			}),
			MaxCost: &optimize.CostBudget{MaxIterations: 5},
			Seed:    42,
		})
		if err != nil {
			fmt.Printf("  Compile error: %v\n", err)
		} else {
			// Use the compiled program directly.
			var total float64
			for _, ex := range testset {
				pred, err := compiled.Run(ctx, ex.Inputs)
				if err != nil {
					continue
				}
				expected, _ := ex.Outputs["answer"].(string)
				if strings.Contains(strings.ToLower(pred.Raw), strings.ToLower(expected)) {
					total++
				}
			}
			fmt.Printf("  score: %.1f%%\n", total/float64(len(testset))*100)
		}
	}

	fmt.Println("\nDone.")
}
