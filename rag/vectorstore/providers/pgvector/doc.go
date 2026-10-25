// Package pgvector provides a VectorStore backed by PostgreSQL with the
// pgvector extension. It uses pgx for connection management and supports
// cosine, dot-product, and Euclidean distance strategies.
//
// # Registration
//
// The provider registers as "pgvector" in the vectorstore registry:
//
//	import _ "github.com/lookatitude/beluga-ai/v2/rag/vectorstore/providers/pgvector"
//
//	store, err := vectorstore.New("pgvector", config.ProviderConfig{
//	    BaseURL: "postgres://user:pass@localhost:5432/db",
//	    Options: map[string]any{
//	        "table":     "documents",
//	        "dimension": float64(1536),
//	    },
//	})
//
// # Configuration
//
// ProviderConfig fields:
//   - BaseURL — PostgreSQL connection string (required)
//   - Options["table"] — table name (default: "documents"); must match
//     ^[a-zA-Z_][a-zA-Z0-9_]*$ or construction fails with core.ErrInvalidInput
//   - Options["dimension"] — vector dimension (default: 1536)
//
// # Connection Lifecycle
//
// [NewFromConfig] creates a concurrency-safe pgxpool.Pool owned by the
// Store; call [Store.Close] to release it. A Store built with [New] uses the
// caller's pool, and Close leaves it open.
//
// # Table Management
//
// Use [Store.EnsureTable] to create the documents table and pgvector
// extension if they do not exist.
//
// # Distance Operators
//
// The provider maps [vectorstore.SearchStrategy] to pgvector SQL operators:
//   - Cosine — <=> (returns 1 - distance as similarity score)
//   - DotProduct — <#> (returns negated inner product)
//   - Euclidean — <-> (returns negated distance)
package pgvector
