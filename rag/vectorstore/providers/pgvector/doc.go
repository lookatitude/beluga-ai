// Package pgvector provides a VectorStore backed by PostgreSQL with the
// pgvector extension. It uses pgx for connection management and supports
// cosine, dot-product, and Euclidean distance strategies.
//
// # Registration
//
// The provider registers as "pgvector" in the vectorstore registry:
//
//	import _ "github.com/lookatitude/beluga-ai/rag/vectorstore/providers/pgvector"
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
//   - Options["table"] — table name (default: "documents")
//   - Options["dimension"] — vector dimension (default: 1536)
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
