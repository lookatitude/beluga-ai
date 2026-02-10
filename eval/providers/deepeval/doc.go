// Package deepeval provides a DeepEval evaluation metric for the Beluga AI
// eval framework. It implements the eval.Metric interface and sends evaluation
// requests to a DeepEval API endpoint.
//
// DeepEval provides LLM evaluation metrics including faithfulness, answer
// relevancy, contextual precision, hallucination, and bias.
//
// # Configuration
//
// The metric is configured using functional options:
//
//   - WithBaseURL sets the DeepEval API base URL. Defaults to
//     "http://localhost:8080".
//   - WithAPIKey sets the API key for authentication (optional).
//   - WithMetricName sets the metric to evaluate. Defaults to "faithfulness".
//   - WithTimeout sets the HTTP client timeout. Defaults to 30 seconds.
//
// The metric Name is prefixed with "deepeval_" (e.g., "deepeval_faithfulness").
//
// # Usage
//
//	metric, err := deepeval.New(
//	    deepeval.WithBaseURL("http://localhost:8080"),
//	    deepeval.WithMetricName("faithfulness"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	score, err := metric.Score(ctx, sample)
package deepeval
