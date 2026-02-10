// Package braintrust provides a Braintrust evaluation metric for the Beluga AI
// eval framework. It implements the eval.Metric interface and sends evaluation
// requests to the Braintrust API.
//
// Braintrust provides evaluation scoring for LLM outputs including
// factuality, relevance, and custom scoring functions.
//
// # Configuration
//
// The metric is configured using functional options:
//
//   - WithAPIKey sets the Braintrust API key (required).
//   - WithProjectName sets the Braintrust project name. Defaults to "default".
//   - WithMetricName sets the metric to evaluate. Defaults to "factuality".
//   - WithBaseURL sets the API base URL. Defaults to "https://api.braintrust.dev".
//   - WithTimeout sets the HTTP client timeout. Defaults to 30 seconds.
//
// The metric Name is prefixed with "braintrust_" (e.g., "braintrust_factuality").
//
// # Usage
//
//	metric, err := braintrust.New(
//	    braintrust.WithAPIKey("bt-..."),
//	    braintrust.WithProjectName("my-project"),
//	    braintrust.WithMetricName("factuality"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	score, err := metric.Score(ctx, sample)
package braintrust
