// Package optimize provides DSPy-style automated prompt/agent optimization for Beluga AI.
//
// # Overview
//
// This package implements automated optimization of prompts and agent configurations
// using training data and user-defined metrics. The key insight from DSPy is that
// prompt engineering should be treated as an optimization problem, not a manual craft.
//
// # Core Concepts
//
// [Optimizer]: The main abstraction that transforms an uncompiled program into an
// optimized version. Four optimizers are registered by name and created via the
// registry:
//
//   - "bootstrapfewshot" — Greedily selects high-quality few-shot examples
//   - "mipro" — Bayesian optimization with TPE sampler for joint instruction/demo search
//   - "gepa" — Genetic-Pareto multi-objective prompt evolution
//   - "simba" — Stochastic introspective mini-batch ascent for large datasets
//
// [Metric]: Evaluates prediction quality. Higher scores are better; binary
// metrics (0/1) work best for optimization.
//
// [Program]: An optimizable unit that the optimizer can compile. Agents are
// adapted to this interface via AgentProgram.
//
// [Signature]: Defines the input/output contract for programs, enabling
// type-safe optimization.
//
// # Basic Usage
//
// Create optimizer via registry:
//
//	import _ "github.com/lookatitude/beluga-ai/optimize/optimizers" // register all
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
// # Unified Compiler API
//
// For agent-level optimization, the optimizer package provides a higher-level
// [Compiler] API that bridges this package to the agent framework:
//
//	import optpkg "github.com/lookatitude/beluga-ai/optimizer"
//
//	compiler := optpkg.CompilerForStrategy(optpkg.StrategyBootstrapFewShot)
//	optimized, err := compiler.Compile(ctx, myAgent,
//	    optpkg.WithMetric(metric),
//	    optpkg.WithTrainsetExamples(trainset),
//	)
//
// # Package Structure
//
//	optimize/
//	├── interfaces.go          # Core interfaces (Optimizer, Metric, etc.)
//	├── registry.go            # Optimizer registry for named creation
//	├── metric/                # Metric implementations (ExactMatch, F1, etc.)
//	├── optimizers/            # Optimizer implementations
//	│   ├── bootstrap_fewshot.go
//	│   ├── mipro.go
//	│   ├── gepa.go
//	│   └── simba.go
//	├── bayesian/              # Tree-structured Parzen Estimator for MIPROv2
//	├── pareto/                # Pareto frontier for GEPA
//	└── cost/                  # Cost tracking
//
// # Design Patterns
//
// The optimize package follows Beluga AI design patterns:
//
//   - Registry pattern: Optimizers register via init() and are created by name
//   - Functional options: Configurable via WithXxx option functions
//   - Interface-based: Core abstractions are interfaces, not concrete types
//   - Thread-safe: Registry uses sync.RWMutex for concurrent access
//
// # References
//
//   - DSPy: https://dspy.ai
//   - BootstrapFewShot: based on DSPy BootstrapFewShot algorithm
//   - MIPROv2 paper: https://arxiv.org/abs/2406.11695
//   - GEPA paper: https://arxiv.org/abs/2507.19457
//   - SIMBA: Stochastic Introspective Mini-Batch Ascent (Beluga implementation)
package optimize
