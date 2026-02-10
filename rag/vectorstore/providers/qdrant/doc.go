// Package qdrant provides a VectorStore backed by the Qdrant vector database.
// It communicates with Qdrant via its HTTP REST API to avoid heavy gRPC
// dependencies, and supports cosine, dot-product, and Euclidean distance.
//
// # Registration
//
// The provider registers as "qdrant" in the vectorstore registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/qdrant"
//
//	store, err := vectorstore.New("qdrant", config.ProviderConfig{
//	    BaseURL: "http://localhost:6333",
//	    APIKey:  "optional-api-key",
//	    Options: map[string]any{
//	        "collection": "my_collection",
//	        "dimension":  float64(1536),
//	    },
//	})
//
// # Configuration
//
// ProviderConfig fields:
//   - BaseURL — Qdrant server URL (required)
//   - APIKey — API key for authentication (optional)
//   - Options["collection"] — collection name (required)
//   - Options["dimension"] — vector dimension (default: 1536)
package qdrant
