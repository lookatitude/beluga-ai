// Package weaviate provides a VectorStore backed by the Weaviate vector database.
// It communicates with Weaviate via its REST and GraphQL APIs.
//
// # Registration
//
// The provider registers as "weaviate" in the vectorstore registry:
//
//	import _ "github.com/lookatitude/beluga-ai/v2/rag/vectorstore/providers/weaviate"
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
//   - Options["class"] — Weaviate class name (default: "Document"); must match
//     ^[A-Z][_0-9A-Za-z]*$
//
// # Filters and IDs
//
// Search filters are equality predicates. Keys must match
// ^[A-Za-z_][A-Za-z0-9_]*$ and values must be strings, bools or numbers;
// string values are emitted as escaped string literals and mapped to
// valueText, bools to valueBoolean, integers to valueInt and floats to
// valueNumber. GraphQL errors and per-object batch errors (both reported
// with HTTP 200) are returned as errors.
//
// Object UUIDs are name-based UUIDv5 values derived from the full document
// ID. Releases before this one truncated the ID instead, which made IDs that
// share a 16-byte prefix collide; re-index existing data after upgrading.
package weaviate
