// Package main demonstrates agent optimization using Beluga AI's DSPy-style
// optimization system. It shows how to define training data, set up metrics,
// run optimization with different optimizers, and compare results.
//
// This example optimizes a simple Q&A agent on geography questions.
//
// Usage:
//
//	export OPENAI_API_KEY=your-key
//	go run .
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/lookatitude/beluga-ai/agent"
	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/optimize"
	"github.com/lookatitude/beluga-ai/optimize/metric"

	// Register the LLM provider
	_ "github.com/lookatitude/beluga-ai/llm/providers/openai"

	// Register all optimizers
	_ "github.com/lookatitude/beluga-ai/optimize/optimizers"
)

// trainingData returns geography Q&A examples for optimization.
func trainingData() []optimize.Example {
	return []optimize.Example{
		{
			Inputs:  map[string]interface{}{"input": "What is the capital of France?"},
			Outputs: map[string]interface{}{"output": "Paris"},
		},
		{
			Inputs:  map[string]interface{}{"input": "What is the capital of Japan?"},
			Outputs: map[string]interface{}{"output": "Tokyo"},
		},
		{
			Inputs:  map[string]interface{}{"input": "What is the capital of Brazil?"},
			Outputs: map[string]interface{}{"output": "Brasília"},
		},
		{
			Inputs:  map[string]interface{}{"input": "What is the capital of Australia?"},
			Outputs: map[string]interface{}{"output": "Canberra"},
		},
		{
			Inputs:  map[string]interface{}{"input": "What is the capital of Egypt?"},
			Outputs: map[string]interface{}{"output": "Cairo"},
		},
		{
			Inputs:  map[string]interface{}{"input": "What is the capital of Canada?"},
			Outputs: map[string]interface{}{"output": "Ottawa"},
		},
		{
			Inputs:  map[string]interface{}{"input": "What is the capital of South Korea?"},
			Outputs: map[string]interface{}{"output": "Seoul"},
		},
		{
			Inputs:  map[string]interface{}{"input": "What is the capital of Italy?"},
			Outputs: map[string]interface{}{"output": "Rome"},
		},
		{
			Inputs:  map[string]interface{}{"input": "What is the capital of Thailand?"},
			Outputs: map[string]interface{}{"output": "Bangkok"},
		},
		{
			Inputs:  map[string]interface{}{"input": "What is the capital of Nigeria?"},
			Outputs: map[string]interface{}{"output": "Abuja"},
		},
		{
			Inputs:  map[string]interface{}{"input": "What is the capital of Turkey?"},
			Outputs: map[string]interface{}{"output": "Ankara"},
		},
		{
			Inputs:  map[string]interface{}{"input": "What is the capital of Myanmar?"},
			Outputs: map[string]interface{}{"output": "Naypyidaw"},
		},
	}
}

// testQuestions are held-out questions for evaluating the optimized agent.
func testQuestions() []optimize.Example {
	return []optimize.Example{
		{
			Inputs:  map[string]interface{}{"input": "What is the capital of Germany?"},
			Outputs: map[string]interface{}{"output": "Berlin"},
		},
		{
			Inputs:  map[string]interface{}{"input": "What is the capital of Spain?"},
			Outputs: map[string]interface{}{"output": "Madrid"},
		},
		{
			Inputs:  map[string]interface{}{"input": "What is the capital of India?"},
			Outputs: map[string]interface{}{"output": "New Delhi"},
		},
	}
}

// progressCallback logs optimization progress to stdout.
type progressCallback struct{}

func (p *progressCallback) OnTrialComplete(trial optimize.Trial) {
	status := "ok"
	if trial.Error != nil {
		status = fmt.Sprintf("error: %v", trial.Error)
	}
	fmt.Printf("  Trial %d: score=%.3f (%s)\n", trial.ID, trial.Score, status)
}

func (p *progressCallback) OnOptimizationComplete(result optimize.OptimizationResult) {
	fmt.Printf("  Done: best_score=%.3f trials=%d\n\n", result.BestScore, result.NumTrials)
}

func main() {
	ctx := context.Background()

	// Create the LLM
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Set OPENAI_API_KEY to run this example")
		os.Exit(1)
	}

	model, err := llm.New("openai", config.ProviderConfig{
		Provider: "openai",
		APIKey:   apiKey,
		Model:    "gpt-4o-mini",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create LLM: %v\n", err)
		os.Exit(1)
	}

	trainset := trainingData()
	testset := testQuestions()

	// Define a metric: the agent's response must contain the expected answer.
	containsMetric := optimize.MetricFunc(metric.Contains)

	// ------------------------------------------------------------------
	// Step 1: Evaluate the unoptimized agent
	// ------------------------------------------------------------------
	fmt.Println("=== Unoptimized Agent ===")

	baseAgent := agent.New("geography-qa",
		agent.WithLLM(model),
		agent.WithPersona(agent.Persona{
			Role: "Geography expert",
			Goal: "Answer geography questions with just the city name",
		}),
	)

	score := evaluate(ctx, baseAgent, testset, containsMetric)
	fmt.Printf("Test score: %.1f%%\n\n", score*100)

	// ------------------------------------------------------------------
	// Step 2: Optimize with BootstrapFewShot
	// ------------------------------------------------------------------
	fmt.Println("=== Optimizing with BootstrapFewShot ===")

	bsAgent := agent.New("geography-qa-bs",
		agent.WithLLM(model),
		agent.WithPersona(agent.Persona{
			Role: "Geography expert",
			Goal: "Answer geography questions with just the city name",
		}),
		agent.WithOptimizer("bootstrapfewshot", optimize.OptimizerConfig{}),
		agent.WithTrainset(trainset),
		agent.WithMetric(containsMetric),
		agent.WithOptimizationCallbacks(&progressCallback{}),
		agent.WithOptimizationSeed(42),
	)

	optimizedBS, err := bsAgent.Optimize(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "BootstrapFewShot optimization failed: %v\n", err)
		os.Exit(1)
	}

	score = evaluate(ctx, optimizedBS, testset, containsMetric)
	fmt.Printf("Test score (BootstrapFewShot): %.1f%%\n\n", score*100)

	// ------------------------------------------------------------------
	// Step 3: Optimize with SIMBA
	// ------------------------------------------------------------------
	fmt.Println("=== Optimizing with SIMBA ===")

	simbaAgent := agent.New("geography-qa-simba",
		agent.WithLLM(model),
		agent.WithPersona(agent.Persona{
			Role: "Geography expert",
			Goal: "Answer geography questions with just the city name",
		}),
		agent.WithOptimizer("simba", optimize.OptimizerConfig{}),
		agent.WithTrainset(trainset),
		agent.WithMetric(containsMetric),
		agent.WithOptimizationBudget(optimize.CostBudget{
			MaxIterations: 10,
		}),
		agent.WithOptimizationCallbacks(&progressCallback{}),
		agent.WithOptimizationSeed(42),
	)

	optimizedSIMBA, err := simbaAgent.Optimize(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SIMBA optimization failed: %v\n", err)
		os.Exit(1)
	}

	score = evaluate(ctx, optimizedSIMBA, testset, containsMetric)
	fmt.Printf("Test score (SIMBA): %.1f%%\n\n", score*100)

	// ------------------------------------------------------------------
	// Step 4: Compare results
	// ------------------------------------------------------------------
	fmt.Println("=== Comparison ===")
	fmt.Println("Asking: 'What is the capital of Germany?'")
	fmt.Println()

	for _, a := range []struct {
		name  string
		agent agent.Agent
	}{
		{"Unoptimized", baseAgent},
		{"BootstrapFewShot", optimizedBS},
		{"SIMBA", optimizedSIMBA},
	} {
		result, err := a.agent.Invoke(ctx, "What is the capital of Germany?")
		if err != nil {
			fmt.Printf("  %-20s error: %v\n", a.name+":", err)
			continue
		}
		fmt.Printf("  %-20s %s\n", a.name+":", truncate(result, 80))
	}
}

// evaluate runs the agent on test examples and returns the average metric score.
func evaluate(ctx context.Context, a agent.Agent, examples []optimize.Example, m optimize.Metric) float64 {
	var total float64
	for _, ex := range examples {
		input := ex.Inputs["input"].(string)
		result, err := a.Invoke(ctx, input)
		if err != nil {
			continue
		}

		pred := optimize.Prediction{
			Outputs: map[string]interface{}{"output": result},
			Raw:     result,
		}

		score, err := m.Evaluate(ex, pred, nil)
		if err != nil {
			continue
		}
		total += score
	}
	return total / float64(len(examples))
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
