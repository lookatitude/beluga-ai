// Package ragas provides RAGAS (Retrieval Augmented Generation Assessment)
// evaluation metrics for the Beluga AI eval framework. It implements the
// eval.Metric interface and sends evaluation requests to a RAGAS API endpoint.
//
// RAGAS provides metrics for evaluating RAG pipelines including faithfulness,
// answer relevancy, context precision, and context recall.
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
