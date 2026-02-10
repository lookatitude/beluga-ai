// Package github provides a DocumentLoader that loads files from GitHub
// repositories via the GitHub API. It implements the [loader.DocumentLoader]
// interface using direct HTTP calls.
//
// # Registration
//
// The provider registers as "github" in the loader registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/loader/providers/github"
//
//	l, err := loader.New("github", config.ProviderConfig{
//	    APIKey: "ghp_...",
//	})
//	docs, err := l.Load(ctx, "owner/repo/path/to/file.go")
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — GitHub personal access token (required)
package github
