---
title: "RAGAS Evaluation Provider"
description: "Evaluate RAG pipeline quality with RAGAS in Beluga AI. Faithfulness, answer relevancy, and context precision metrics for retrieval evaluation in Go."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "RAGAS, RAG evaluation, faithfulness, answer relevancy, context precision, eval metrics, Go, Beluga AI"
---

The RAGAS provider connects Beluga AI's evaluation framework to a [RAGAS](https://docs.ragas.io/) server instance. It implements the `eval.Metric` interface with RAG-specific evaluation metrics such as faithfulness, answer relevancy, context precision, and context recall.

Choose RAGAS when you are evaluating RAG pipelines and need metrics that specifically measure retrieval quality and answer groundedness. RAGAS provides four complementary metrics (faithfulness, answer relevancy, context precision, context recall) designed for end-to-end RAG assessment. For general LLM evaluation beyond RAG, consider [DeepEval](/providers/eval/deepeval) or [Braintrust](/providers/eval/braintrust).

## Installation

```bash
go get github.com/lookatitude/beluga-ai/eval/providers/ragas
```

## Configuration

| Option | Type | Default | Description |
|---|---|---|---|
| `WithMetricName(name)` | `string` | `"faithfulness"` | Metric to evaluate |
| `WithBaseURL(url)` | `string` | `http://localhost:8080` | RAGAS server endpoint |
| `WithAPIKey(key)` | `string` | — | Optional bearer token for authentication |
| `WithTimeout(d)` | `time.Duration` | `30s` | HTTP request timeout |

## Supported Metrics

| Metric Name | Description |
|---|---|
| `faithfulness` | Measures whether the answer is grounded in the provided context |
| `answer_relevancy` | Measures how relevant the answer is to the question |
| `context_precision` | Measures whether the retrieved context contains relevant information |
| `context_recall` | Measures whether all relevant information is present in the context |

## Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lookatitude/beluga-ai/eval"
    "github.com/lookatitude/beluga-ai/eval/providers/ragas"
    "github.com/lookatitude/beluga-ai/schema"
)

func main() {
    metric, err := ragas.New(
        ragas.WithMetricName("faithfulness"),
        ragas.WithBaseURL("http://localhost:8080"),
    )
    if err != nil {
        log.Fatal(err)
    }

    sample := eval.EvalSample{
        Input:          "What is photosynthesis?",
        Output:         "Photosynthesis converts sunlight into chemical energy in plants.",
        ExpectedOutput: "Photosynthesis is the process by which plants convert light energy into glucose.",
        RetrievedDocs: []schema.Document{
            {Content: "Photosynthesis is a process used by plants to convert light energy into chemical energy."},
            {Content: "The process occurs primarily in the leaves of plants using chlorophyll."},
        },
    }

    score, err := metric.Score(context.Background(), sample)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("%s: %.3f\n", metric.Name(), score)
    // Output: ragas_faithfulness: 0.920
}
```

## With EvalRunner

Use RAGAS metrics with the evaluation runner for batch evaluation:

```go
faithfulness, err := ragas.New(
    ragas.WithMetricName("faithfulness"),
    ragas.WithBaseURL("http://localhost:8080"),
)
if err != nil {
    log.Fatal(err)
}

relevancy, err := ragas.New(
    ragas.WithMetricName("answer_relevancy"),
    ragas.WithBaseURL("http://localhost:8080"),
)
if err != nil {
    log.Fatal(err)
}

runner := eval.NewRunner(
    eval.WithMetrics(faithfulness, relevancy),
    eval.WithDataset(samples),
    eval.WithParallel(4),
    eval.WithTimeout(5 * time.Minute),
)

report, err := runner.Run(context.Background())
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Faithfulness:     %.3f\n", report.Metrics["ragas_faithfulness"])
fmt.Printf("Answer relevancy: %.3f\n", report.Metrics["ragas_answer_relevancy"])
```

## Multi-Metric RAG Evaluation

Combine multiple RAGAS metrics for a comprehensive RAG pipeline assessment:

```go
metricNames := []string{"faithfulness", "answer_relevancy", "context_precision", "context_recall"}
var metrics []eval.Metric

for _, name := range metricNames {
    m, err := ragas.New(
        ragas.WithMetricName(name),
        ragas.WithBaseURL("http://localhost:8080"),
    )
    if err != nil {
        log.Fatal(err)
    }
    metrics = append(metrics, m)
}

runner := eval.NewRunner(
    eval.WithMetrics(metrics...),
    eval.WithDataset(samples),
    eval.WithParallel(4),
)

report, err := runner.Run(ctx)
if err != nil {
    log.Fatal(err)
}

for name, score := range report.Metrics {
    fmt.Printf("%s: %.3f\n", name, score)
}
```

## Field Mapping

RAGAS uses RAG-specific terminology. The provider automatically maps `EvalSample` fields to RAGAS conventions:

| EvalSample Field | RAGAS Field | Description |
|---|---|---|
| `Input` | `question` | The user's query |
| `Output` | `answer` | The generated response |
| `ExpectedOutput` | `ground_truth` | The reference answer |
| `RetrievedDocs` | `contexts` | Array of context document contents |

## Authenticated Server

For RAGAS servers that require authentication, provide an API key:

```go
metric, err := ragas.New(
    ragas.WithMetricName("faithfulness"),
    ragas.WithBaseURL("https://ragas.example.com"),
    ragas.WithAPIKey(os.Getenv("RAGAS_API_KEY")),
)
```

The API key is sent as a bearer token in the `Authorization` header.

## Metric Naming

RAGAS metrics are prefixed with `ragas_` to distinguish them from metrics from other providers. For example, a metric configured with `WithMetricName("faithfulness")` reports its name as `ragas_faithfulness`.

## Error Handling

```go
score, err := metric.Score(ctx, sample)
if err != nil {
    // Errors include HTTP failures, invalid metric names, and server-side errors
    log.Printf("RAGAS scoring failed: %v", err)
}
```

Scores are clamped to the `[0.0, 1.0]` range. If the API returns a score outside this range, it is automatically normalized.
