# Optimize Package

DSPy-style automated prompt/agent optimization for Beluga AI.

## Quick Start

```go
import "github.com/lookatitude/beluga-ai/optimize"
import "github.com/lookatitude/beluga-ai/optimize/metric"
import "github.com/lookatitude/beluga-ai/optimize/optimizers"

// Create optimizer
optimizer := optimizers.NewBootstrapFewShot(optimizers.BootstrapFewShotConfig{
    MaxBootstrapped: 4,
    MaxLabeled:      16,
})

// Compile (optimize) your program
compiled, err := optimizer.Compile(ctx, program, optimize.CompileOptions{
    Trainset: trainExamples,
    Metric:   &metric.F1Metric{},
})
```

## Optimizers

### BootstrapFewShot
Simplest optimizer. Bootstraps few-shot examples from training data.

```go
optimizer := optimizers.NewBootstrapFewShot(optimizers.BootstrapFewShotConfig{
    MaxBootstrapped: 4,  // Max auto-generated examples
    MaxLabeled:      16, // Max ground truth examples
})
```

### MIPROv2 (Coming Soon)
Bayesian optimization with TPE sampler.

### GEPA (Coming Soon)
Genetic-Pareto prompt evolution.

### SIMBA (Coming Soon)
Stochastic introspective mini-batch ascent.

## Metrics

- `metric.ExactMatch`: 1.0 if exact match, 0.0 otherwise
- `metric.F1Metric`: Token-based F1 score
- `metric.Contains`: Checks if output contains expected answer

## Testing

```bash
cd optimize
go test ./...
```

## References

- [DSPy Documentation](https://dspy.ai)
- [MIPROv2 Paper](https://arxiv.org/abs/2406.11695)
- [GEPA Paper](https://arxiv.org/abs/2507.19457)
