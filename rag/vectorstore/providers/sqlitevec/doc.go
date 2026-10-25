//go:build cgo

// Package sqlitevec provides a VectorStore backed by SQLite with the
// sqlite-vec extension for vector similarity search.
//
// This provider requires CGO and the sqlite-vec extension. Build with:
//
//	CGO_ENABLED=1 go build
//
// # Registration
//
// The provider registers as "sqlitevec" in the vectorstore registry:
//
//	import _ "github.com/lookatitude/beluga-ai/v2/rag/vectorstore/providers/sqlitevec"
//
//	store, err := vectorstore.New("sqlitevec", config.ProviderConfig{
//	    BaseURL: "/path/to/database.db",
//	    Options: map[string]any{
//	        "table":     "documents",
//	        "dimension": float64(1536),
//	    },
//	})
//
// # Configuration
//
// ProviderConfig fields:
//   - BaseURL — path to the SQLite database file (required)
//   - Options["table"] — table name (default: "documents"); must match
//     ^[a-zA-Z_][a-zA-Z0-9_]*$ or construction fails with core.ErrInvalidInput
//   - Options["dimension"] — vector dimension (default: 1536)
//
// # Metadata Filtering
//
// vec0 applies its KNN limit before the JOIN with the metadata table, so
// metadata filters are evaluated on the KNN candidates. A filtered search
// over-fetches (k*10, growing 4x per pass, capped at vec0's maximum k of
// 4096) until it has k matching documents or runs out of candidates.
package sqlitevec
