// Package vespa provides a VectorStore backed by the Vespa search engine.
// It communicates with Vespa's document and search APIs via HTTP REST,
// supporting cosine, dot-product, and Euclidean distance strategies.
//
// # Registration
//
// The provider registers as "vespa" in the vectorstore registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/vespa"
//
//	store, err := vectorstore.New("vespa", config.ProviderConfig{
//	    BaseURL: "http://localhost:8080",
//	    Options: map[string]any{
//	        "namespace":  "my_namespace",
//	        "doc_type":   "my_doc_type",
//	    },
//	})
//
// # Configuration
//
// ProviderConfig fields:
//   - BaseURL — Vespa endpoint URL (required)
//   - Options["namespace"] — document namespace
//   - Options["doc_type"] — document type
package vespa
