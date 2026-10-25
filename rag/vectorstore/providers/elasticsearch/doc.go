// Package elasticsearch provides a VectorStore backed by Elasticsearch's kNN search.
// It uses Elasticsearch's dense_vector field type and approximate kNN search.
//
// # Registration
//
// The provider registers as "elasticsearch" in the vectorstore registry:
//
//	import _ "github.com/lookatitude/beluga-ai/v2/rag/vectorstore/providers/elasticsearch"
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
//
// # Metadata
//
// Document metadata is stored under a "metadata" object, so metadata keys
// never collide with the "content" and "embedding" fields. EnsureIndex maps
// every string under "metadata" to "keyword", and metadata filters are
// exact-match term queries on "metadata.<key>". Filter keys must match
// ^[A-Za-z_][A-Za-z0-9_.-]*$ or Search returns core.ErrInvalidInput.
//
// Indexes written by earlier versions stored metadata as top-level fields.
// Search still returns those fields as metadata, but filters only match the
// nested layout, so such indexes should be re-indexed.
//
// # HTTP Client
//
// The default HTTP client has a 30 second timeout; supply another with
// [WithHTTPClient]. Response bodies larger than 32 MiB are rejected.
package elasticsearch
