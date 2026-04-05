# Agent Optimization

Beluga AI brings **DSPy-style automated optimization** to Go agents. Instead of manually crafting prompts and selecting few-shot examples, you define a metric and let an optimizer systematically improve your agent's performance on training data.

## Why Optimize?

Hand-tuning prompts is brittle. A prompt that works for 80% of inputs often fails on edge cases, and small wording changes can cause unpredictable regressions. Optimization treats prompt engineering as a search problem:

1. **Define what "good" means** — a metric that scores agent outputs
2. **Provide examples** — input/output pairs representing desired behavior
3. **Let the optimizer search** — it systematically tries prompt variations and few-shot example selections, keeping what works

The result is an optimized agent that behaves identically to the original but with better prompts and demonstrations baked in.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "strings"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/optimize"
    "github.com/lookatitude/beluga-ai/optimize/metric"

    // Register the optimizer you want to use
    _ "github.com/lookatitude/beluga-ai/optimize/optimizers"
)

func main() {
    ctx := context.Background()

    // 1. Define training examples
    trainset := []optimize.Example{
        {
            Inputs:  map[string]interface{}{"input": "What is the capital of France?"},
            Outputs: map[string]interface{}{"output": "Paris"},
        },
        {
            Inputs:  map[string]interface{}{"input": "What is the capital of Japan?"},
            Outputs: map[string]interface{}{"output": "Tokyo"},
        },
        // ... more examples
    }

    // 2. Create an agent with optimization config
    a := agent.New("geography-expert",
        agent.WithLLM(model),  // your LLM instance
        agent.WithPersona(agent.Persona{
            Role: "Geography expert",
            Goal: "Answer geography questions accurately and concisely",
        }),
        agent.WithOptimizer("bootstrapfewshot", optimize.OptimizerConfig{}),
        agent.WithTrainset(trainset),
        agent.WithMetric(optimize.MetricFunc(metric.Contains)),
    )

    // 3. Run optimization
    optimized, err := a.Optimize(ctx)
    if err != nil {
        panic(err)
    }

    // 4. Use the optimized agent — same interface as the original
    result, err := optimized.Invoke(ctx, "What is the capital of Germany?")
    if err != nil {
        panic(err)
    }
    fmt.Println(result) // More likely to produce "Berlin" concisely
}
```

## How It Works

Optimization follows a **Compile** flow:

```
Agent → AgentProgram → Optimizer.Compile() → Optimized Program → OptimizedAgent
```

1. **Wrap**: The agent is wrapped as an `optimize.Program` via `agent.NewAgentProgram()`. This adapter translates between the agent's `Invoke`/`Stream` interface and the optimizer's `Run`/`WithDemos` interface.

2. **Compile**: The optimizer runs the program against training examples, evaluates results with your metric, and searches for better configurations (prompt wording, few-shot example selection, or both).

3. **Return**: The optimized program is wrapped back as an `OptimizedAgent` that implements the full `agent.Agent` interface. It's a drop-in replacement — `Invoke()`, `Stream()`, `Tools()`, `Children()` all work.

### What Gets Optimized

- **Few-shot demonstrations**: The optimizer selects which examples to include in the agent's context. These are injected into the persona's backstory as formatted input/output pairs.
- **Prompt instructions**: Advanced optimizers (MIPROv2, GEPA, SIMBA) also explore prompt variations — different phrasings, additional instructions, and structural changes.

### What Doesn't Change

- The agent's tools, children, and handoff configuration
- The LLM provider and model
- The planner strategy (ReAct, Reflexion, etc.)

## Optimizers

### BootstrapFewShot

**Best for**: Quick baseline optimization, small datasets, when you want fast results.

Bootstraps few-shot examples by running a teacher program on training data and keeping examples that pass the metric threshold. Simple but effective.

```go
// Via registry
a := agent.New("my-agent",
    agent.WithOptimizer("bootstrapfewshot", optimize.OptimizerConfig{}),
    agent.WithTrainset(trainset),
    agent.WithMetric(myMetric),
)

// Direct creation with options
import "github.com/lookatitude/beluga-ai/optimize/optimizers"

bs := optimizers.NewBootstrapFewShot(
    optimizers.WithMaxBootstrapped(4),   // max bootstrapped demos (default: 4)
    optimizers.WithMaxLabeled(16),       // max ground-truth demos (default: 16)
    optimizers.WithMaxRounds(1),         // attempts per example (default: 1)
    optimizers.WithMetricThreshold(1.0), // min score to accept (default: 1.0)
    optimizers.WithTeacher(teacherProg), // optional teacher program
)
```

**How it works**:
1. For each training example, run the teacher (or the student itself) to produce a prediction
2. Evaluate the prediction against the metric
3. If it passes the threshold, add it as a bootstrapped demonstration
4. Fill remaining slots with labeled (ground truth) examples
5. Return the student program with the selected demonstrations

**Configuration**:

| Option | Default | Description |
|--------|---------|-------------|
| `MaxBootstrapped` | 4 | Maximum bootstrapped (model-generated) examples |
| `MaxLabeled` | 16 | Maximum labeled (ground truth) examples |
| `MaxRounds` | 1 | Bootstrap attempts per training example |
| `MetricThreshold` | 1.0 | Minimum metric score to accept a demonstration |
| `Teacher` | nil | Teacher program (defaults to student if nil) |

### MIPROv2

**Best for**: Highest quality optimization, when you have compute budget, complex tasks.

Multi-step Instruction Proposal Optimization with Bayesian search using Tree-structured Parzen Estimators (TPE). Jointly optimizes prompt instructions and demonstration selection.

```go
a := agent.New("my-agent",
    agent.WithOptimizer("mipro", optimize.OptimizerConfig{}),
    agent.WithTrainset(trainset),
    agent.WithMetric(myMetric),
)

// Direct creation
import "github.com/lookatitude/beluga-ai/optimize/optimizers"

mipro := optimizers.NewMIPROv2(
    optimizers.WithNumTrials(30),                 // Bayesian trials (default: 30)
    optimizers.WithMinibatchSize(25),             // examples per eval (default: 25)
    optimizers.WithNumInstructionCandidates(5),   // instruction variants (default: 5)
    optimizers.WithNumDemoCandidates(5),          // demo subsets (default: 5)
    optimizers.WithMIPROv2Seed(42),               // reproducibility (default: 42)
)
```

**How it works**:
1. **Proposal phase**: Generate candidate instructions and demo subsets from training data
2. **Bayesian optimization**: For each trial, TPE samples instruction and demo indices based on past performance, evaluates on a mini-batch, and records the result
3. The TPE sampler learns which instruction/demo combinations work best, focusing search on promising regions

**Configuration**:

| Option | Default | Description |
|--------|---------|-------------|
| `NumTrials` | 30 | Number of Bayesian optimization trials |
| `MinibatchSize` | 25 | Training examples per trial evaluation |
| `NumInstructionCandidates` | 5 | Instruction variants to explore |
| `NumDemoCandidates` | 5 | Demo subset candidates |
| `Seed` | 42 | Random seed for reproducibility |

### GEPA

**Best for**: Complex prompt optimization, multi-objective trade-offs, creative prompt exploration.

Genetic-Pareto Prompt Optimizer uses evolutionary algorithms with Pareto frontier tracking to explore diverse prompt strategies.

```go
a := agent.New("my-agent",
    agent.WithOptimizer("gepa", optimize.OptimizerConfig{}),
    agent.WithTrainset(trainset),
    agent.WithMetric(myMetric),
)

// Direct creation
gepa := optimizers.NewGEPA(
    optimizers.WithPopulationSize(10),     // candidates per generation (default: 10)
    optimizers.WithMaxGenerations(10),     // evolution rounds (default: 10)
    optimizers.WithMutationRate(0.3),      // mutation probability (default: 0.3)
    optimizers.WithCrossoverRate(0.5),     // crossover probability (default: 0.5)
    optimizers.WithGEPASeed(42),           // reproducibility (default: 42)
)
```

**How it works**:
1. **Initialize** a population of candidates with random demo selections
2. **Evaluate** each candidate against training data
3. **Selection**: Tournament selection picks parents for reproduction
4. **Crossover**: Combine demos from two parents (50/50 mixing)
5. **Mutation**: Modify prompts and swap individual demos
6. **Archive**: Non-dominated solutions are preserved on the Pareto frontier
7. Repeat for `MaxGenerations`, then select the best from the archive

**Configuration**:

| Option | Default | Description |
|--------|---------|-------------|
| `PopulationSize` | 10 | Candidates per generation |
| `MaxGenerations` | 10 | Number of evolution rounds |
| `MutationRate` | 0.3 | Probability of mutating a candidate |
| `CrossoverRate` | 0.5 | Probability of crossover between parents |
| `ArchiveSize` | 50 | Maximum Pareto archive size |
| `ReflectionInterval` | 3 | Generations between reflection steps |
| `Seed` | 42 | Random seed for reproducibility |

### SIMBA

**Best for**: Large datasets, when you need adaptive exploration, stochastic environments.

Stochastic Introspective Mini-Batch Ascent uses mini-batch evaluation with introspective analysis to focus optimization on challenging examples.

```go
a := agent.New("my-agent",
    agent.WithOptimizer("simba", optimize.OptimizerConfig{}),
    agent.WithTrainset(trainset),
    agent.WithMetric(myMetric),
)

// Direct creation
simba := optimizers.NewSIMBA(
    optimizers.WithSIMBAMaxIterations(15),             // max optimization rounds (default: 15)
    optimizers.WithSIMBAMinibatchSize(20),             // examples per mini-batch (default: 20)
    optimizers.WithSIMBACandidatePoolSize(8),          // candidates to maintain (default: 8)
    optimizers.WithSIMBASamplingTemperature(0.2),      // softmax temperature (default: 0.2)
    optimizers.WithSIMBAConvergenceThreshold(0.001),   // early stopping (default: 0.001)
    optimizers.WithSIMBAMinVariabilityThreshold(0.3),  // challenge detection (default: 0.3)
    optimizers.WithSIMBASeed(42),                      // reproducibility (default: 42)
)
```

**How it works**:
1. **Initialize** a diverse candidate pool
2. Per iteration:
   - **Sample** a mini-batch using Fisher-Yates shuffle
   - **Evaluate** all candidates on the mini-batch
   - **Identify challenging examples** — those with high variance across candidates (different candidates disagree on them)
   - **Reflect** — generate improvement rules (e.g., "focus on challenging examples", "diversify prompts")
   - **Generate improved candidates** biased toward challenging examples
   - **Softmax select** the next candidate pool (temperature-controlled)
3. **Convergence check** — stop early if score variance drops below threshold

**Configuration**:

| Option | Default | Description |
|--------|---------|-------------|
| `MaxIterations` | 15 | Maximum optimization iterations |
| `MinibatchSize` | 20 | Training examples per mini-batch |
| `CandidatePoolSize` | 8 | Number of candidates to maintain |
| `SamplingTemperature` | 0.2 | Softmax selection temperature (lower = greedier) |
| `ConvergenceThreshold` | 0.001 | Score variance threshold for early stopping |
| `MinVariabilityThreshold` | 0.3 | Variance threshold for identifying challenging examples |
| `Seed` | 42 | Random seed for reproducibility |

## Choosing an Optimizer

| Criterion | BootstrapFewShot | MIPROv2 | GEPA | SIMBA |
|-----------|-----------------|---------|------|-------|
| **Speed** | Fast | Slow | Medium | Medium |
| **Quality** | Good baseline | Highest | High | High |
| **Compute cost** | Low | High | Medium | Medium |
| **Best dataset size** | 10-100 | 20-500 | 20-200 | 50-1000+ |
| **Optimizes prompts** | No (demos only) | Yes | Yes | Yes |
| **Handles complexity** | Simple tasks | Complex tasks | Multi-objective | Noisy/large data |

**Rules of thumb**:
- Start with **BootstrapFewShot** — it's fast and often sufficient
- Use **MIPROv2** when BootstrapFewShot plateaus and you need the best quality
- Use **GEPA** when you want to explore diverse prompt strategies or have multi-objective trade-offs
- Use **SIMBA** when your dataset is large or your task has high variance across examples

## Metrics

Metrics evaluate how well the agent's output matches the expected output. Higher scores are better.

### Built-in Metrics

```go
import "github.com/lookatitude/beluga-ai/optimize/metric"

// Exact match — 1.0 if all output fields match exactly
exactMetric := optimize.MetricFunc(metric.ExactMatch)

// Contains — 1.0 if prediction contains the expected answer (case-insensitive)
containsMetric := optimize.MetricFunc(metric.Contains)

// F1 score — token-level overlap between prediction and expected
f1Metric := &metric.F1Metric{
    Field: "output", // which field to compare (empty = all)
}

// Weighted combination of multiple metrics
combined := &metric.MultiMetric{
    Metrics: []optimize.Metric{
        optimize.MetricFunc(metric.Contains),
        &metric.F1Metric{},
    },
    Weights: []float64{0.7, 0.3}, // 70% contains, 30% F1
}
```

### Custom Metrics

Implement `optimize.Metric` or use `optimize.MetricFunc` for simple functions:

```go
// Using MetricFunc (no error return)
myMetric := optimize.MetricFunc(func(
    example optimize.Example,
    pred optimize.Prediction,
    trace *optimize.Trace,
) float64 {
    expected := example.Outputs["output"].(string)
    actual := pred.Outputs["output"].(string)

    // Your scoring logic — return 0.0 to 1.0
    if strings.TrimSpace(actual) == strings.TrimSpace(expected) {
        return 1.0
    }
    return 0.0
})

// Implementing the Metric interface (with error handling)
type SemanticSimilarity struct {
    embedder embedding.Embedder
    threshold float64
}

func (s *SemanticSimilarity) Evaluate(
    example optimize.Example,
    pred optimize.Prediction,
    trace *optimize.Trace,
) (float64, error) {
    expected := example.Outputs["output"].(string)
    actual := pred.Outputs["output"].(string)

    // Compute embedding similarity
    score, err := s.embedder.Similarity(expected, actual)
    if err != nil {
        return 0.0, err
    }
    return score, nil
}
```

**Tips for good metrics**:
- Binary metrics (0 or 1) tend to work best for optimization
- Avoid overly lenient metrics — they give the optimizer nowhere to improve
- Test your metric manually on a few examples before running optimization
- For classification tasks, use `ExactMatch`; for open-ended generation, use `Contains` or `F1`

## Budgets

Control optimization costs with `CostBudget`:

```go
a := agent.New("my-agent",
    agent.WithOptimizer("mipro", optimize.OptimizerConfig{}),
    agent.WithTrainset(trainset),
    agent.WithMetric(myMetric),
    agent.WithOptimizationBudget(optimize.CostBudget{
        MaxDollars:    5.00,  // stop after $5 spent
        MaxTokens:     500000, // stop after 500K tokens
        MaxIterations: 50,    // stop after 50 trials
    }),
)
```

Any limit being reached will stop the optimization. Set a field to `0` to disable that limit.

## Callbacks

Monitor optimization progress with callbacks:

```go
type ProgressLogger struct{}

func (p *ProgressLogger) OnTrialComplete(trial optimize.Trial) {
    fmt.Printf("Trial %d: score=%.3f cost=$%.4f duration=%dms\n",
        trial.ID, trial.Score, trial.Cost, trial.Duration)
    if trial.Error != nil {
        fmt.Printf("  error: %v\n", trial.Error)
    }
}

func (p *ProgressLogger) OnOptimizationComplete(result optimize.OptimizationResult) {
    fmt.Printf("Optimization complete: best=%.3f trials=%d cost=$%.4f\n",
        result.BestScore, result.NumTrials, result.TotalCost)
}

// Use it
a := agent.New("my-agent",
    // ... other options
    agent.WithOptimizationCallbacks(&ProgressLogger{}),
)
```

## Advanced Configuration

### Parallel Workers

Speed up optimization by evaluating multiple examples concurrently:

```go
agent.WithOptimizationWorkers(8) // default: 10
```

### Reproducibility

Set a seed for deterministic optimization runs:

```go
agent.WithOptimizationSeed(42)
```

### Validation Set

Provide a separate validation set for early stopping and generalization checking:

```go
agent.WithValset(validationExamples)
```

### Using the Optimizer Directly

You can use the optimize package without the agent integration:

```go
import (
    "github.com/lookatitude/beluga-ai/optimize"
    "github.com/lookatitude/beluga-ai/optimize/optimizers"
)

// Create an optimizer directly
bs := optimizers.NewBootstrapFewShot(
    optimizers.WithMaxBootstrapped(8),
)

// Compile any Program implementation
optimized, err := bs.Compile(ctx, myProgram, optimize.CompileOptions{
    Trainset: trainset,
    Metric:   myMetric,
})
```

### Registry Pattern

Optimizers use Beluga's standard registry pattern:

```go
import "github.com/lookatitude/beluga-ai/optimize"

// List available optimizers
names := optimize.ListOptimizers()
// ["bootstrapfewshot", "gepa", "mipro", "simba"]

// Create by name
opt, err := optimize.NewOptimizer("mipro", optimize.OptimizerConfig{})
```

## Best Practices

1. **Start simple**: Use `BootstrapFewShot` with `Contains` metric first. Only move to advanced optimizers if you need better results.

2. **Quality training data**: 20-50 diverse, representative examples are better than 1000 noisy ones. Cover edge cases and common patterns.

3. **Binary metrics first**: Metrics that return 0 or 1 give optimizers the clearest signal. Continuous metrics (like F1) work but may converge slower.

4. **Set budgets**: Always set a `CostBudget` when using MIPROv2 or GEPA to avoid unexpected API costs.

5. **Use callbacks**: Monitor optimization progress to catch issues early — if scores plateau immediately, your metric might be too lenient or too strict.

6. **Separate train/val**: Use `WithValset` to catch overfitting. If training scores are high but validation scores are low, you need more diverse training data.

7. **Reproducibility**: Set `WithOptimizationSeed` for deterministic runs when comparing configurations.

8. **Evaluate the optimized agent**: After optimization, test on held-out examples not in the training or validation set.

## Unified Compiler API

The `optimizer` package provides a higher-level **Compiler** interface that bridges the agent framework to the optimization algorithms. This is the recommended approach when optimizing agents directly.

### CompilerForStrategy

```go
import (
    optpkg "github.com/lookatitude/beluga-ai/optimizer"
    _ "github.com/lookatitude/beluga-ai/optimize/optimizers" // register all optimizers
)

// Create a compiler for any strategy
compiler := optpkg.CompilerForStrategy(optpkg.StrategyBootstrapFewShot)

// Compile returns an optimized agent (drop-in replacement)
optimized, err := compiler.Compile(ctx, myAgent,
    optpkg.WithMetric(&optpkg.ContainsMetric{Field: "answer"}),
    optpkg.WithTrainsetExamples(trainset),
    optpkg.WithMaxIterations(10),
    optpkg.WithSeed(42),
)
```

### CompileWithResult

For access to trial history, scores, and convergence status:

```go
result, err := compiler.CompileWithResult(ctx, myAgent,
    optpkg.WithMetric(metric),
    optpkg.WithTrainsetExamples(trainset),
    optpkg.WithValsetExamples(valset),
    optpkg.WithMaxIterations(20),
    optpkg.WithCallback(progressCallback),
)

fmt.Printf("Best score: %.3f\n", result.Score)
fmt.Printf("Trials: %d\n", len(result.Trials))
fmt.Printf("Duration: %s\n", result.TotalDuration)
fmt.Printf("Convergence: %s\n", result.ConvergenceStatus)
```

### Named Compiler via Registry

```go
// Create by strategy name
compiler, err := optpkg.NewCompiler("bootstrap_few_shot", optpkg.CompilerConfig{})

// List available compilers
names := optpkg.ListCompilers()
// ["bootstrap_few_shot", "gepa", "mipro_v2", "simba"]
```

### Available Strategies

| Strategy | Constant | Underlying Optimizer |
|----------|----------|---------------------|
| `"bootstrap_few_shot"` | `StrategyBootstrapFewShot` | `bootstrapfewshot` |
| `"mipro_v2"` | `StrategyMIPROv2` | `mipro` |
| `"gepa"` | `StrategyGEPA` | `gepa` |
| `"simba"` | `StrategySIMBA` | `simba` |

## Architecture

The optimization system is organized in two layers:

```
optimizer/                  High-level: agent-aware Compiler API
├── compiler.go            Compiler interface, CompilerForStrategy
├── bridge.go              Bridges optimizer.Optimizer ↔ optimize.Optimizer
├── bootstrap_fewshot.go   Direct BootstrapFewShot implementation
├── types.go               Result, Trial, Budget, Progress types
├── config.go              CompileOption functional options
├── metric.go              ExactMatchMetric, ContainsMetric
└── registry.go            Compiler/Optimizer/Metric registries

optimize/                   Low-level: program-level optimization
├── interfaces.go          Core types: Optimizer, Metric, Program, Signature
├── registry.go            Standard Register/New/List pattern
├── metric/
│   └── basic.go           ExactMatch, Contains, F1, MultiMetric
├── bayesian/
│   └── tpe.go             Tree-structured Parzen Estimator (for MIPROv2)
├── pareto/
│   └── frontier.go        Pareto frontier (for GEPA)
└── optimizers/
    ├── bootstrap_fewshot.go
    ├── mipro.go
    ├── gepa.go
    ├── simba.go
    └── simba_batch.go
```

The bridge layer (`optimizer/bridge.go`) handles:
- `agent.Agent` → `optimize.Program` adaptation via `bridgeProgram`
- `optimize.Program` → `agent.Agent` wrapping via `compiledAgent`
- Metric, dataset, and callback type conversion between the two packages

The `agent/optimize.go` file provides the agent-level integration:
- `AgentProgram` adapts `Agent` to `optimize.Program`
- `OptimizedAgent` wraps the result back as an `Agent`
- `WithOptimizer`, `WithTrainset`, `WithMetric`, etc. configure optimization via agent options

## References

- [DSPy: Compiling Declarative Language Model Calls](https://arxiv.org/abs/2310.03714) — the foundational paper
- [MIPROv2: Multi-step Instruction Proposal Optimization](https://arxiv.org/abs/2406.11695)
- [Tree-structured Parzen Estimators](https://proceedings.neurips.cc/paper/2011/hash/86e8f7ab32cfd12577bc2619bc635690-Abstract.html) — the Bayesian sampler behind MIPROv2
