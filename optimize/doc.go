// Package optimize provides DSPy-style automated prompt/agent optimization for Beluga AI.
//
// Overview
//
// This package implements automated optimization of prompts and agent configurations
// using training data and user-defined metrics. The key insight from DSPy is that
// prompt engineering should be treated as an optimization problem, not a manual craft.
//
// Core Concepts
//
// Optimizer: The main abstraction that transforms an uncompiled program into an
// optimized version. Optimizers are registered by name and can be created via the
// registry:
//
//   - "bootstrapfewshot" — Bootstraps few-shot examples from training data
//   - "mipro" — Bayesian optimization with TPE sampler (coming soon)
//   - "gepa" — Genetic-Pareto prompt evolution (coming soon)
//   - "simba" — Stochastic introspective mini-batch ascent (coming soon)
//
// Metric: Evaluates prediction quality. Binary metrics (0/1) work best.
//
// Signature: Defines the input/output contract for programs, enabling type-safe
// optimization.
//
// Basic Usage
//
// Create optimizer via registry:
//
//	optimizer, err := optimize.NewOptimizer("bootstrapfewshot", optimize.OptimizerConfig{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Or create directly:
//
//	optimizer := optimizers.NewBootstrapFewShot(
//	    optimizers.WithMaxBootstrapped(4),
//	    optimizers.WithMetricThreshold(0.8),
//	)
//
// Compile (optimize) a program:
//
//	compiled, err := optimizer.Compile(ctx, program, optimize.CompileOptions{
//	    Trainset: trainset,
//	    Metric:   &metric.F1Metric{},
//	    MaxCost:  &optimize.CostBudget{MaxDollars: 10},
//	})
//
// Use the optimized program:
//
//	result, err := compiled.Run(ctx, inputs)
//
// Package Structure
//
//	optimize/
//	├── interfaces.go          # Core interfaces (Optimizer, Metric, etc.)
//	├── registry.go            # Optimizer registry for named creation
//	├── metric/                # Metric implementations
//	│   └── basic.go
//	├── optimizers/            # Optimizer implementations
//	│   ├── bootstrap_fewshot.go
//	│   ├── mipro.go
//	│   ├── gepa.go
//	│   └── simba.go
//	├── bayesian/              # TPE sampler for MIPRO
//	├── pareto/                # Pareto frontier for GEPA
//	├── cost/                  # Cost tracking
//	└── examples/              # Usage examples
//
// Design Patterns
//
// The optimize package follows Beluga AI design patterns:
//
//   - Registry pattern: Optimizers register via init() and are created by name
//   - Functional options: Configurable via WithXxx option functions
//   - Interface-based: Core abstractions are interfaces, not concrete types
//   - Thread-safe: Registry uses sync.RWMutex for concurrent access
//
// References
//
//   - DSPy: https://dspy.ai
//   - MIPROv2 paper: https://arxiv.org/abs/2406.11695
//   - GEPA paper: https://arxiv.org/abs/2507.19457
//
package optimize
