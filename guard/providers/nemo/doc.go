// Package nemo provides an NVIDIA NeMo Guardrails guard implementation for the
// Beluga AI safety pipeline. It implements the guard.Guard interface and sends
// content validation requests to a NeMo Guardrails API endpoint.
//
// NeMo Guardrails can be configured to check for topic safety, jailbreak
// detection, fact-checking, and more via Colang configurations.
//
// # Configuration
//
// The guard is configured using functional options:
//
//   - WithBaseURL sets the NeMo Guardrails API base URL. Defaults to
//     "http://localhost:8080".
//   - WithAPIKey sets the API key for authentication (optional).
//   - WithConfigID sets the guardrails configuration ID. Defaults to "default".
//   - WithTimeout sets the HTTP client timeout. Defaults to 15 seconds.
//
// # Usage
//
//	g, err := nemo.New(
//	    nemo.WithBaseURL("http://localhost:8080"),
//	    nemo.WithConfigID("my-config"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	result, err := g.Validate(ctx, guard.GuardInput{Content: text, Role: "input"})
package nemo
