// Package firecrawl provides a DocumentLoader that uses Firecrawl to crawl
// websites and extract their content as markdown.
//
// Firecrawl (https://firecrawl.dev) is a web scraping service that handles
// JavaScript rendering, anti-bot detection, and returns clean markdown.
//
// # Registration
//
// The provider registers as "firecrawl" in the loader registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/firecrawl"
//
//	l, err := loader.New("firecrawl", config.ProviderConfig{
//	    APIKey: "fc-...",
//	})
//	docs, err := l.Load(ctx, "https://example.com")
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — Firecrawl API key (required)
package firecrawl
