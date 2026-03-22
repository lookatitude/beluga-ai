# Optimize Package

DSPy-style automated prompt/agent optimization for Beluga AI.

## Quick Start

### Via Registry (Recommended)

```go
import "github.com/lookatitude/beluga-ai/optimize"
import "github.com/lookatitude/beluga-ai/optimize/metric"

// Register all optimizers
import _ "github.com/lookatitude/beluga-ai/optimize/optimizers"

// Create optimizer by name
optimizer, err := optimize.NewOptimizer("bootstrapfewshot", optimize.OptimizerConfig{})
if err != nil {
    log.Fatal(err)
}

// Compile (optimize) your program
compiled, err := optimizer.Compile(ctx, program, optimize.CompileOptions{
    Trainset: trainExamples,
    Metric:   &metric.F1Metric{},
})
```

### Direct Creation

```go
import "github.com/lookatitude/beluga-ai/optimize/optimizers"

// Create optimizer directly with options
optimizer := optimizers.NewBootstrapFewShot(
    optimizers.WithMaxBootstrapped(4),
    optimizers.WithMaxLabeled(16),
    optimizers.WithMetricThreshold(0.8),
)
```

### Unified Compiler API (Agent-Level)

For optimizing agents directly, use the `optimizer` package:

```go
import (
    optpkg "github.com/lookatitude/beluga-ai/optimizer"
    _ "github.com/lookatitude/beluga-ai/optimize/optimizers"
)

compiler := optpkg.CompilerForStrategy(optpkg.StrategyBootstrapFewShot)

optimized, err := compiler.Compile(ctx, myAgent,
    optpkg.WithMetric(containsMetric),
    optpkg.WithTrainsetExamples(trainset),
    optpkg.WithMaxIterations(10),
    optpkg.WithSeed(42),
)
```

## Available Optimizers

| Name | Registry Key | Description | Best For |
|------|-------------|-------------|----------|
| **BootstrapFewShot** | `bootstrapfewshot` | Greedily selects high-quality few-shot examples | Quick baseline, small datasets |
| **MIPROv2** | `mipro` | Bayesian optimization with TPE sampler | Highest quality, instruction-sensitive tasks |
| **GEPA** | `gepa` | Genetic-Pareto multi-objective prompt evolution | Multi-objective trade-offs (accuracy vs cost) |
| **SIMBA** | `simba` | Stochastic introspective mini-batch ascent | Large datasets, diminishing-returns detection |

## Metrics

- `metric.ExactMatch` — Binary exact match (1.0 or 0.0)
- `metric.F1Metric` — Token-based F1 score
- `metric.Contains` — Case-insensitive substring match
- `metric.MultiMetric` — Weighted combination of metrics

## Choosing an Optimizer

```
Is your trainset very small (< 10 examples)?
  → BootstrapFewShot

Do you have a strict dollar/token budget?
  → BootstrapFewShot or MIPROv2 with MaxCost

Do you need to optimise multiple objectives (accuracy, latency, cost)?
  → GEPA

Do you have a large trainset (100+ examples)?
  → SIMBA

Do you want the best accuracy with a moderate budget?
  → MIPROv2
```

## Design Patterns

The optimize package follows Beluga AI conventions:

- **Registry pattern**: Optimizers register via `init()` and are created by name
- **Functional options**: Configurable via `WithXxx` option functions
- **Interface-based**: Core abstractions are interfaces, not concrete types
- **Thread-safe**: Registry uses `sync.RWMutex` for concurrent access

## Testing

```bash
# Unit tests
go test ./optimize/...

# With race detection
go test -race ./optimize/...

# Integration tests (requires optimize/optimizers import)
go test -race ./optimizer/...
```

## References

- [DSPy Documentation](https://dspy.ai)
- [MIPROv2 Paper](https://arxiv.org/abs/2406.11695)
- [GEPA Paper](https://arxiv.org/abs/2507.19457)
- [BootstrapFewShot](https://dspy.ai) — based on DSPy BootstrapFewShot algorithm
- SIMBA — Stochastic Introspective Mini-Batch Ascent (Beluga implementation)
