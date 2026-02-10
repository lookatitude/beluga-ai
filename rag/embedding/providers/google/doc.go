// Package google provides a Google AI embeddings provider for the Beluga AI framework.
// It implements the [embedding.Embedder] interface using the internal httpclient
// to call the Google AI Gemini embedding API.
//
// # Registration
//
// The provider registers as "google" in the embedding registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/google"
//
//	emb, err := embedding.New("google", config.ProviderConfig{
//	    APIKey: "...",
//	})
//
// # Models
//
// Supported models and their default dimensions:
//   - text-embedding-004 (768, default)
//   - embedding-001 (768)
//   - text-multilingual-embedding-002 (768)
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — Google AI API key (required)
//   - Model — model name (default: "text-embedding-004")
//   - BaseURL — API base URL (default: "https://generativelanguage.googleapis.com/v1beta")
//   - Timeout — request timeout
//   - Options["dimensions"] — override output dimensionality
package google
