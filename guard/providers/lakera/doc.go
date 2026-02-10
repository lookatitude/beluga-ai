// Package lakera provides a Lakera Guard API guard implementation for the
// Beluga AI safety pipeline. It implements the guard.Guard interface and sends
// content validation requests to the Lakera Guard API endpoint.
//
// Lakera Guard detects prompt injections, jailbreaks, PII, and harmful content.
//
// # Configuration
//
// The guard is configured using functional options:
//
//   - WithAPIKey sets the Lakera Guard API key (required).
//   - WithBaseURL sets the API base URL. Defaults to "https://api.lakera.ai".
//   - WithTimeout sets the HTTP client timeout. Defaults to 15 seconds.
//
// # Usage
//
//	g, err := lakera.New(
//	    lakera.WithAPIKey("lk-..."),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	result, err := g.Validate(ctx, guard.GuardInput{Content: text, Role: "input"})
package lakera
