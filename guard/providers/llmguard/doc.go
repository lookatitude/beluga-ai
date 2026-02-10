// Package llmguard provides an LLM Guard API guard implementation for the
// Beluga AI safety pipeline. It implements the guard.Guard interface and sends
// content validation requests to an LLM Guard API endpoint.
//
// LLM Guard provides prompt injection detection, toxicity filtering, and
// sensitive data detection via its REST API. It uses the /analyze/prompt
// endpoint for input content and /analyze/output for output content.
//
// # Configuration
//
// The guard is configured using functional options:
//
//   - WithBaseURL sets the LLM Guard API base URL. Defaults to
//     "http://localhost:8000".
//   - WithAPIKey sets the API key for authentication (optional).
//   - WithTimeout sets the HTTP client timeout. Defaults to 15 seconds.
//
// # Usage
//
//	g, err := llmguard.New(
//	    llmguard.WithBaseURL("http://localhost:8000"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	result, err := g.Validate(ctx, guard.GuardInput{Content: text, Role: "input"})
package llmguard
