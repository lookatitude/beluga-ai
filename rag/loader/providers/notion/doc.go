// Package notion provides a DocumentLoader that loads pages from Notion via
// its API. It implements the [loader.DocumentLoader] interface using direct HTTP
// calls to the Notion API.
//
// # Registration
//
// The provider registers as "notion" in the loader registry:
//
//	import _ "github.com/lookatitude/beluga-ai/v2/rag/loader/providers/notion"
//
//	l, err := loader.New("notion", config.ProviderConfig{
//	    APIKey: "ntn_...",
//	})
//	docs, err := l.Load(ctx, "page-id-here")
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — Notion integration token (required)
package notion
