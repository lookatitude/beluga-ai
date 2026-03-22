# Migration Guide: From Manual Prompts to Automated Optimization

This guide helps you move from hand-crafted, static prompts to Beluga AI's
automated optimizer pipeline.

---

## What changes

| Before (manual) | After (optimized) |
|-----------------|-------------------|
| You write few-shot examples by hand | The optimizer discovers the best examples from your training data |
| You tune system instructions through trial-and-error | MIPROv2 / GEPA search the instruction space automatically |
| Improvements require redeploy | You rerun `Compile` on new training data and swap the compiled agent |
| No measurable quality signal | Every optimization run records metric scores per trial |

---

## Step 1: Identify your current prompt structure

Most hand-crafted prompts follow this pattern:

```
You are an expert at [task].
[Optional system instructions]

Examples:
Q: [example input 1]
A: [expected output 1]

Q: [example input 2]
A: [expected output 2]

Now answer:
Q: [user input]
A:
```

The migration maps these three parts to Beluga concepts:
- **System instructions** → `Persona.Goal` / `Persona.Backstory` (agent level)
- **Few-shot examples** → `optimize.Example` (training data)
- **Input/output schema** → `optimize.Signature` (inferred automatically)

---

## Step 2: Convert examples to training data

Take your hand-crafted examples and convert them to `[]optimize.Example`:

```go
// Before: hard-coded in the prompt string
// """
// Q: What is the capital of France?
// A: Paris
// """

// After: structured training data
trainset := []optimize.Example{
    {
        Inputs:  map[string]interface{}{"input": "What is the capital of France?"},
        Outputs: map[string]interface{}{"output": "Paris"},
    },
    {
        Inputs:  map[string]interface{}{"input": "What is the capital of Japan?"},
        Outputs: map[string]interface{}{"output": "Tokyo"},
    },
    // Add 5–20 more examples for good coverage
}
```

**Tip:** You need at least 4 examples for BootstrapFewShot and at least 6 for
MIPROv2/SIMBA. More examples produce better results.

---

## Step 3: Define a metric

Replace your subjective "does this look right?" evaluation with a programmatic
metric:

```go
// Simple contains check (good for Q&A, entity extraction)
metric := optimize.MetricFunc(func(ex optimize.Example, pred optimize.Prediction, _ *optimize.Trace) float64 {
    expected, _ := ex.Outputs["output"].(string)
    if strings.Contains(strings.ToLower(pred.Raw), strings.ToLower(expected)) {
        return 1.0
    }
    return 0.0
})

// Or use the built-in metrics
import "github.com/lookatitude/beluga-ai/optimize/metric"

exactMetric := optimize.MetricFunc(metric.ExactMatch)
containsMetric := optimize.MetricFunc(metric.Contains)
f1Metric := &metric.F1Metric{Field: "output"}
```

---

## Step 4: Replace static prompt with optimization

### Before (manual, static):

```go
agent := agent.New("my-agent",
    agent.WithLLM(model),
    agent.WithPersona(agent.Persona{
        Role:      "Expert assistant",
        Backstory: "Here are examples:\nQ: Capital of France?\nA: Paris\nQ: Capital of Japan?\nA: Tokyo",
    }),
)
```

### After (automated, via `agent.Optimize`):

```go
import _ "github.com/lookatitude/beluga-ai/optimize/optimizers"  // register optimizers

agent := agent.New("my-agent",
    agent.WithLLM(model),
    agent.WithPersona(agent.Persona{
        Role: "Expert assistant",
        // No manual examples in backstory — the optimizer will find them
    }),
    agent.WithOptimizer("bootstrapfewshot", optimize.OptimizerConfig{}),
    agent.WithTrainset(trainset),
    agent.WithMetric(metric),
    agent.WithOptimizationSeed(42),
)

optimized, err := agent.Optimize(context.Background())
// Use optimized instead of agent
result, err := optimized.Invoke(ctx, "What is the capital of Germany?")
```

### After (via `optimizer.Compiler`, more control):

```go
import (
    optpkg "github.com/lookatitude/beluga-ai/optimizer"
    _ "github.com/lookatitude/beluga-ai/optimize/optimizers" // register all optimizers
)

compiler := optpkg.CompilerForStrategy(optpkg.StrategyBootstrapFewShot)

optimizedAgent, err := compiler.Compile(ctx, baseAgent,
    optpkg.WithMetric(&optpkg.ContainsMetric{Field: "answer"}),
    optpkg.WithTrainsetExamples(trainset),
    optpkg.WithMaxIterations(10),
    optpkg.WithSeed(42),
)
// optimizedAgent implements agent.Agent — use it as a drop-in replacement
```

---

## Step 5: Persist and reload optimized programs

After optimization, the selected demos are baked into the agent's program.
To avoid re-running optimization on every deploy:

```go
// Save the optimized program state (TODO: implement serialization in your app)
// For now, cache the optimized agent in memory or re-run at startup.

// Pattern: run once at application start
var cachedOptimized *agent.OptimizedAgent
var once sync.Once

func getAgent(ctx context.Context) agent.Agent {
    once.Do(func() {
        a := agent.New("my-agent", /* ... */)
        opt, err := a.Optimize(ctx)
        if err == nil {
            cachedOptimized = opt
        }
    })
    if cachedOptimized != nil {
        return cachedOptimized
    }
    return baseAgent
}
```

---

## Step 6: Upgrade to stronger optimizers

Start with **BootstrapFewShot** (fast, cheap) and upgrade to **MIPROv2** or
**SIMBA** once you have a working baseline:

```go
// Level 1: Fast baseline (seconds)
compiler := optpkg.CompilerForStrategy(optpkg.StrategyBootstrapFewShot)

// Level 2: Better accuracy, moderate cost
compiler = optpkg.CompilerForStrategy(optpkg.StrategyMIPROv2)

// Level 3: Multi-objective (accuracy + efficiency)
compiler = optpkg.CompilerForStrategy(optpkg.StrategyGEPA)

// Level 4: Large trainset, stochastic exploration
compiler = optpkg.CompilerForStrategy(optpkg.StrategySIMBA)
```

---

## Common migration patterns

### From n-shot to BootstrapFewShot

If you currently pass `n` fixed examples, migrate them to the trainset and let
BootstrapFewShot select the best `MaxBootstrapped` of them.

```go
// Before: 3 hard-coded examples in prompt
// After:
import optim "github.com/lookatitude/beluga-ai/optimize/optimizers"

bs := optim.NewBootstrapFewShot(
    optim.WithMaxBootstrapped(3),   // match your old n-shot count
    optim.WithMaxLabeled(0),        // only keep teacher-verified examples
    optim.WithMetricThreshold(0.8), // accept if score >= 0.8
)
```

### From a custom system prompt to MIPROv2

If you've been hand-tuning the system instruction, MIPROv2 can search that space:

```go
// MIPROv2 will propose NumInstructionCandidates variations and pick the best.
mipro := optim.NewMIPROv2(
    optim.WithNumInstructionCandidates(8),
    optim.WithNumTrials(20),
)
```

Provide an LLM client to MIPROv2 for task-specific instruction generation:

```go
mipro := optim.NewMIPROv2(
    optim.WithMIPROv2LLM(myLLMClient), // generates task-grounded instructions
    optim.WithNumTrials(20),
)
```

---

## Metrics reference

| Use case | Recommended metric |
|----------|--------------------|
| Single-word answer (capitals, labels) | `metric.ExactMatch` |
| Partial answer OK (summaries, descriptions) | `metric.Contains` |
| Token-overlap (NLP tasks) | `&metric.F1Metric{}` |
| Pass@k (code generation) | `&metric.PassAtK{K: 5}` |
| Multiple signals combined | `&metric.MultiMetric{...}` |
| Custom logic | `optimize.MetricFunc(...)` |

---

## Troubleshooting

**Optimizer returns no demos / empty trainset error**

Ensure `Trainset` has at least 1 example before calling `Compile`.
BootstrapFewShot can compile with an empty trainset but will skip bootstrapping.

**All trial scores are 0.0**

Check that your `Metric` receives the correct field names. The `predict.Raw`
field contains the full LLM response; `predict.Outputs["output"]` is set when
using `AgentProgram`.

**Optimization is too slow**

- Use `MaxIterations` to cap early.
- Reduce `MinibatchSize` (MIPROv2, SIMBA) or `EvalSampleSize` (GEPA).
- Use BootstrapFewShot as a quick first pass.

**Results are not reproducible**

Pass a fixed `Seed` to `CompileOptions` and to the optimizer constructor.
