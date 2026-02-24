---
title: "Braintrust Evaluation Provider"
description: "Evaluate LLM outputs with Braintrust scoring API in Beluga AI. Automated scoring, experiment tracking, and dataset management in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Braintrust, LLM evaluation, scoring API, experiment tracking, eval metrics, AI testing, Go, Beluga AI"
---

The Braintrust provider connects Beluga AI's evaluation framework to the [Braintrust](https://www.braintrust.dev/) cloud platform. It implements the `eval.Metric` interface by sending samples to the Braintrust scoring API and returning normalized scores.

Choose Braintrust when you want a managed evaluation platform with a dashboard for tracking scores over time. Braintrust supports factuality, relevancy, and other LLM-specific metrics with built-in project organization. For self-hosted evaluation with Python-native metrics, consider [DeepEval](/docs/providers/eval/deepeval). For RAG-specific metrics, consider [RAGAS](/docs/providers/eval/ragas).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/eval/providers/braintrust
```

## Configuration

| Option | Type | Default | Description |
|---|---|---|---|
| `WithAPIKey(key)` | `string` | (required) | Braintrust API key |
| `WithMetricName(name)` | `string` | `"factuality"` | Metric to evaluate |
| `WithProjectName(name)` | `string` | `"default"` | Braintrust project name |
| `WithBaseURL(url)` | `string` | `https://api.braintrust.dev` | API endpoint |
| `WithTimeout(d)` | `time.Duration` | `30s` | HTTP request timeout |

**Environment variables:**

| Variable | Maps to |
|---|---|
| `BRAINTRUST_API_KEY` | `WithAPIKey` |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lookatitude/beluga-ai/eval"
    "github.com/lookatitude/beluga-ai/eval/providers/braintrust"
)

func main() {
    metric, err := braintrust.New(
        braintrust.WithAPIKey(os.Getenv("BRAINTRUST_API_KEY")),
        braintrust.WithMetricName("factuality"),
        braintrust.WithProjectName("my-project"),
    )
    if err != nil {
        log.Fatal(err)
    }

    sample := eval.EvalSample{
        Input:          "What is the capital of France?",
        Output:         "The capital of France is Paris.",
        ExpectedOutput: "Paris",
    }

    score, err := metric.Score(context.Background(), sample)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("%s: %.3f\n", metric.Name(), score)
    // Output: braintrust_factuality: 0.950
}
```

## With EvalRunner

Use Braintrust metrics with the evaluation runner for batch evaluation:

```go
metric, err := braintrust.New(
    braintrust.WithAPIKey(os.Getenv("BRAINTRUST_API_KEY")),
    braintrust.WithMetricName("factuality"),
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

fmt.Printf("Average factuality: %.3f\n", report.Metrics["braintrust_factuality"])
```

## RAG Evaluation

When evaluating RAG pipelines, include retrieved documents as context:

```go
sample := eval.EvalSample{
    Input:          "How does photosynthesis work?",
    Output:         "Photosynthesis converts CO2 and water into glucose using sunlight.",
    ExpectedOutput: "Plants use light energy to convert carbon dioxide and water into glucose and oxygen.",
    RetrievedDocs: []schema.Document{
        {Content: "Photosynthesis is the process by which plants convert light energy..."},
        {Content: "The process occurs in chloroplasts and involves two stages..."},
    },
}

score, err := metric.Score(ctx, sample)
```

The provider extracts content from `RetrievedDocs` and sends it as context to the Braintrust API alongside the input, output, and expected output.

## Metric Naming

Braintrust metrics are prefixed with `braintrust_` to distinguish them from metrics from other providers. For example, a metric configured with `WithMetricName("factuality")` reports its name as `braintrust_factuality`.

## Error Handling

```go
score, err := metric.Score(ctx, sample)
if err != nil {
    // Errors include HTTP failures, invalid API responses, and authentication issues
    log.Printf("Braintrust scoring failed: %v", err)
}
```

Scores are clamped to the `[0.0, 1.0]` range. If the API returns a score outside this range, it is automatically normalized.
