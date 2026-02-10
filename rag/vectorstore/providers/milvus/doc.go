// Package milvus provides a VectorStore backed by the Milvus vector database.
// It communicates with Milvus via its REST API for broad compatibility.
//
// # Registration
//
// The provider registers as "milvus" in the vectorstore registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/milvus"
//
//	store, err := vectorstore.New("milvus", config.ProviderConfig{
//	    BaseURL: "http://localhost:19530",
//	    Options: map[string]any{
//	        "collection": "documents",
//	        "dimension":  float64(1536),
//	    },
//	})
//
// # Configuration
//
// ProviderConfig fields:
//   - BaseURL — Milvus server URL (required)
//   - APIKey — API key for authentication (optional)
//   - Options["collection"] — collection name (default: "documents")
//   - Options["dimension"] — vector dimension (default: 1536)
package milvus
