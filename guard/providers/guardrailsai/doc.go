// Package guardrailsai provides a Guardrails AI guard implementation for the
// Beluga AI safety pipeline. It implements the guard.Guard interface and sends
// content validation requests to a Guardrails AI API endpoint.
//
// Guardrails AI provides validators for PII detection, toxicity, hallucination,
// prompt injection, and custom rules defined via RAIL specifications.
//
// # Configuration
//
// The guard is configured using functional options:
//
//   - WithBaseURL sets the Guardrails AI API base URL. Defaults to
//     "http://localhost:8000".
//   - WithAPIKey sets the API key for authentication (optional).
//   - WithGuardName sets the guard name to invoke on the server. Defaults to
//     "default".
//   - WithTimeout sets the HTTP client timeout. Defaults to 15 seconds.
//
// # Usage
//
//	g, err := guardrailsai.New(
//	    guardrailsai.WithBaseURL("http://localhost:8000"),
//	    guardrailsai.WithGuardName("my-guard"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	result, err := g.Validate(ctx, guard.GuardInput{Content: text, Role: "output"})
package guardrailsai
