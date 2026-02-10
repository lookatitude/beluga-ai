// Package redis provides a VectorStore backed by Redis with the RediSearch module.
// It uses Redis hashes to store documents and RediSearch's vector similarity
// search for retrieval.
//
// # Registration
//
// The provider registers as "redis" in the vectorstore registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/redis"
//
//	store, err := vectorstore.New("redis", config.ProviderConfig{
//	    BaseURL: "localhost:6379",
//	    Options: map[string]any{
//	        "index":     "idx:documents",
//	        "prefix":    "doc:",
//	        "dimension": float64(1536),
//	    },
//	})
//
// # Configuration
//
// ProviderConfig fields:
//   - BaseURL — Redis server address (required)
//   - Options["index"] — RediSearch index name (default: "idx:documents")
//   - Options["prefix"] — key prefix for document hashes (default: "doc:")
//   - Options["dimension"] — vector dimension (default: 1536)
package redis
