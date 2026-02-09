---
title: "Eval Package"
description: "Evaluation framework: metrics, runners, and provider integrations"
---

## eval

```go
import "github.com/lookatitude/beluga-ai/eval"
```

Package eval provides an evaluation framework for measuring the quality of
AI-generated outputs. It defines a Metric interface for scoring individual
samples, an EvalRunner for parallel metric execution across datasets, and
types for representing evaluation results.

## Metric Interface

The Metric interface is the core abstraction:

- Name returns the unique name of the metric (e.g., "faithfulness").
- Score evaluates a single EvalSample and returns a float64 in [0, 1].
  Higher scores indicate better quality for quality metrics.

Built-in metrics are available in the eval/metrics sub-package. External
evaluation providers (Braintrust, DeepEval, RAGAS) are available under
eval/providers/.

## EvalSample

EvalSample represents a single evaluation sample containing the input
question, the generated output, the expected output, retrieved documents,
and arbitrary metadata (e.g., latency_ms, input_tokens, model).

## EvalRunner

EvalRunner runs a set of metrics against a dataset of samples with
configurable concurrency. Configure it with functional options:

- WithMetrics sets the metrics to evaluate.
- WithDataset sets the evaluation samples.
- WithParallel sets the concurrency level.
- WithTimeout sets the maximum evaluation duration.
- WithStopOnError stops on the first metric error.
- WithHooks sets lifecycle callbacks (BeforeRun, AfterRun, BeforeSample,
  AfterSample).

## Dataset

Dataset is a named collection of EvalSample values that can be loaded from
and saved to JSON files via LoadDataset and Save.

## Augmenter

The Augmenter interface generates additional evaluation samples from
existing ones for more robust evaluation.

## Usage

```go
runner := eval.NewRunner(
    eval.WithMetrics(metrics.NewToxicity(), metrics.NewLatency()),
    eval.WithDataset(samples),
    eval.WithParallel(4),
)
report, err := runner.Run(ctx)
if err != nil {
    log.Fatal(err)
}
for name, avg := range report.Metrics {
    fmt.Printf("%s: %.2f\n", name, avg)
}
```

---

## metrics

```go
import "github.com/lookatitude/beluga-ai/eval/metrics"
```

Package metrics provides built-in evaluation metrics for the Beluga AI
eval framework. Each metric implements the eval.Metric interface, returning
a score in [0.0, 1.0] for a given EvalSample.

## LLM-as-Judge Metrics

These metrics use an LLM to evaluate AI-generated output quality:

- Faithfulness evaluates whether an answer is grounded in the provided
  context documents. Requires an llm.ChatModel as judge.
- Relevance evaluates whether an answer adequately addresses the input
  question. Requires an llm.ChatModel as judge.
- Hallucination detects fabricated facts by comparing answers against
  context documents. Requires an llm.ChatModel as judge.

## Keyword-Based Metrics

- Toxicity performs keyword-based toxicity checking. Returns 1.0 (not
  toxic) when no toxic keywords are found, decreasing toward 0.0 as more
  keywords are detected. Configurable keyword list and threshold.

## Metadata-Based Metrics

- Latency reads Metadata["latency_ms"] and returns a normalized score
  where 1.0 is instantaneous and 0.0 is at or above a configurable
  maximum threshold (default 10 seconds).
- Cost reads Metadata["input_tokens"], Metadata["output_tokens"], and
  Metadata["model"] to calculate the dollar cost based on configurable
  per-model pricing. Returns the raw dollar amount rather than a
  normalized score.

## Usage

```go
// LLM-as-judge metric
faith := metrics.NewFaithfulness(judgeModel)
score, err := faith.Score(ctx, sample)

// Keyword-based metric
tox := metrics.NewToxicity()
score, err = tox.Score(ctx, sample)

// Metadata-based metric
lat := metrics.NewLatency(metrics.WithMaxLatencyMs(5000))
score, err = lat.Score(ctx, sample)
```

---

## braintrust

```go
import "github.com/lookatitude/beluga-ai/eval/providers/braintrust"
```

Package braintrust provides a Braintrust evaluation metric for the Beluga AI
eval framework. It implements the eval.Metric interface and sends evaluation
requests to the Braintrust API.

Braintrust provides evaluation scoring for LLM outputs including
factuality, relevance, and custom scoring functions.

## Configuration

The metric is configured using functional options:

- WithAPIKey sets the Braintrust API key (required).
- WithProjectName sets the Braintrust project name. Defaults to "default".
- WithMetricName sets the metric to evaluate. Defaults to "factuality".
- WithBaseURL sets the API base URL. Defaults to "https://api.braintrust.dev".
- WithTimeout sets the HTTP client timeout. Defaults to 30 seconds.

The metric Name is prefixed with "braintrust_" (e.g., "braintrust_factuality").

## Usage

```go
metric, err := braintrust.New(
    braintrust.WithAPIKey("bt-..."),
    braintrust.WithProjectName("my-project"),
    braintrust.WithMetricName("factuality"),
)
if err != nil {
    log.Fatal(err)
}
score, err := metric.Score(ctx, sample)
```

---

## deepeval

```go
import "github.com/lookatitude/beluga-ai/eval/providers/deepeval"
```

Package deepeval provides a DeepEval evaluation metric for the Beluga AI
eval framework. It implements the eval.Metric interface and sends evaluation
requests to a DeepEval API endpoint.

DeepEval provides LLM evaluation metrics including faithfulness, answer
relevancy, contextual precision, hallucination, and bias.

## Configuration

The metric is configured using functional options:

- WithBaseURL sets the DeepEval API base URL. Defaults to
  "http://localhost:8080".
- WithAPIKey sets the API key for authentication (optional).
- WithMetricName sets the metric to evaluate. Defaults to "faithfulness".
- WithTimeout sets the HTTP client timeout. Defaults to 30 seconds.

The metric Name is prefixed with "deepeval_" (e.g., "deepeval_faithfulness").

## Usage

```go
metric, err := deepeval.New(
    deepeval.WithBaseURL("http://localhost:8080"),
    deepeval.WithMetricName("faithfulness"),
)
if err != nil {
    log.Fatal(err)
}
score, err := metric.Score(ctx, sample)
```

---

## ragas

```go
import "github.com/lookatitude/beluga-ai/eval/providers/ragas"
```

Package ragas provides RAGAS (Retrieval Augmented Generation Assessment)
evaluation metrics for the Beluga AI eval framework. It implements the
eval.Metric interface and sends evaluation requests to a RAGAS API endpoint.

RAGAS provides metrics for evaluating RAG pipelines including faithfulness,
answer relevancy, context precision, and context recall.

## Configuration

The metric is configured using functional options:

- WithBaseURL sets the RAGAS API base URL. Defaults to
  "http://localhost:8080".
- WithAPIKey sets the API key for authentication (optional).
- WithMetricName sets the metric to evaluate (e.g., "faithfulness",
  "answer_relevancy", "context_precision", "context_recall"). Defaults
  to "faithfulness".
- WithTimeout sets the HTTP client timeout. Defaults to 30 seconds.

The metric Name is prefixed with "ragas_" (e.g., "ragas_faithfulness").

## Usage

```go
metric, err := ragas.New(
    ragas.WithBaseURL("http://localhost:8080"),
    ragas.WithMetricName("faithfulness"),
)
if err != nil {
    log.Fatal(err)
}
score, err := metric.Score(ctx, sample)
```
