---
title: "Evaluation Providers"
description: "Overview of the evaluation framework and supported eval providers in Beluga AI v2."
---

Beluga AI v2 includes a built-in evaluation framework for measuring the quality of LLM outputs, RAG pipelines, and agent behavior. The framework defines a `Metric` interface that external evaluation platforms implement, and an `EvalRunner` that orchestrates parallel evaluation across datasets.

## Core Interface

All evaluation providers implement the `Metric` interface:

```go
type Metric interface {
    Name() string
    Score(ctx context.Context, sample EvalSample) (float64, error)
}
```

Each `Score` call returns a value in the range `[0.0, 1.0]`, where higher values indicate better quality.

## EvalSample

The `EvalSample` struct carries all the data needed for evaluation:

```go
type EvalSample struct {
    Input          string            // Original question or prompt
    Output         string            // AI-generated response
    ExpectedOutput string            // Ground-truth reference answer
    RetrievedDocs  []schema.Document // Context documents used for generation
    Metadata       map[string]any    // Metric-specific data (latency, tokens, model)
}
```

## EvalRunner

The `EvalRunner` orchestrates evaluation across a dataset of samples with configurable parallelism, timeouts, and lifecycle hooks:

```go
import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/eval"
    _ "github.com/lookatitude/beluga-ai/eval/providers/ragas"
)

func main() {
    metric, err := ragas.New(
        ragas.WithMetricName("faithfulness"),
        ragas.WithBaseURL("http://localhost:8080"),
    )
    if err != nil {
        log.Fatal(err)
    }

    runner := eval.NewRunner(
        eval.WithMetrics(metric),
        eval.WithDataset(samples),
        eval.WithParallel(4),
        eval.WithTimeout(5 * time.Minute),
    )

    report, err := runner.Run(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    for name, score := range report.Metrics {
        fmt.Printf("%s: %.3f\n", name, score)
    }
}
```

## Runner Options

| Option | Type | Default | Description |
|---|---|---|---|
| `WithMetrics(metrics ...Metric)` | `[]Metric` | (required) | Metrics to evaluate |
| `WithDataset(samples []EvalSample)` | `[]EvalSample` | (required) | Evaluation dataset |
| `WithParallel(n int)` | `int` | `1` | Concurrent sample evaluation |
| `WithTimeout(d time.Duration)` | `time.Duration` | `0` (none) | Maximum duration for entire run |
| `WithStopOnError(stop bool)` | `bool` | `false` | Stop on first metric error |
| `WithHooks(hooks Hooks)` | `Hooks` | — | Lifecycle callbacks |

## Hooks

The evaluation runner supports lifecycle hooks for logging, progress tracking, and integration with CI systems:

```go
hooks := eval.Hooks{
    BeforeRun: func(ctx context.Context, samples []eval.EvalSample) error {
        log.Printf("Starting evaluation with %d samples", len(samples))
        return nil
    },
    AfterRun: func(ctx context.Context, report *eval.EvalReport) {
        log.Printf("Evaluation complete: %d samples in %s", len(report.Samples), report.Duration)
    },
    BeforeSample: func(ctx context.Context, sample eval.EvalSample) error {
        log.Printf("Evaluating: %s", sample.Input[:50])
        return nil
    },
    AfterSample: func(ctx context.Context, result eval.SampleResult) {
        for name, score := range result.Scores {
            log.Printf("  %s: %.3f", name, score)
        }
    },
}

runner := eval.NewRunner(
    eval.WithMetrics(metric),
    eval.WithDataset(samples),
    eval.WithHooks(hooks),
)
```

## EvalReport

The `EvalReport` aggregates results across all samples:

```go
type EvalReport struct {
    Samples  []SampleResult         // Per-sample results
    Metrics  map[string]float64     // Average score per metric
    Duration time.Duration          // Total evaluation time
    Errors   []error                // Collected errors
}

type SampleResult struct {
    Sample  EvalSample
    Scores  map[string]float64     // Metric name to score
    Error   error
}
```

## Datasets

Load and save evaluation datasets as JSON:

```go
dataset, err := eval.LoadDataset("testdata/qa_pairs.json")
if err != nil {
    log.Fatal(err)
}

runner := eval.NewRunner(
    eval.WithMetrics(metric),
    eval.WithDataset(dataset.Samples),
)

// Save results for later analysis
err = dataset.Save("testdata/results.json")
```

## Available Providers

| Provider | Prefix | Default Metric | Description |
|---|---|---|---|
| [Braintrust](/providers/eval/braintrust) | `braintrust_` | `factuality` | Cloud-hosted evaluation via Braintrust API |
| [DeepEval](/providers/eval/deepeval) | `deepeval_` | `faithfulness` | Evaluation via DeepEval server |
| [RAGAS](/providers/eval/ragas) | `ragas_` | `faithfulness` | RAG-focused evaluation via RAGAS server |

## Multiple Metrics

Combine metrics from different providers in a single evaluation run:

```go
btMetric, err := braintrust.New(
    braintrust.WithAPIKey(os.Getenv("BRAINTRUST_API_KEY")),
    braintrust.WithMetricName("factuality"),
)
if err != nil {
    log.Fatal(err)
}

ragasMetric, err := ragas.New(
    ragas.WithMetricName("answer_relevancy"),
    ragas.WithBaseURL("http://localhost:8080"),
)
if err != nil {
    log.Fatal(err)
}

runner := eval.NewRunner(
    eval.WithMetrics(btMetric, ragasMetric),
    eval.WithDataset(samples),
    eval.WithParallel(4),
)

report, err := runner.Run(ctx)
// report.Metrics contains both "braintrust_factuality" and "ragas_answer_relevancy"
```
