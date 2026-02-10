// Package weaviate provides a VectorStore backed by the Weaviate vector database.
// It communicates with Weaviate via its REST API using the internal httpclient.
//
// # Registration
//
// The provider registers as "weaviate" in the vectorstore registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/weaviate"
//
//	store, err := vectorstore.New("weaviate", config.ProviderConfig{
//	    BaseURL: "http://localhost:8080",
//	    APIKey:  "optional-api-key",
//	    Options: map[string]any{
//	        "class": "Document",
//	    },
//	})
//
// # Configuration
//
// ProviderConfig fields:
//   - BaseURL — Weaviate server URL (required)
//   - APIKey — API key for authentication (optional)
//   - Options["class"] — Weaviate class name (default: "Document")
package weaviate
