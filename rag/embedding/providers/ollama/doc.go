// Package ollama provides an Ollama embeddings provider for the Beluga AI framework.
// It implements the [embedding.Embedder] interface using the Ollama REST API
// via the internal httpclient. Ollama enables running embedding models locally
// without external API calls.
//
// # Registration
//
// The provider registers as "ollama" in the embedding registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/ollama"
//
//	emb, err := embedding.New("ollama", config.ProviderConfig{
//	    BaseURL: "http://localhost:11434",
//	})
//
// # Models
//
// Supported models and their default dimensions:
//   - nomic-embed-text (768, default)
//   - mxbai-embed-large (1024)
//   - all-minilm (384)
//   - snowflake-arctic-embed (1024)
//
// # Configuration
//
// ProviderConfig fields:
//   - Model — model name (default: "nomic-embed-text")
//   - BaseURL — Ollama server URL (default: "http://localhost:11434")
//   - Timeout — request timeout
//   - Options["dimensions"] — override output dimensionality
package ollama
