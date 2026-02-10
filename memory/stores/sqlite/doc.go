// Package sqlite provides a SQLite-backed implementation of [memory.MessageStore].
// Messages are stored in a table with columns for role, content (JSON), metadata
// (JSON), and created_at timestamp. This uses the pure-Go modernc.org/sqlite
// driver (no CGO required).
//
// # Usage
//
//	import (
//	    "database/sql"
//	    _ "modernc.org/sqlite"
//	    "github.com/lookatitude/beluga-ai/memory/stores/sqlite"
//	)
//
//	db, err := sql.Open("sqlite", ":memory:")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	store, err := sqlite.New(sqlite.Config{DB: db})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	err = store.EnsureTable(ctx) // auto-create table if needed
//
// # Schema
//
// The auto-created table has the following columns:
//
//   - id: INTEGER PRIMARY KEY AUTOINCREMENT
//   - role: TEXT NOT NULL
//   - content: TEXT NOT NULL (JSON)
//   - metadata: TEXT (JSON)
//   - created_at: TEXT NOT NULL DEFAULT datetime('now')
//
// Use [MessageStore.EnsureTable] to create it automatically.
package sqlite
