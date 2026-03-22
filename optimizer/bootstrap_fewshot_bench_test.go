package optimizer

import (
	"context"
	"fmt"
	"iter"
	"testing"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/tool"
)

// benchAgent echoes inputs — deterministic and allocation-light.
type benchAgent struct{ id string }

func (a *benchAgent) ID() string              { return a.id }
func (a *benchAgent) Persona() agent.Persona  { return agent.Persona{} }
func (a *benchAgent) Tools() []tool.Tool      { return nil }
func (a *benchAgent) Children() []agent.Agent { return nil }
func (a *benchAgent) Invoke(_ context.Context, input string, _ ...agent.Option) (string, error) {
	return input, nil
}
func (a *benchAgent) Stream(_ context.Context, input string, _ ...agent.Option) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		yield(agent.Event{Type: agent.EventText, Text: input, AgentID: a.id}, nil)
	}
}

// buildTrainset creates n examples where input == expected answer.
func buildTrainset(n int) Dataset {
	examples := make([]Example, n)
	for i := range examples {
		v := fmt.Sprintf("example-%d", i)
		examples[i] = Example{
			Inputs:  map[string]any{"input": v},
			Outputs: map[string]any{"answer": v},
		}
	}
	return Dataset{Examples: examples}
}

// manualDemoPrompt builds a demo prompt manually (baseline without optimizer machinery).
func manualDemoPrompt(input string, examples []Example) string {
	return buildDemoPrompt(input, examples)
}

// BenchmarkBootstrapFewShot_SmallTrainset benchmarks compilation with 5 examples.
func BenchmarkBootstrapFewShot_SmallTrainset(b *testing.B) {
	trainset := buildTrainset(5)
	opt := newBootstrapFewShotOptimizer(1, nil,
		WithBootstrapMaxBootstrapped(4),
		WithBootstrapMaxLabeled(4),
	)
	agt := &benchAgent{id: "bench-small"}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := opt.Optimize(context.Background(), agt, OptimizeOptions{
			Metric:   exactMatchMetricFn,
			Trainset: trainset,
		})
		if err != nil {
			b.Fatalf("Optimize error: %v", err)
		}
	}
}

// BenchmarkBootstrapFewShot_MediumTrainset benchmarks compilation with 25 examples.
func BenchmarkBootstrapFewShot_MediumTrainset(b *testing.B) {
	trainset := buildTrainset(25)
	opt := newBootstrapFewShotOptimizer(1, nil,
		WithBootstrapMaxBootstrapped(4),
		WithBootstrapMaxLabeled(16),
	)
	agt := &benchAgent{id: "bench-medium"}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := opt.Optimize(context.Background(), agt, OptimizeOptions{
			Metric:   exactMatchMetricFn,
			Trainset: trainset,
		})
		if err != nil {
			b.Fatalf("Optimize error: %v", err)
		}
	}
}

// BenchmarkBootstrapFewShot_LargeTrainset benchmarks compilation with 100 examples.
func BenchmarkBootstrapFewShot_LargeTrainset(b *testing.B) {
	trainset := buildTrainset(100)
	opt := newBootstrapFewShotOptimizer(1, nil,
		WithBootstrapMaxBootstrapped(4),
		WithBootstrapMaxLabeled(16),
	)
	agt := &benchAgent{id: "bench-large"}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_, err := opt.Optimize(context.Background(), agt, OptimizeOptions{
			Metric:   exactMatchMetricFn,
			Trainset: trainset,
		})
		if err != nil {
			b.Fatalf("Optimize error: %v", err)
		}
	}
}

// BenchmarkOptimizedVsManual compares:
//  1. Running a demoAgent (bootstrapped, 4 demos) vs a raw agent.
//  2. Both call the same underlying echo agent so latency is pure overhead.
func BenchmarkOptimizedVsManual(b *testing.B) {
	const numDemos = 4
	demos := make([]Example, numDemos)
	for i := range demos {
		demos[i] = Example{
			Inputs:  map[string]any{"input": fmt.Sprintf("q%d", i)},
			Outputs: map[string]any{"answer": fmt.Sprintf("a%d", i)},
		}
	}

	base := &benchAgent{id: "vs-base"}

	b.Run("manual-no-demos", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_, err := base.Invoke(context.Background(), "test input")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("manual-prompt-build", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			prompt := manualDemoPrompt("test input", demos)
			_, err := base.Invoke(context.Background(), prompt)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("optimized-demo-agent", func(b *testing.B) {
		da := newDemoAgent(base, demos)
		b.ReportAllocs()
		for range b.N {
			_, err := da.Invoke(context.Background(), "test input")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkBuildDemoPrompt benchmarks the prompt construction function in isolation.
func BenchmarkBuildDemoPrompt(b *testing.B) {
	demos := make([]Example, 8)
	for i := range demos {
		demos[i] = Example{
			Inputs:  map[string]any{"input": fmt.Sprintf("question-%d", i)},
			Outputs: map[string]any{"answer": fmt.Sprintf("answer-%d", i)},
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for range b.N {
		_ = buildDemoPrompt("my test question", demos)
	}
}

// BenchmarkBootstrapFewShot_Registry measures registry lookup + instantiation.
func BenchmarkBootstrapFewShot_Registry(b *testing.B) {
	b.ReportAllocs()
	for range b.N {
		_, err := NewOptimizer(string(StrategyBootstrapFewShot), OptimizerConfig{Seed: 42})
		if err != nil {
			b.Fatalf("NewOptimizer: %v", err)
		}
	}
}
