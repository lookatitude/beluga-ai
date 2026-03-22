// Package optimizer provides the unified Compiler API for DSPy-style
// automated optimization of Beluga AI agents.
//
// # Overview
//
// The optimizer package is the recommended entry point for agent optimization.
// It bridges the agent framework to the low-level optimize/ package, providing
// a [Compiler] interface that accepts an [agent.Agent] and returns an optimized
// version with better prompts and demonstrations.
//
// Four optimization strategies are available:
//
//   - [StrategyBootstrapFewShot] — Greedily selects high-quality few-shot examples
//   - [StrategyMIPROv2] — Bayesian optimization with TPE sampler
//   - [StrategyGEPA] — Genetic-Pareto multi-objective prompt evolution
//   - [StrategySIMBA] — Stochastic introspective mini-batch ascent
//
// # Quick Start
//
// The simplest path is [CompilerForStrategy]:
//
//	import (
//	    optpkg "github.com/lookatitude/beluga-ai/optimizer"
//	    _ "github.com/lookatitude/beluga-ai/optimize/optimizers" // register all
//	)
//
//	compiler := optpkg.CompilerForStrategy(optpkg.StrategyBootstrapFewShot)
//
//	optimized, err := compiler.Compile(ctx, myAgent,
//	    optpkg.WithMetric(&optpkg.ContainsMetric{Field: "answer"}),
//	    optpkg.WithTrainsetExamples(trainset),
//	    optpkg.WithSeed(42),
//	)
//
// For detailed results including trial history:
//
//	result, err := compiler.CompileWithResult(ctx, myAgent,
//	    optpkg.WithMetric(metric),
//	    optpkg.WithTrainsetExamples(trainset),
//	    optpkg.WithMaxIterations(20),
//	    optpkg.WithCallback(myCallback),
//	)
//	fmt.Printf("Score: %.3f, Trials: %d\n", result.Score, len(result.Trials))
//
// # Registry Pattern
//
// Compilers and optimizers follow Beluga's standard registry pattern:
//
//	// By strategy name
//	compiler, err := optpkg.NewCompiler("bootstrap_few_shot", optpkg.CompilerConfig{})
//
//	// List available
//	names := optpkg.ListCompilers()   // ["bootstrap_few_shot", "gepa", "mipro_v2", "simba"]
//	names = optpkg.ListOptimizers()
//
// # Configuration
//
// Use functional options to configure the compilation process:
//
//	compiler.Compile(ctx, agent,
//	    optpkg.WithStrategy(optpkg.StrategyMIPROv2),
//	    optpkg.WithMetric(metric),
//	    optpkg.WithTrainsetExamples(trainset),
//	    optpkg.WithValsetExamples(valset),
//	    optpkg.WithMaxIterations(30),
//	    optpkg.WithMaxCost(5.00),
//	    optpkg.WithSeed(42),
//	    optpkg.WithNumWorkers(8),
//	    optpkg.WithCallback(progressCB),
//	)
//
// # Metrics
//
// Built-in metrics are available via the registry:
//
//	metric, _ := optpkg.NewMetric("exact_match", optpkg.MetricConfig{})
//	metric, _ = optpkg.NewMetric("contains", optpkg.MetricConfig{})
//
// Or use the concrete types directly:
//
//	metric := &optpkg.ExactMatchMetric{Field: "answer", CaseInsensitive: true}
//	metric := &optpkg.ContainsMetric{Field: "answer"}
//
// Custom metrics implement [Metric]:
//
//	metric := optpkg.MetricFunc(func(ctx context.Context, ex optpkg.Example, pred optpkg.Prediction) (float64, error) {
//	    expected, _ := ex.Outputs["answer"].(string)
//	    if strings.Contains(pred.Text, expected) {
//	        return 1.0, nil
//	    }
//	    return 0.0, nil
//	})
//
// # Callbacks
//
// Monitor progress with [Callback] or [CallbackFunc]:
//
//	cb := optpkg.CallbackFunc{
//	    OnProgressFunc: func(ctx context.Context, p optpkg.Progress) {
//	        fmt.Printf("Phase: %s, Trial: %d/%d\n", p.Phase, p.CurrentTrial, p.TotalTrials)
//	    },
//	    OnCompleteFunc: func(ctx context.Context, r optpkg.Result) {
//	        fmt.Printf("Done: score=%.3f\n", r.Score)
//	    },
//	}
//
// # Architecture
//
// The optimizer package acts as a bridge between the agent framework and the
// low-level optimize/ package:
//
//	optimizer.Compiler  →  optimizer.Optimizer  →  optimize.Optimizer
//	   (agent-level)         (bridge layer)         (program-level)
//
// The bridge layer (bridge.go) handles type conversion between agent.Agent and
// optimize.Program, metric adaptation, dataset mapping, and callback translation.
// Import optimize/optimizers to register all four algorithms.
//
// The package follows Beluga conventions:
//   - Registry pattern for extensibility
//   - Functional options for configuration
//   - Context-aware interfaces
//   - Thread-safe operations
package optimizer
