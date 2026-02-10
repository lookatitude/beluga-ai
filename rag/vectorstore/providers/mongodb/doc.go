// Package mongodb provides a VectorStore backed by MongoDB Atlas Vector Search.
// It communicates with MongoDB via its HTTP Data API to avoid requiring the
// full MongoDB Go driver as a dependency, and supports cosine, dot-product,
// and Euclidean distance strategies.
//
// # Registration
//
// The provider registers as "mongodb" in the vectorstore registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/mongodb"
//
//	store, err := vectorstore.New("mongodb", config.ProviderConfig{
//	    BaseURL: "https://data.mongodb-api.com/app/<app-id>/endpoint/data/v1",
//	    APIKey:  "your-api-key",
//	    Options: map[string]any{
//	        "database":   "my_db",
//	        "collection": "documents",
//	        "index":      "vector_index",
//	    },
//	})
//
// # Configuration
//
// ProviderConfig fields:
//   - BaseURL — MongoDB Data API endpoint (required)
//   - APIKey — API key for authentication (required)
//   - Options["database"] — database name (default: "beluga")
//   - Options["collection"] — collection name (default: "documents")
//   - Options["index"] — vector search index name (default: "vector_index")
package mongodb
