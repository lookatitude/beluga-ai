// Package azuresafety provides an Azure Content Safety guard implementation for
// the Beluga AI safety pipeline. It implements the guard.Guard interface and
// sends content validation requests to the Azure Content Safety API.
//
// Azure Content Safety provides text moderation across categories including
// Hate, SelfHarm, Sexual, and Violence with configurable severity thresholds.
//
// # Configuration
//
// The guard is configured using functional options:
//
//   - WithEndpoint sets the Azure Content Safety endpoint URL (required).
//   - WithAPIKey sets the API key for authentication (required).
//   - WithThreshold sets the severity threshold (0-6); content at or above
//     this severity in any category is blocked. Defaults to 2.
//   - WithTimeout sets the HTTP client timeout. Defaults to 15 seconds.
//
// # Usage
//
//	g, err := azuresafety.New(
//	    azuresafety.WithEndpoint("https://myinstance.cognitiveservices.azure.com"),
//	    azuresafety.WithAPIKey("key-..."),
//	    azuresafety.WithThreshold(4),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	result, err := g.Validate(ctx, guard.GuardInput{Content: text, Role: "input"})
package azuresafety
