// Package voyage provides a Voyage AI embeddings provider for the Beluga AI framework.
// It implements the [embedding.Embedder] interface using the Voyage Embed API
// via the internal httpclient.
//
// # Registration
//
// The provider registers as "voyage" in the embedding registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/voyage"
//
//	emb, err := embedding.New("voyage", config.ProviderConfig{
//	    APIKey: "...",
//	})
//
// # Models
//
// Supported models and their default dimensions:
//   - voyage-2 (1024, default)
//   - voyage-large-2 (1536)
//   - voyage-code-2 (1536)
//   - voyage-lite-02-instruct (1024)
//   - voyage-3 (1024)
//   - voyage-3-lite (512)
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — Voyage AI API key (required)
//   - Model — model name (default: "voyage-2")
//   - BaseURL — API base URL (default: "https://api.voyageai.com/v1")
//   - Timeout — request timeout
//   - Options["dimensions"] — override output dimensionality
//   - Options["input_type"] — input type hint (default: "document")
package voyage
