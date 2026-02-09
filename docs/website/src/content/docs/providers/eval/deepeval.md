---
title: "DeepEval"
description: "Evaluate LLM outputs using a DeepEval server."
---

The DeepEval provider connects Beluga AI's evaluation framework to a [DeepEval](https://github.com/confident-ai/deepeval) server instance. It implements the `eval.Metric` interface by sending samples to the DeepEval evaluation API and returning normalized scores along with optional reasoning.

## Installation

```bash
go get github.com/lookatitude/beluga-ai/eval/providers/deepeval
```

## Configuration

| Option | Type | Default | Description |
|---|---|---|---|
| `WithMetricName(name)` | `string` | `"faithfulness"` | Metric to evaluate |
| `WithBaseURL(url)` | `string` | `http://localhost:8080` | DeepEval server endpoint |
| `WithAPIKey(key)` | `string` | — | Optional bearer token for authentication |
| `WithTimeout(d)` | `time.Duration` | `30s` | HTTP request timeout |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/eval"
    "github.com/lookatitude/beluga-ai/eval/providers/deepeval"
)

func main() {
    metric, err := deepeval.New(
        deepeval.WithMetricName("faithfulness"),
        deepeval.WithBaseURL("http://localhost:8080"),
    )
    if err != nil {
        log.Fatal(err)
    }

    sample := eval.EvalSample{
        Input:          "What causes rain?",
        Output:         "Rain forms when water vapor condenses in clouds.",
        ExpectedOutput: "Rain is caused by condensation of atmospheric water vapor into droplets.",
    }

    score, err := metric.Score(context.Background(), sample)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("%s: %.3f\n", metric.Name(), score)
    // Output: deepeval_faithfulness: 0.890
}
```

## With EvalRunner

Use DeepEval metrics with the evaluation runner for batch evaluation:

```go
metric, err := deepeval.New(
    deepeval.WithMetricName("faithfulness"),
    deepeval.WithBaseURL("http://localhost:8080"),
)
if err != nil {
    log.Fatal(err)
}

runner := eval.NewRunner(
    eval.WithMetrics(metric),
    eval.WithDataset(samples),
    eval.WithParallel(4),
)

report, err := runner.Run(context.Background())
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Average faithfulness: %.3f\n", report.Metrics["deepeval_faithfulness"])
```

## RAG Evaluation

Include retrieved documents as context for RAG-specific metrics:

```go
sample := eval.EvalSample{
    Input:  "What are the benefits of exercise?",
    Output: "Exercise improves cardiovascular health, mental well-being, and muscle strength.",
    RetrievedDocs: []schema.Document{
        {Content: "Regular physical activity reduces the risk of heart disease..."},
        {Content: "Exercise releases endorphins that improve mood..."},
    },
}

score, err := metric.Score(ctx, sample)
```

The provider extracts content from `RetrievedDocs` and sends it as context to the DeepEval API alongside the input, output, and expected output.

## Authenticated Server

For DeepEval servers that require authentication, provide an API key:

```go
metric, err := deepeval.New(
    deepeval.WithMetricName("hallucination"),
    deepeval.WithBaseURL("https://deepeval.example.com"),
    deepeval.WithAPIKey(os.Getenv("DEEPEVAL_API_KEY")),
)
```

The API key is sent as a bearer token in the `Authorization` header.

## Metric Naming

DeepEval metrics are prefixed with `deepeval_` to distinguish them from metrics from other providers. For example, a metric configured with `WithMetricName("faithfulness")` reports its name as `deepeval_faithfulness`.

## Error Handling

```go
score, err := metric.Score(ctx, sample)
if err != nil {
    // Errors include HTTP failures, server-side evaluation failures, and timeouts
    log.Printf("DeepEval scoring failed: %v", err)
}
```

The provider checks the `success` field in the API response. If the server reports `success: false`, the provider returns an error even if an HTTP 200 response was received.

Scores are clamped to the `[0.0, 1.0]` range. If the API returns a score outside this range, it is automatically normalized.
