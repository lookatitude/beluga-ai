// Package elasticsearch provides a VectorStore backed by Elasticsearch's kNN search.
// It uses Elasticsearch's dense_vector field type and approximate kNN search.
//
// # Registration
//
// The provider registers as "elasticsearch" in the vectorstore registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/elasticsearch"
//
//	store, err := vectorstore.New("elasticsearch", config.ProviderConfig{
//	    BaseURL: "http://localhost:9200",
//	    APIKey:  "optional-api-key",
//	    Options: map[string]any{
//	        "index":     "documents",
//	        "dimension": float64(1536),
//	    },
//	})
//
// # Configuration
//
// ProviderConfig fields:
//   - BaseURL — Elasticsearch server URL (required)
//   - APIKey — API key for authentication (optional)
//   - Options["index"] — index name (default: "documents")
//   - Options["dimension"] — vector dimension (default: 1536)
//
// # Index Management
//
// Use [Store.EnsureIndex] to create the Elasticsearch index with the
// appropriate dense_vector mapping if it does not exist.
package elasticsearch
