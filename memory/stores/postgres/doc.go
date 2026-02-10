// Package postgres provides a PostgreSQL-backed implementation of
// [memory.MessageStore]. Messages are stored in a table with columns for role,
// content (JSONB), metadata (JSONB), and created_at timestamp. Search uses
// case-insensitive ILIKE queries on the content column.
//
// This implementation uses github.com/jackc/pgx/v5 as the PostgreSQL driver.
//
// # Usage
//
//	import "github.com/lookatitude/beluga-ai/memory/stores/postgres"
//
//	store, err := postgres.New(postgres.Config{
//	    DB:    pgxConn,       // *pgx.Conn, pgxpool.Pool, or pgxmock
//	    Table: "messages",    // optional, this is the default
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	err = store.EnsureTable(ctx) // auto-create table if needed
//
// # DBTX Interface
//
// The store accepts any value satisfying the [DBTX] interface, which is
// implemented by pgx.Conn, pgxpool.Pool, and pgxmock for testing.
//
// # Schema
//
// The auto-created table has the following columns:
//
//   - id: SERIAL PRIMARY KEY
//   - role: TEXT NOT NULL
//   - content: JSONB NOT NULL
//   - metadata: JSONB
//   - created_at: TIMESTAMPTZ NOT NULL DEFAULT NOW()
//
// Use [MessageStore.EnsureTable] to create it automatically.
package postgres
