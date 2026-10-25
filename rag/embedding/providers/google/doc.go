// Package google provides a Google AI embeddings provider for the Beluga AI framework.
// It implements the [embedding.Embedder] interface using the internal httpclient
// to call the Google AI Gemini embedding API.
//
// # Registration
//
// The provider registers as "google" in the embedding registry:
//
//	import _ "github.com/lookatitude/beluga-ai/v2/rag/embedding/providers/google"
//
//	emb, err := embedding.New("google", config.ProviderConfig{
//	    APIKey: "...",
//	})
//
// # Models
//
// Supported models and their default dimensions:
//   - gemini-embedding-001 (3072, default)
//   - text-embedding-004 (768, deprecated)
//   - embedding-001 (768)
//   - text-multilingual-embedding-002 (768)
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — Google AI API key (required); sent in the x-goog-api-key header
//   - Model — model name (default: "gemini-embedding-001")
//   - BaseURL — API base URL (default: "https://generativelanguage.googleapis.com/v1beta")
//   - Timeout — request timeout
//   - Options["dimensions"] — override output dimensionality
package google
