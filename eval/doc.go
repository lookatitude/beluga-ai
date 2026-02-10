// Package eval provides an evaluation framework for measuring the quality of
// AI-generated outputs. It defines a Metric interface for scoring individual
// samples, an EvalRunner for parallel metric execution across datasets, and
// types for representing evaluation results.
//
// # Metric Interface
//
// The Metric interface is the core abstraction:
//
//   - Name returns the unique name of the metric (e.g., "faithfulness").
//   - Score evaluates a single EvalSample and returns a float64 in [0, 1].
//     Higher scores indicate better quality for quality metrics.
//
// Built-in metrics are available in the eval/metrics sub-package. External
// evaluation providers (Braintrust, DeepEval, RAGAS) are available under
// eval/providers/.
//
// # EvalSample
//
// EvalSample represents a single evaluation sample containing the input
// question, the generated output, the expected output, retrieved documents,
// and arbitrary metadata (e.g., latency_ms, input_tokens, model).
//
// # EvalRunner
//
// EvalRunner runs a set of metrics against a dataset of samples with
// configurable concurrency. Configure it with functional options:
//
//   - WithMetrics sets the metrics to evaluate.
//   - WithDataset sets the evaluation samples.
//   - WithParallel sets the concurrency level.
//   - WithTimeout sets the maximum evaluation duration.
//   - WithStopOnError stops on the first metric error.
//   - WithHooks sets lifecycle callbacks (BeforeRun, AfterRun, BeforeSample,
//     AfterSample).
//
// # Dataset
//
// Dataset is a named collection of EvalSample values that can be loaded from
// and saved to JSON files via LoadDataset and Save.
//
// # Augmenter
//
// The Augmenter interface generates additional evaluation samples from
// existing ones for more robust evaluation.
//
// # Usage
//
//	runner := eval.NewRunner(
//	    eval.WithMetrics(metrics.NewToxicity(), metrics.NewLatency()),
//	    eval.WithDataset(samples),
//	    eval.WithParallel(4),
//	)
//	report, err := runner.Run(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for name, avg := range report.Metrics {
//	    fmt.Printf("%s: %.2f\n", name, avg)
//	}
package eval
