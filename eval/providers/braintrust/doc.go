// Package braintrust provides a Braintrust evaluation metric for the Beluga AI
// eval framework. It implements the eval.Metric interface and sends evaluation
// requests to the Braintrust API.
//
// Braintrust provides evaluation scoring for LLM outputs including
// factuality, relevance, and custom scoring functions.
//
// # Endpoint compatibility (preview)
//
// This adapter POSTs to {BaseURL}/v1/score. That endpoint is NOT part of the
// public Braintrust REST API: Braintrust scores outputs by invoking functions
// (POST /v1/function/{function_id}/invoke, or the SDKs' invoke with a project
// slug or a global scorer name such as "Factuality"). Against the default
// BaseURL "https://api.braintrust.dev" every Score call is therefore expected
// to fail. Until the adapter is ported to the function-invoke API, point
// WithBaseURL at a self-hosted shim that implements this Beluga-defined
// contract:
//
//	POST {BaseURL}/v1/score
//	Authorization: Bearer <api key>
//	Content-Type: application/json
//
//	{"project_name": "default",
//	 "scores": [{"name": "factuality", "input": "...", "output": "...",
//	             "expected": "...", "context": ["..."]}]}
//
//	200 OK
//	{"results": [{"name": "factuality", "score": 0.8}]}
//
// The first result's score is used and clamped to [0.0, 1.0].
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
