// Package ragas provides RAGAS (Retrieval Augmented Generation Assessment)
// evaluation metrics for the Beluga AI eval framework. It implements the
// eval.Metric interface and sends evaluation requests to a RAGAS API endpoint.
//
// RAGAS provides metrics for evaluating RAG pipelines including faithfulness,
// answer relevancy, context precision, and context recall.
//
// # Requires a user-provided HTTP shim
//
// RAGAS is a Python library; the RAGAS project does not ship an HTTP server or
// a hosted API. This adapter therefore does NOT talk to any vendor endpoint.
// It expects a small self-hosted service at BaseURL (for example a FastAPI app
// wrapping ragas.evaluate) that implements this contract:
//
//	POST {BaseURL}/api/v1/evaluate
//	Authorization: Bearer <api key>          (only when WithAPIKey is set)
//	Content-Type: application/json
//
//	{"metric_name": "faithfulness",
//	 "data": [{"question": "...", "answer": "...",
//	           "contexts": ["..."], "ground_truth": "..."}]}
//
//	200 OK
//	{"scores": [{"metric_name": "faithfulness", "score": 0.93}]}
//
// The first returned score is used and clamped to [0.0, 1.0]. Without such a
// shim every Score call fails. The contract is defined by Beluga, not by the
// RAGAS project.
//
// # Configuration
//
// The metric is configured using functional options:
//
//   - WithBaseURL sets the RAGAS API base URL. Defaults to
//     "http://localhost:8080".
//   - WithAPIKey sets the API key for authentication (optional).
//   - WithMetricName sets the metric to evaluate (e.g., "faithfulness",
//     "answer_relevancy", "context_precision", "context_recall"). Defaults
//     to "faithfulness".
//   - WithTimeout sets the HTTP client timeout. Defaults to 30 seconds.
//
// The metric Name is prefixed with "ragas_" (e.g., "ragas_faithfulness").
//
// # Usage
//
//	metric, err := ragas.New(
//	    ragas.WithBaseURL("http://localhost:8080"),
//	    ragas.WithMetricName("faithfulness"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	score, err := metric.Score(ctx, sample)
package ragas
