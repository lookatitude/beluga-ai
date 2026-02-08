// Package eval provides an evaluation framework for measuring the quality of
// AI-generated outputs. It defines a Metric interface for scoring individual
// samples, an EvalRunner for parallel metric execution across datasets, and
// built-in metrics covering faithfulness, relevance, hallucination detection,
// toxicity, latency, and cost.
//
// Basic usage:
//
//	runner := eval.NewRunner(
//	    eval.WithMetrics(metrics.NewToxicity(), metrics.NewLatency()),
//	    eval.WithDataset(samples),
//	    eval.WithParallel(4),
//	)
//	report, err := runner.Run(ctx)
package eval

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/schema"
)

// Metric is the interface that all evaluation metrics implement.
// A Metric scores a single evaluation sample and returns a float64 in [0, 1].
type Metric interface {
	// Name returns the unique name of this metric (e.g., "faithfulness").
	Name() string

	// Score evaluates a single sample and returns a score in [0.0, 1.0].
	// Higher scores indicate better quality for quality metrics.
	Score(ctx context.Context, sample EvalSample) (float64, error)
}

// EvalSample represents a single evaluation sample containing the input
// question, the generated output, the expected output, and any additional
// context such as retrieved documents and metadata.
type EvalSample struct {
	// Input is the original question or prompt.
	Input string
	// Output is the AI-generated response to evaluate.
	Output string
	// ExpectedOutput is the ground-truth or reference answer.
	ExpectedOutput string
	// RetrievedDocs are the documents used as context for generation.
	RetrievedDocs []schema.Document
	// Metadata holds arbitrary key-value pairs for metric-specific data,
	// such as latency_ms, input_tokens, output_tokens, model, etc.
	Metadata map[string]any
}

// SampleResult holds the evaluation scores for a single sample across
// all configured metrics.
type SampleResult struct {
	// Sample is the original evaluation sample.
	Sample EvalSample
	// Scores maps metric names to their scores for this sample.
	Scores map[string]float64
	// Error is set if any metric failed for this sample.
	Error error
}

// EvalReport is the aggregate result of running an evaluation suite.
type EvalReport struct {
	// Samples contains the per-sample results.
	Samples []SampleResult
	// Metrics contains the average score for each metric across all samples.
	Metrics map[string]float64
	// Duration is the total wall-clock time of the evaluation run.
	Duration time.Duration
	// Errors collects all errors encountered during evaluation.
	Errors []error
}
