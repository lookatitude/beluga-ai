// Package deepeval provides a DeepEval evaluation metric for the Beluga AI
// eval framework. It implements the eval.Metric interface and sends evaluation
// requests to a DeepEval API endpoint.
//
// DeepEval provides LLM evaluation metrics including faithfulness, answer
// relevancy, contextual precision, hallucination, and bias.
//
// # Requires a user-provided HTTP shim
//
// DeepEval is a Python library; it does not expose an /api/v1/evaluate HTTP
// endpoint (and Confident AI's hosted platform uses a different API). This
// adapter therefore does NOT talk to any vendor endpoint. It expects a small
// self-hosted service at BaseURL (for example a FastAPI app wrapping a
// deepeval metric's measure call) that implements this contract:
//
//	POST {BaseURL}/api/v1/evaluate
//	Authorization: Bearer <api key>          (only when WithAPIKey is set)
//	Content-Type: application/json
//
//	{"metric": "faithfulness", "input": "...", "actual_output": "...",
//	 "expected_output": "...", "context": ["..."]}
//
//	200 OK
//	{"score": 0.87, "reason": "...", "success": true}
//
// The score is clamped to [0.0, 1.0]. Without such a shim every Score call
// fails. The contract is defined by Beluga, not by the DeepEval project.
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
