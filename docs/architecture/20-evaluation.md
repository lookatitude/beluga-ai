# DOC-20: Evaluation Framework

**Audience:** ML engineers and CI pipeline builders who want to measure and gate agent quality.
**Prerequisites:** [01 — Architecture Overview](./01-overview.md), [03 — Extensibility Patterns](./03-extensibility-patterns.md).
**Related:** [14 — Observability](./14-observability.md), [05 — Agent Anatomy](./05-agent-anatomy.md).

## Overview

The `eval` package provides a structured, CI-integrable framework for measuring the quality, safety, cost, and behavioural correctness of Beluga AI agents. Rather than requiring a separate evaluation platform, `eval` treats quality measurement as a first-class framework concern: evaluation metrics implement the same `Metric` interface and run inside the same `EvalRunner` regardless of whether they execute locally, call an LLM-as-judge, or delegate to an external platform like Ragas, Braintrust, or DeepEval.

The central abstraction is `eval.Metric` — a single interface with two methods (`Name()` and `Score()`). Every evaluation surface in the framework — per-call quality metrics, LLM-as-judge rubrics, trajectory analysis, simulated user interactions, and adversarial red-teaming — converges to this interface. That uniformity means you can compose heterogeneous metrics into a single `EvalRunner.Run()` call and get a unified `EvalReport`.

CI integration is a design goal, not an afterthought. `EvalRunner` supports configurable parallelism, timeouts, and stop-on-error; `EvalReport` surfaces per-metric average scores suitable for threshold gates; and datasets are plain JSON files (`eval.LoadDataset`) that version-control alongside test fixtures.

## The five evaluation surfaces

```
┌────────────────────────────────────────────────────────────────────┐
│  eval.Metric (the universal interface)                             │
├────────────────────────────┬───────────────────────────────────────┤
│  Per-call metrics          │  Single (input, output) pair          │
│  eval/metrics/             │  faithfulness, relevance,             │
│                            │  hallucination, toxicity,             │
│                            │  latency, cost                        │
├────────────────────────────┼───────────────────────────────────────┤
│  LLM-as-Judge              │  Rubric-based scoring with            │
│  eval/judge/               │  weighted criteria and cross-model    │
│                            │  consistency checking                 │
├────────────────────────────┼───────────────────────────────────────┤
│  Trajectory evaluation     │  Agent execution traces:              │
│  eval/trajectory/          │  tool selection, step efficiency,     │
│                            │  goal completion                      │
├────────────────────────────┼───────────────────────────────────────┤
│  Simulated user            │  Multi-turn episodes driven by        │
│  eval/simulation/          │  an LLM persona with a stated goal    │
├────────────────────────────┼───────────────────────────────────────┤
│  Red team                  │  7 attack categories, static          │
│  eval/redteam/             │  patterns + LLM-generated attacks,    │
│                            │  per-category defense scoring         │
└────────────────────────────┴───────────────────────────────────────┘
```

## Per-call metrics — `eval/metrics`

Each type in `eval/metrics` implements `eval.Metric` and scores a single `EvalSample`. All scores are in `[0.0, 1.0]` where a higher score is better, with one exception noted below.

| Type | Name | Requires | What it measures |
|---|---|---|---|
| `Faithfulness` | `"faithfulness"` | LLM judge, `RetrievedDocs` | Whether the answer is grounded in the retrieved documents |
| `Relevance` | `"relevance"` | LLM judge | Whether the answer addresses the question |
| `Hallucination` | `"hallucination"` | LLM judge, `RetrievedDocs` | Absence of fabricated claims (1.0 = no hallucination) |
| `Toxicity` | `"toxicity"` | keyword list | Absence of toxic content (1.0 = not toxic) |
| `Latency` | `"latency"` | `Metadata["latency_ms"]` | Normalized response speed (1.0 = instant) |
| `Cost` (metrics) | `"cost"` | `Metadata["model","input_tokens","output_tokens"]` | Raw dollar cost (not normalized — use `eval/cost` for normalized) |

The `Faithfulness`, `Relevance`, and `Hallucination` metrics use an LLM-as-judge pattern: they construct a structured prompt and parse a single float from the response (`eval/metrics/faithfulness.go:47-58`). `Toxicity` uses a configurable keyword list and needs no external model (`eval/metrics/toxicity.go:58-74`). `Latency` reads `Metadata["latency_ms"]` and normalises against a configurable maximum (`eval/metrics/latency.go:49-68`).

Constructors use functional options:

```go
import (
    "context"
    "fmt"
    "log"

    _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/eval"
    "github.com/lookatitude/beluga-ai/eval/metrics"
    "github.com/lookatitude/beluga-ai/llm"
)

func main() {
    judge, err := llm.New("anthropic", config.ProviderConfig{
        Provider: "anthropic",
        Model:    "claude-3-haiku-20240307",
    })
    if err != nil {
        log.Fatal(err)
    }

    faithfulness := metrics.NewFaithfulness(judge)
    relevance := metrics.NewRelevance(judge)
    toxicity := metrics.NewToxicity(
        metrics.WithKeywords([]string{"hate", "violence"}),
        metrics.WithThreshold(0.5),
    )
    latency := metrics.NewLatency(metrics.WithMaxLatencyMs(5000))

    sample := eval.EvalSample{
        Input:  "What is the capital of France?",
        Output: "The capital of France is Paris.",
        Metadata: map[string]any{"latency_ms": float64(320)},
    }

    score, err := faithfulness.Score(context.Background(), sample)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("faithfulness: %.2f\n", score)

    score, err = relevance.Score(context.Background(), sample)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("relevance: %.2f\n", score)

    score, err = toxicity.Score(context.Background(), sample)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("toxicity: %.2f\n", score)

    score, err = latency.Score(context.Background(), sample)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("latency: %.2f\n", score)
}
```

## LLM-as-Judge — `eval/judge`

The `eval/judge` package provides rubric-based evaluation using one or more LLM judges. It extends the basic LLM-judge approach in `eval/metrics` with structured `Rubric` definitions, cross-model consistency checking, and batched concurrent evaluation.

### Core types

**`Rubric`** (`eval/judge/rubric.go:40`) defines the evaluation dimensions. Each `Criterion` has a `Name`, `Weight`, `Description`, and a list of `ScoreLevel` values that the LLM selects from. `Rubric.Validate()` checks non-empty names, positive weights, and score levels in `[0, 1]`. `Rubric.ToPrompt()` renders the rubric as structured text for the judge prompt.

**`JudgeMetric`** (`eval/judge/judge.go:77`) implements `eval.Metric`. It constructs the judge prompt, calls the LLM, parses per-criterion scores, and returns a weighted average. Use `NewJudgeMetric(WithModel(m), WithRubric(r))`.

**`ConsistencyChecker`** (`eval/judge/consistency.go:76`) runs the same rubric with multiple LLM models and/or multiple repeats to measure scoring reliability. It returns `ConsistencyResult` with `MeanScore`, `StdDev`, and `Agreement` (fraction of score pairs within 0.1 of each other).

**`BatchJudge`** (`eval/judge/batch.go:55`) evaluates multiple samples concurrently with bounded parallelism using a single `JudgeMetric`.

```go
import (
    "context"
    "fmt"
    "log"

    _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/eval"
    "github.com/lookatitude/beluga-ai/eval/judge"
    "github.com/lookatitude/beluga-ai/llm"
)

func main() {
    model, err := llm.New("anthropic", config.ProviderConfig{
        Provider: "anthropic",
        Model:    "claude-3-haiku-20240307",
    })
    if err != nil {
        log.Fatal(err)
    }

    rubric := &judge.Rubric{
        Name:        "customer-support",
        Description: "Evaluate a customer support response.",
        Criteria: []judge.Criterion{
            {
                Name: "accuracy", Weight: 2.0,
                Description: "Is the answer factually correct?",
                Levels: []judge.ScoreLevel{
                    {Label: "correct", Score: 1.0, Description: "Fully correct."},
                    {Label: "incorrect", Score: 0.0, Description: "Contains errors."},
                },
            },
            {
                Name: "clarity", Weight: 1.0,
                Description: "Is the response clear and concise?",
                Levels: []judge.ScoreLevel{
                    {Label: "clear", Score: 1.0, Description: "Easy to understand."},
                    {Label: "unclear", Score: 0.0, Description: "Confusing."},
                },
            },
        },
    }

    metric, err := judge.NewJudgeMetric(
        judge.WithModel(model),
        judge.WithRubric(rubric),
        judge.WithMetricName("support-quality"),
    )
    if err != nil {
        log.Fatal(err)
    }

    sample := eval.EvalSample{
        Input:  "How do I reset my password?",
        Output: "Click 'Forgot Password' on the login page and follow the instructions.",
    }

    score, err := metric.Score(context.Background(), sample)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("support-quality: %.2f\n", score)
}
```

## Trajectory evaluation — `eval/trajectory`

Agent trajectories record the ordered sequence of planning, tool-call, respond, handoff, and finish steps that an agent executed. The `eval/trajectory` package evaluates these traces to detect inefficiency, wrong tool choices, and goal-completion failures.

### Core types

**`Trajectory`** (`eval/trajectory/trajectory.go:68`) is the complete execution trace: `AgentID`, `Input`, `Output`, `ExpectedOutput`, `ExpectedTools`, and an ordered slice of `Step`. Each `Step` carries `Type` (`plan`, `tool_call`, `respond`, `handoff`, `finish`), `Action`, `Result`, `Latency`, and `Metadata`. `Trajectory.ActualTools()` returns the deduplicated list of tools used.

**`TrajectoryMetric`** (`eval/trajectory/metric.go:11`) is the evaluation interface:

```go
type TrajectoryMetric interface {
    Name() string
    ScoreTrajectory(ctx context.Context, t Trajectory) (*TrajectoryScore, error)
}
```

`TrajectoryScore` carries an `Overall` score in `[0, 1]`, per-step `StepScores`, and a `Details` map for metric-specific data such as precision and recall on tool selection.

**`trajectory.Runner`** (`eval/trajectory/runner.go:65`) evaluates a set of trajectories against configured metrics with bounded concurrency and lifecycle hooks.

**Registry** (`eval/trajectory/registry.go`): custom trajectory metrics register via `trajectory.Register(name, factory)` and are instantiated with `trajectory.New(name, cfg)`.

Serialise and load trajectories with `trajectory.SaveTrajectories` / `trajectory.LoadTrajectories` (both validate paths against `..` traversal).

## Simulated user — `eval/simulation`

Simulation-based evaluation measures an agent's ability to complete realistic goals over multi-turn conversations. An LLM-driven `SimulatedUser` takes on a persona and goal, generates realistic user turns, and signals `[GOAL_COMPLETE]` or `[GOAL_FAILED]` when the episode concludes.

### Core types

**`SimulatedUser`** (`eval/simulation/user.go:72`) wraps an `llm.ChatModel` with a persona and goal. `Respond(ctx, agentResponse)` generates the next user turn. `Reset()` clears conversation history for the next episode.

**`SimEnvironment`** (`eval/simulation/environment.go:23`) is an optional interface for stateful environments (e.g., a form-filling scenario). It has `Reset`, `Step`, `Observe`, and `Close`. Pass it via `WithEnvironment`.

**`SimRunner`** (`eval/simulation/runner.go:116`) drives episodes. `RunEpisode(ctx)` runs one episode; `Run(ctx, episodes)` runs N episodes and returns a `SimReport` with `SuccessRate` and `AverageTurns`.

**`AgentFunc`** (`eval/simulation/runner.go:14`) decouples the runner from any specific agent implementation:

```go
type AgentFunc func(ctx context.Context, userMessage string) (string, error)
```

Wrap your agent's `Invoke()` call to satisfy this type.

```go
import (
    "context"
    "fmt"
    "log"

    _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/eval/simulation"
    "github.com/lookatitude/beluga-ai/llm"
)

func main() {
**`AttackCategory`** (`eval/redteam/types.go:6`) — typed constant for each attack class. `AllCategories()` returns all seven.
        Provider: "anthropic",
        Model:    "claude-3-haiku-20240307",
    })
    if err != nil {
        log.Fatal(err)
    }

    user, err := simulation.NewSimulatedUser(
        simulation.WithUserModel(userModel),
        simulation.WithPersona("A customer who recently purchased a laptop."),
        simulation.WithGoal("Find out the return policy for electronics."),
    )
    if err != nil {
        log.Fatal(err)
    }

    agentFn := simulation.AgentFunc(func(ctx context.Context, userMsg string) (string, error) {
        // Replace with your actual agent.Invoke call.
        return "Our electronics return policy is 30 days with receipt.", nil
    })

    runner, err := simulation.NewSimRunner(
        simulation.WithSimUser(user),
        simulation.WithAgent(agentFn),
        simulation.WithMaxTurns(5),
    )
    if err != nil {
        log.Fatal(err)
    }

    report, err := runner.Run(context.Background(), 3)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("success rate: %.0f%%  avg turns: %.1f\n",
        report.SuccessRate*100, report.AverageTurns)
}
```

## Adversarial / red team — `eval/redteam`

The `eval/redteam` package tests an agent's defences against seven attack categories: prompt injection, jailbreak, obfuscation, tool misuse, data exfiltration, role-play manipulation, and multi-turn escalation. Attacks come from static registered `AttackPattern` implementations or from an LLM-driven `AttackGenerator`. Results aggregate into a `RedTeamReport` with per-category defense scores and an overall score in `[0, 1]`.

### Core types

**`AttackCategory`** (`eval/redteam/types.go:8`) — typed constant for each attack class. `AllCategories()` returns all seven.

**`Severity`** (`eval/redteam/types.go:46`) — `low`, `medium`, `high`, `critical`. Assigned by `DefenseScorer`.

**`AttackGenerator`** (`eval/redteam/generator.go:48`) — uses an `llm.ChatModel` to generate adversarial prompts beyond static patterns. Configure with `WithModel`, `WithMaxAttacks`, and `WithCategories`.

**`RedTeamRunner`** (`eval/redteam/runner.go:80`) — orchestrates the exercise. Takes an `agent.Agent` via `WithTarget`, combines static patterns and a generator, and runs attacks with bounded concurrency.

**`RedTeamReport`** (`eval/redteam/types.go:84`) — aggregate results: per-category defense scores, `OverallScore`, `TotalAttacks`, `SuccessfulAttacks`.

**`Hooks`** (`eval/redteam/hooks.go`) — `BeforeAttack`, `AfterAttack`, `OnVulnerabilityFound`. Use `OnVulnerabilityFound` to alert in CI when a new attack succeeds.

## The `EvalRunner` contract

`EvalRunner` (`eval/runner.go:82`) is the central orchestrator for per-call metric evaluation:

```go
type EvalRunner struct {
    metrics []Metric
    dataset []EvalSample
    cfg     Config
    hooks   Hooks
}
```

**Construction**: `eval.NewRunner(opts ...RunnerOption)` with functional options:

| Option | Default | Effect |
|---|---|---|
| `WithMetrics(metrics ...Metric)` | none | Metrics to run |
| `WithDataset(samples []EvalSample)` | none | Dataset to evaluate |
| `WithParallel(n int)` | `1` | Concurrent sample evaluations |
| `WithTimeout(d time.Duration)` | none | Wall-clock limit for the whole run |
| `WithStopOnError(bool)` | `false` | Stop on the first metric error |
| `WithHooks(hooks Hooks)` | none | Lifecycle callbacks |

**`Run(ctx) (*EvalReport, error)`** (`eval/runner.go:105`) evaluates all samples with bounded concurrency using a semaphore channel. It respects context cancellation at every sample boundary.

**`EvalReport`** (`eval/eval.go:49`):

```go
type EvalReport struct {
    Samples  []SampleResult         // per-sample scores
    Metrics  map[string]float64     // per-metric averages
    Duration time.Duration
    Errors   []error
}
```

`Metrics` contains the arithmetic mean score for each metric across all samples. Use these averages as quality gates in CI.

**Lifecycle hooks** (`eval/runner.go:10`): `BeforeRun`, `AfterRun`, `BeforeSample`, `AfterSample`. All fields optional. `AfterRun` receives the completed report — a convenient place to publish metrics to Prometheus or a CI dashboard.

## Dataset management — `eval/dataset.go`

**`Dataset`** wraps a named slice of `EvalSample` with JSON serialisation. Load from and save to disk:

```go
ds, err := eval.LoadDataset("testdata/qa_suite.json")
if err != nil {
    log.Fatal(err)
}
// ds.Name, ds.Samples
```

**`EvalSample`** (`eval/eval.go:24`):

```go
type EvalSample struct {
    Input          string
    Output         string
    ExpectedOutput string
    RetrievedDocs  []schema.Document
    Metadata       map[string]any   // latency_ms, input_tokens, output_tokens, model, ...
}
```

`RetrievedDocs` is used by faithfulness and hallucination metrics. `Metadata` carries non-string measurement data consumed by latency and cost metrics.

**`Augmenter`** (`eval/dataset.go:40`) is an optional interface for generating additional samples from an existing one — for example, an LLM-based paraphraser that produces adversarial variants for robustness testing.

## Providers — `eval/providers`

The three provider packages each implement `eval.Metric` by forwarding to an external evaluation service over HTTP.

### Ragas — `eval/providers/ragas`

Sends a single `EvalSample` to a Ragas API endpoint. Metric names: `faithfulness`, `answer_relevancy`, `context_precision`, `context_recall`. Returned metric name is prefixed `ragas_<metric>`. Default base URL `http://localhost:8080` for a self-hosted instance.

```go
import "github.com/lookatitude/beluga-ai/eval/providers/ragas"

m, err := ragas.New(
    ragas.WithBaseURL("http://ragas-svc:8080"),
    ragas.WithMetricName("answer_relevancy"),
)
```

### Braintrust — `eval/providers/braintrust`

Sends to the Braintrust Cloud Evaluation API at `https://api.braintrust.dev/v1/score`. Requires `WithAPIKey`. Groups results under a `WithProjectName`. Metric name is prefixed `braintrust_<metric>`.

```go
import "github.com/lookatitude/beluga-ai/eval/providers/braintrust"

m, err := braintrust.New(
    braintrust.WithAPIKey(os.Getenv("BRAINTRUST_API_KEY")),
    braintrust.WithProjectName("my-agent"),
    braintrust.WithMetricName("factuality"),
)
```

### DeepEval — `eval/providers/deepeval`

Sends to the DeepEval API. Default base URL `http://localhost:8080` for a self-hosted instance; cloud endpoint configurable via `WithBaseURL`. Returns an error if `evaluateResponse.Success` is false (the service itself considers the evaluation failed). Metric name is prefixed `deepeval_<metric>`.

```go
import "github.com/lookatitude/beluga-ai/eval/providers/deepeval"

m, err := deepeval.New(
    deepeval.WithBaseURL("https://app.confident-ai.com"),
    deepeval.WithAPIKey(os.Getenv("DEEPEVAL_API_KEY")),
    deepeval.WithMetricName("g-eval"),
)
```

All three provider metrics implement `var _ eval.Metric = (*Metric)(nil)` for compile-time interface verification.

## Conversation clustering — `eval/clustering`

The `eval/clustering` package groups conversation logs by similarity to surface recurring patterns. It provides three interfaces — `SimilarityScorer`, `PatternDetector`, `ConversationClusterer` — and a built-in agglomerative hierarchical clusterer (`AgglomerativeClusterer`) registered as `"agglomerative"`. Use `clustering.New("agglomerative", cfg)` or call `clustering.NewAgglomerative(opts...)` directly. `TurnPatternDetector` identifies structural patterns (turn count, role sequences, length outliers) without an LLM.

## Cost analysis — `eval/cost`

`eval/cost.CostMetric` extends the raw cost calculation in `eval/metrics.Cost` with quality-per-dollar normalization. When constructed with `WithQualityMetric(m)`, `Score` returns `quality/cost` normalized to `[0, 1]` using a configurable QPD reference; without a quality metric it returns a cost-only normalized score `(1 - cost/costReference)`. `ComputeRawCost` returns the raw dollar amount for budget tracking.

## Wiring eval into CI

A typical CI quality gate pattern:

```go
import (
    "context"
    "fmt"
    "time"

    _ "github.com/lookatitude/beluga-ai/llm/providers/anthropic"
    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/eval"
    "github.com/lookatitude/beluga-ai/eval/metrics"
    "github.com/lookatitude/beluga-ai/llm"
)

func runQualityGate(datasetPath string) error {
    ctx := context.Background()

    ds, err := eval.LoadDataset(datasetPath)
    if err != nil {
        return fmt.Errorf("load dataset: %w", err)
    }

    judge, err := llm.New("anthropic", config.ProviderConfig{
        Provider: "anthropic",
        Model:    "claude-3-haiku-20240307",
    })
    if err != nil {
        return fmt.Errorf("create judge model: %w", err)
    }

    runner := eval.NewRunner(
        eval.WithDataset(ds.Samples),
        eval.WithMetrics(
            metrics.NewFaithfulness(judge),
            metrics.NewRelevance(judge),
            metrics.NewHallucination(judge),
            metrics.NewToxicity(),
            metrics.NewLatency(metrics.WithMaxLatencyMs(3000)),
        ),
        eval.WithParallel(4),
        eval.WithTimeout(10*time.Minute),
        eval.WithHooks(eval.Hooks{
            AfterRun: func(_ context.Context, r *eval.EvalReport) {
                for name, score := range r.Metrics {
                    fmt.Printf("metric=%s score=%.3f\n", name, score)
                }
            },
        }),
    )

    report, err := runner.Run(ctx)
    if err != nil {
        return fmt.Errorf("eval run: %w", err)
    }

    thresholds := map[string]float64{
        "faithfulness":  0.80,
        "relevance":     0.80,
        "hallucination": 0.85,
        "toxicity":      0.95,
        "latency":       0.70,
    }

    for name, threshold := range thresholds {
        if score, ok := report.Metrics[name]; ok && score < threshold {
            return fmt.Errorf("quality gate failed: %s=%.3f < %.2f", name, score, threshold)
        }
    }

    return nil
}
```

Run this in a GitHub Actions step or as a Go test with `//go:build integration`. The runner returns an error only for infrastructure failures (context timeout, LLM provider down); metric scores below threshold are surfaced in `EvalReport.Metrics` and require application-level gate logic as shown above.

## Common mistakes

**Treating `eval/metrics.Cost` as a normalized metric.** `metrics.Cost.Score` returns a raw dollar amount, not a `[0, 1]` score. Pass it to `eval/cost.CostMetric` with `WithQualityMetric` for a normalized view, or use `ComputeRawCost` for budget tracking.

**Omitting `Metadata` keys required by latency and cost metrics.** `Latency` requires `Metadata["latency_ms"]` and `Cost` requires `Metadata["model"]`, `Metadata["input_tokens"]`, and `Metadata["output_tokens"]`. Both return `core.ErrInvalidInput` when keys are absent. Populate these fields when constructing `EvalSample` in your agent instrumentation.

**Running LLM-judge metrics synchronously on large datasets.** Each faithfulness, relevance, or hallucination score makes one LLM call. On a 500-sample dataset this is 500–1500 LLM calls. Use `WithParallel` to bound concurrency and `WithTimeout` to prevent CI hangs.

**Constructing a `JudgeMetric` without calling `Rubric.Validate()`.** `NewJudgeMetric` calls `Validate()` for you and returns an error on malformed rubrics. Do not bypass this by constructing `JudgeMetric` directly — the zero-value struct has no nil-safety.

**Using `ConsistencyChecker` with a single model and one repeat.** Agreement is only meaningful when there is variance to measure. Use at least two models or three repeats to get a non-trivial `Agreement` score.

**Setting `WithStopOnError(true)` in CI without reading `EvalReport.Errors`.** When `StopOnError` is false (default), metric errors are recorded per-sample in `SampleResult.Error` and aggregated in `EvalReport.Errors`. Inspect this slice before treating a partial report as authoritative.

**Pointing the red team runner at a production agent.** `RedTeamRunner` calls `agent.Invoke` directly with adversarial inputs. Always run red-team evaluations against a staging or sandbox agent instance, never production.

## Related reading

- [03 — Extensibility Patterns](./03-extensibility-patterns.md) — the interface/registry/hooks/middleware model that `eval/trajectory` reuses.
- [05 — Agent Anatomy](./05-agent-anatomy.md) — how to capture trajectory data from a running agent.
- [14 — Observability](./14-observability.md) — OTel spans complement metric scores by providing per-call timing and token data to populate `EvalSample.Metadata`.
- [13 — Security Model](./13-security-model.md) — the guard pipeline that red-team evaluation exercises.
- [`docs/reference/providers.md`](../reference/providers.md) — provider catalog including `eval/providers` entries.
