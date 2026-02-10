// Package chroma provides a VectorStore backed by ChromaDB. It communicates
// with ChromaDB via its HTTP REST API.
//
// # Registration
//
// The provider registers as "chroma" in the vectorstore registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/chroma"
//
//	store, err := vectorstore.New("chroma", config.ProviderConfig{
//	    BaseURL: "http://localhost:8000",
//	    Options: map[string]any{
//	        "collection": "my_collection",
//	        "tenant":     "default_tenant",
//	        "database":   "default_database",
//	    },
//	})
//
// # Configuration
//
// ProviderConfig fields:
//   - BaseURL — ChromaDB server URL (required)
//   - Options["collection"] — collection name
//   - Options["tenant"] — tenant name (default: "default_tenant")
//   - Options["database"] — database name (default: "default_database")
//
// # Collection Management
//
// Use [Store.EnsureCollection] to create the collection if it does not exist.
// The collection ID is resolved automatically on first Add or Search call.
package chroma
