# Optimizer Performance Comparison

This document compares the four Beluga AI optimizers across the dimensions that
matter most when choosing between them: **accuracy improvement**, **compute cost**,
**convergence speed**, and **suitability by task type**.

## Optimizer Overview

| Optimizer | Algorithm | Best for | Typical trials |
|-----------|-----------|----------|----------------|
| **BootstrapFewShot** | Greedy teacher-student demo selection | Rapid iteration, cheap baselines | Equal to trainset size |
| **MIPROv2** | Bayesian TPE search over (instruction, demos) | Instruction-sensitive tasks, moderate budget | 10–50 |
| **GEPA** | Genetic evolution + Pareto multi-objective | Multi-objective trade-offs (accuracy vs cost) | PopSize × Generations |
| **SIMBA** | Stochastic mini-batch ascent with reflection | Large trainsets, diminishing-returns detection | 5–20 |

---

## Accuracy

### When optimizers improve accuracy the most

All optimizers improve accuracy when the *unoptimized* agent has meaningful
variance across examples (e.g. 50–80% baseline). If your baseline is already
≥95%, optimizers offer little additional gain.

| Scenario | Best optimizer |
|----------|---------------|
| Small trainset (< 20 examples) | **BootstrapFewShot** |
| Mid-size trainset, instruction matters | **MIPROv2** |
| Need accuracy *and* efficiency | **GEPA** |
| Large trainset (100+ examples) | **SIMBA** |

### Expected accuracy improvement

These are indicative ranges from internal experiments on classification,
extraction, and Q&A tasks. Real results depend heavily on the task and model.

| Optimizer | Typical accuracy lift |
|-----------|-----------------------|
| BootstrapFewShot | +5–20 pp |
| MIPROv2 | +10–25 pp |
| GEPA | +8–20 pp (accuracy objective) |
| SIMBA | +10–20 pp |

---

## Compute Cost

Cost is dominated by the number of `program.Run` calls (each maps to one LLM
inference call in production).

### Cost model

```
BootstrapFewShot:
  cost ≈ min(trainset_size, MaxBootstrapped) × MaxRounds

MIPROv2:
  cost ≈ NumTrials × MinibatchSize

GEPA:
  cost ≈ PopulationSize × MaxGenerations × ConsistencyRuns × EvalSampleSize

SIMBA:
  cost ≈ MaxIterations × (MinibatchSize × CandidatePoolSize + CandidatePoolSize/2)
```

### Example: 50 training examples, moderate budget

| Optimizer | Approx LLM calls |
|-----------|--------------------|
| BootstrapFewShot (defaults) | 4–16 |
| MIPROv2 (30 trials, batch=25) | 750 |
| GEPA (10×10, sample=10) | ~1 000 |
| SIMBA (15 iters, batch=20) | ~2 400 |

**BootstrapFewShot is the cheapest optimizer** by a large margin. Use it as the
baseline before running heavier optimizers.

### Budget controls

Every optimizer respects `CompileOptions.MaxCost`:

```go
opts := optimize.CompileOptions{
    MaxCost: &optimize.CostBudget{
        MaxIterations: 10,    // stop after 10 trials/generations/iterations
        MaxDollars:    1.0,   // stop if accumulated cost exceeds $1
    },
}
```

---

## Convergence Speed

### Wall-clock time (mock agent, no network latency)

| Optimizer | Defaults (50-example trainset) |
|-----------|-------------------------------|
| BootstrapFewShot | < 1 ms |
| MIPROv2 | 5–50 ms |
| GEPA | 10–100 ms |
| SIMBA | 5–50 ms |

With a real LLM (gpt-4o-mini, ~200 ms/call) multiply LLM-calls × 200 ms.

### Early stopping

**MIPROv2** and **SIMBA** implement convergence detection: if the rolling variance
of recent scores drops below `ConvergenceThreshold` (default 0.001) optimization
stops automatically.

**GEPA** also checks convergence via `ConvergenceChecker` (default window=5, threshold=1e-4).

---

## Qualitative Comparison

```
             │ Accuracy │ Cost  │ Speed │ Multi-obj │ Exploration
─────────────┼──────────┼───────┼───────┼───────────┼────────────
Bootstrap    │ Good     │ Low   │ Fast  │ No        │ Low
MIPROv2      │ Better   │ Med   │ Med   │ No        │ Bayesian
GEPA         │ Good     │ High  │ Slow  │ Yes       │ Genetic
SIMBA        │ Better   │ Med   │ Med   │ No        │ Stochastic
```

---

## Choosing an Optimizer

```
Is your trainset very small (< 10 examples)?
  → BootstrapFewShot

Do you have a strict dollar/token budget?
  → BootstrapFewShot or MIPROv2 with MaxCost

Do you need to optimise multiple objectives (accuracy, latency, cost)?
  → GEPA

Do you have a large trainset (100+ examples) and want introspective improvement?
  → SIMBA

Do you want the best accuracy with a moderate budget?
  → MIPROv2
```

---

## Reproducibility

All optimizers accept a `Seed int64` parameter. Setting the same seed produces
the same demo selections and trial orders across runs, making experiments
reproducible:

```go
compiled, err := optimizer.Compile(ctx, program, optimize.CompileOptions{
    Trainset: trainset,
    Metric:   myMetric,
    Seed:     42,
})
```

---

## References

- DSPy: https://dspy.ai
- MIPROv2 paper: https://arxiv.org/abs/2406.11695
- GEPA paper: https://arxiv.org/abs/2507.19457
- BootstrapFewShot: based on DSPy BootstrapFewShot algorithm
- SIMBA: Stochastic Introspective Mini-Batch Ascent (Beluga implementation)
