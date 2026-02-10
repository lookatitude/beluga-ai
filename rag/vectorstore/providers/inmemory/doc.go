// Package inmemory provides an in-memory VectorStore for testing and
// small-scale use. It uses linear scan with cosine similarity for search
// and is safe for concurrent use.
//
// # Registration
//
// The provider registers as "inmemory" in the vectorstore registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/inmemory"
//
//	store, err := vectorstore.New("inmemory", config.ProviderConfig{})
//
// # Features
//
//   - Thread-safe with sync.RWMutex
//   - Supports cosine, dot-product, and Euclidean distance strategies
//   - Documents keyed by ID; re-adding overwrites the previous entry
//   - No external dependencies
package inmemory
