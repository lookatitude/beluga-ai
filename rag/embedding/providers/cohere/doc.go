// Package cohere provides a Cohere embeddings provider for the Beluga AI framework.
// It implements the [embedding.Embedder] interface using the Cohere Embed API
// via the internal httpclient.
//
// # Registration
//
// The provider registers as "cohere" in the embedding registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/cohere"
//
//	emb, err := embedding.New("cohere", config.ProviderConfig{
//	    APIKey: "...",
//	})
//
// # Models
//
// Supported models and their default dimensions:
//   - embed-english-v3.0 (1024, default)
//   - embed-multilingual-v3.0 (1024)
//   - embed-english-light-v3.0 (384)
//   - embed-multilingual-light-v3.0 (384)
//   - embed-english-v2.0 (4096)
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — Cohere API key (required)
//   - Model — model name (default: "embed-english-v3.0")
//   - BaseURL — API base URL (default: "https://api.cohere.com/v2")
//   - Timeout — request timeout
//   - Options["dimensions"] — override output dimensionality
//   - Options["input_type"] — input type hint (default: "search_document")
package cohere
