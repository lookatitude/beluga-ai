// Package inmemory provides a deterministic hash-based Embedder for testing.
// It generates reproducible embeddings by hashing the input text with FNV-1a,
// making it suitable for unit tests and local development without external API
// calls. The resulting vectors are normalized to unit length.
//
// # Registration
//
// The provider registers as "inmemory" in the embedding registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/inmemory"
//
//	emb, err := embedding.New("inmemory", config.ProviderConfig{})
//
// # Configuration
//
// ProviderConfig fields:
//   - Options["dimensions"] — vector size (default: 128)
package inmemory
