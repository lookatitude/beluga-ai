// Package confluence provides a DocumentLoader that loads pages from Atlassian
// Confluence via its REST API. It implements the [loader.DocumentLoader] interface.
//
// # Registration
//
// The provider registers as "confluence" in the loader registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/confluence"
//
//	l, err := loader.New("confluence", config.ProviderConfig{
//	    APIKey:  "your-api-token",
//	    BaseURL: "https://your-domain.atlassian.net/wiki",
//	    Options: map[string]any{"user": "user@example.com"},
//	})
//	docs, err := l.Load(ctx, "12345") // page ID
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — Confluence API token (required)
//   - BaseURL — Confluence wiki base URL (required)
//   - Options["user"] — username for basic auth
package confluence
