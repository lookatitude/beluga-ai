// Package turbopuffer provides a VectorStore backed by the Turbopuffer
// serverless vector database. It communicates with Turbopuffer via its REST API.
//
// # Registration
//
// The provider registers as "turbopuffer" in the vectorstore registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/turbopuffer"
//
//	store, err := vectorstore.New("turbopuffer", config.ProviderConfig{
//	    APIKey: "...",
//	    Options: map[string]any{
//	        "namespace": "my_namespace",
//	    },
//	})
//
// # Configuration
//
// ProviderConfig fields:
//   - APIKey — Turbopuffer API key (required)
//   - Options["namespace"] — namespace for document isolation
package turbopuffer
