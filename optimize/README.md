# Optimize Package

DSPy-style automated prompt/agent optimization for Beluga AI.

## Quick Start

### Via Registry (Recommended)

```go
import "github.com/lookatitude/beluga-ai/optimize"
import "github.com/lookatitude/beluga-ai/optimize/metric"

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

## Available Optimizers

| Name | Description | Status |
|------|-------------|--------|
| `bootstrapfewshot` | Bootstraps few-shot examples from training data | ✅ Ready |
| `mipro` | Bayesian optimization with TPE sampler | 🚧 Coming Soon |
| `gepa` | Genetic-Pareto prompt evolution | 🚧 Coming Soon |
| `simba` | Stochastic introspective mini-batch ascent | 🚧 Coming Soon |

## Metrics

- `metric.ExactMatch` — Binary exact match (1.0 or 0.0)
- `metric.F1Metric` — Token-based F1 score
- `metric.Contains` — Case-insensitive substring match
- `metric.MultiMetric` — Weighted combination of metrics

## Design Patterns

The optimize package follows Beluga AI conventions:

- **Registry pattern**: Optimizers register via `init()` and are created by name
- **Functional options**: Configurable via `WithXxx` option functions
- **Interface-based**: Core abstractions are interfaces, not concrete types
- **Thread-safe**: Registry uses `sync.RWMutex` for concurrent access

## Testing

```bash
cd optimize
go test ./...
```

## References

- [DSPy Documentation](https://dspy.ai)
- [MIPROv2 Paper](https://arxiv.org/abs/2406.11695)
- [GEPA Paper](https://arxiv.org/abs/2507.19457)
