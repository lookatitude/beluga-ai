// Package jina provides a Jina AI embeddings provider for the Beluga AI framework.
// It implements the [embedding.Embedder] interface using the Jina Embeddings API
// via the internal httpclient.
//
// # Registration
//
// The provider registers as "jina" in the embedding registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/jina"
//
//	emb, err := embedding.New("jina", config.ProviderConfig{
//	    APIKey: "...",
//	})
//
// # Models
//
// Supported models and their default dimensions:
//   - jina-embeddings-v2-base-en (768, default)
//   - jina-embeddings-v2-small-en (512)
//   - jina-embeddings-v2-base-de (768)
//   - jina-embeddings-v2-base-zh (768)
//   - jina-embeddings-v3 (1024)
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — Jina AI API key (required)
//   - Model — model name (default: "jina-embeddings-v2-base-en")
//   - BaseURL — API base URL (default: "https://api.jina.ai/v1")
//   - Timeout — request timeout
//   - Options["dimensions"] — override output dimensionality
package jina
