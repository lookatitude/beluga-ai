// Package sentencetransformers provides an Embedder backed by the HuggingFace
// Inference API for Sentence Transformers models. It implements the
// [embedding.Embedder] interface using the feature-extraction pipeline endpoint.
//
// # Registration
//
// The provider registers as "sentence_transformers" in the embedding registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/embedding/providers/sentence_transformers"
//
//	emb, err := embedding.New("sentence_transformers", config.ProviderConfig{
//	    APIKey: "hf_...",
//	    Model:  "sentence-transformers/all-MiniLM-L6-v2",
//	})
//
// # Models
//
// Supported models and their default dimensions:
//   - sentence-transformers/all-MiniLM-L6-v2 (384, default)
//   - sentence-transformers/all-MiniLM-L12-v2 (384)
//   - sentence-transformers/all-mpnet-base-v2 (768)
//   - sentence-transformers/paraphrase-MiniLM-L6-v2 (384)
//   - BAAI/bge-small-en-v1.5 (384)
//   - BAAI/bge-base-en-v1.5 (768)
//   - BAAI/bge-large-en-v1.5 (1024)
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — HuggingFace API token (required)
//   - Model — model name (default: "sentence-transformers/all-MiniLM-L6-v2")
//   - BaseURL — API base URL (default: "https://api-inference.huggingface.co")
//   - Timeout — request timeout
//   - Options["dimensions"] — override output dimensionality
package sentencetransformers
