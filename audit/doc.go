// Package audit provides structured audit logging for Beluga AI v2.
//
// # Overview
//
// The audit package records structured [Entry] values representing significant
// actions performed by agents, tools, and the broader system. Each entry
// captures who performed an action, when, on what session, and whether it
// succeeded.
//
// # Interfaces
//
// [Logger] is the minimal write-only interface:
//
//	type Logger interface {
//	    Log(ctx context.Context, entry Entry) error
//	}
//
// [Store] extends Logger with queryable history:
//
//	type Store interface {
//	    Logger
//	    Query(ctx context.Context, filter Filter) ([]Entry, error)
//	}
//
// # Registry Pattern
//
// Store backends are registered via [Register] (called from init()) and
// created with [New]:
//
//	store, err := audit.New("inmemory", audit.Config{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Use [List] to discover all registered backends.
//
// # Logging Entries
//
//	err := store.Log(ctx, audit.Entry{
//	    TenantID:  "tenant-abc",
//	    AgentID:   "agent-xyz",
//	    SessionID: "session-001",
//	    Action:    "tool.execute",
//	    Input:     inputPayload,
//	    Output:    outputPayload,
//	    Duration:  250 * time.Millisecond,
//	})
//
// IDs and timestamps are generated automatically when the entry's ID or
// Timestamp fields are zero.
//
// # Querying Entries
//
//	entries, err := store.Query(ctx, audit.Filter{
//	    TenantID: "tenant-abc",
//	    Since:    time.Now().Add(-24 * time.Hour),
//	    Limit:    100,
//	})
//
// All filter fields are optional. An empty [Filter] returns all entries
// (subject to [Filter.Limit]).
//
// # Built-in Backends
//
//   - "inmemory" — thread-safe in-memory store (development/testing)
package audit
