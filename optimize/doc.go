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
// optimized version. The package includes four optimizer implementations:
//   - BootstrapFewShot: Simplest optimizer, bootstraps few-shot examples
//   - MIPROv2: Bayesian optimization with TPE sampler
//   - GEPA: Genetic-Pareto prompt evolution (state-of-the-art)
//   - SIMBA: Stochastic introspective mini-batch ascent
//
// Metric: Evaluates prediction quality. Binary metrics (0/1) work best.
//
// Signature: Defines the input/output contract for programs, enabling type-safe
// optimization.
//
// Basic Usage
//
//	// Define a program
//	qa := &ChainOfThought{
//	    Signature: MustSignature("question,context -> answer"),
//	}
//
//	// Define metric
//	metric := &F1Metric{Tokenizer: WordTokenizer}
//
//	// Create optimizer
//	optimizer := optimize.NewBootstrapFewShot(optimize.BootstrapFewShotConfig{
//	    MaxBootstrapped: 4,
//	})
//
//	// Compile (optimize) the program
//	compiled, err := optimizer.Compile(ctx, qa, optimize.CompileOptions{
//	    Trainset: trainset,
//	    Metric:   metric,
//	    MaxCost:  &optimize.CostBudget{MaxDollars: 10},
//	})
//
//	// Use the optimized program
//	answer, err := compiled.Run(ctx, map[string]interface{}{
//	    "question": "What is the capital of France?",
//	    "context":  "France is a country in Europe.",
//	})
//
// Package Structure
//
//	optimize/
//	├── interfaces.go          # Core interfaces (Optimizer, Metric, etc.)
//	├── metric/                # Metric implementations
//	│   └── basic.go
//	├── optimizers/            # Optimizer implementations
//	│   ├── bootstrap_fewshot.go
//	│   ├── mipro.go
//	│   ├── gepa.go
//	│   └── simba.go
//	├── bayesian/              # TPE sampler for MIPROv2
//	├── pareto/                # Pareto frontier for GEPA
//	├── cost/                  # Cost tracking
//	└── examples/              # Usage examples
//
// References
//
//   - DSPy: https://dspy.ai
//   - MIPROv2 paper: https://arxiv.org/abs/2406.11695
//   - GEPA paper: https://arxiv.org/abs/2507.19457
//
package optimize
