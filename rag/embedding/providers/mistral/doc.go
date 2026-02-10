// Package mistral provides an Embedder backed by the Mistral AI embeddings API.
// It implements the [embedding.Embedder] interface using Mistral's embed endpoint
// via the internal httpclient.
//
// # Registration
//
// The provider registers as "mistral" in the embedding registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/mistral"
//
//	emb, err := embedding.New("mistral", config.ProviderConfig{
//	    APIKey: "...",
//	    Model:  "mistral-embed",
//	})
//
// # Models
//
// Supported models and their default dimensions:
//   - mistral-embed (1024, default)
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — Mistral AI API key (required)
//   - Model — model name (default: "mistral-embed")
//   - BaseURL — API base URL (default: "https://api.mistral.ai/v1")
//   - Timeout — request timeout
//   - Options["dimensions"] — override output dimensionality
package mistral
