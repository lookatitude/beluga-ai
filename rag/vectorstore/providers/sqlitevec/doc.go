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
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/sqlitevec"
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
//   - Options["table"] — table name (default: "documents")
//   - Options["dimension"] — vector dimension (default: 1536)
package sqlitevec
