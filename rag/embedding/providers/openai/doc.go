// Package openai provides an OpenAI embeddings provider for the Beluga AI framework.
// It implements the [embedding.Embedder] interface using the openai-go SDK.
//
// # Registration
//
// The provider registers as "openai" in the embedding registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/openai"
//
//	emb, err := embedding.New("openai", config.ProviderConfig{
//	    APIKey: "sk-...",
//	})
//
// # Models
//
// Supported models and their default dimensions:
//   - text-embedding-3-small (1536, default)
//   - text-embedding-3-large (3072)
//   - text-embedding-ada-002 (1536)
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — OpenAI API key (required)
//   - Model — model name (default: "text-embedding-3-small")
//   - BaseURL — API base URL (default: "https://api.openai.com/v1")
//   - Timeout — request timeout
//   - Options["dimensions"] — override output dimensionality
package openai
