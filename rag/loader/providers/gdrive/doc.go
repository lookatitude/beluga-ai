// Package gdrive provides a DocumentLoader that loads files from Google Drive
// via the Google Drive REST API. It implements the [loader.DocumentLoader]
// interface using direct HTTP calls.
//
// # Registration
//
// The provider registers as "gdrive" in the loader registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/gdrive"
//
//	l, err := loader.New("gdrive", config.ProviderConfig{
//	    APIKey: "your-api-key-or-oauth-token",
//	})
//	docs, err := l.Load(ctx, "file-id-here")
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — Google API key or OAuth token (required)
package gdrive
