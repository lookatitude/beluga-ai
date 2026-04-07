// Package audit provides structured audit logging for Beluga AI v2.
//
// # Overview
//
// The audit package records structured [Entry] values representing significant
// actions performed by agents, tools, and the broader system. Each entry
// captures who performed an action, when, in which session, and whether it
// succeeded. The [Logger] interface is write-only; [Store] extends it with
// queryable history.
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
// Store backends register via [Register] (called from init()) and are created
// with [New]. The "inmemory" backend is registered automatically.
//
//	store, err := audit.New("inmemory", audit.Config{})
//	if err != nil {
//	    return err
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
//	    Input:     redactedInput,
//	    Output:    redactedOutput,
//	    Duration:  250 * time.Millisecond,
//	})
//	if err != nil {
//	    return err
//	}
//
// IDs and timestamps are generated automatically when the entry's ID or
// Timestamp fields are zero. IDs are generated with crypto/rand.
//
// # Security
//
// The [Entry.Input] and [Entry.Output] fields accept any value. Callers MUST
// redact PII, API keys, secrets, and passwords before calling Log. Unredacted
// sensitive data in audit logs creates an information disclosure risk.
//
// # Querying Entries
//
//	entries, err := store.Query(ctx, audit.Filter{
//	    TenantID: "tenant-abc",
//	    Action:   "tool.execute",
//	    Since:    time.Now().Add(-24 * time.Hour),
//	    Limit:    100,
//	})
//	if err != nil {
//	    return err
//	}
//
// All filter fields are optional. An empty [Filter] returns all entries
// (subject to [Filter.Limit]).
//
// # Extension
//
// Implement [Store] and register in init():
//
//	func init() {
//	    audit.Register("postgres", func(cfg audit.Config) (audit.Store, error) {
//	        return newPostgresStore(cfg)
//	    })
//	}
//
// Note: [Register] panics on duplicate registration. Each backend name must be
// unique across all imported packages.
//
// # Built-in Backends
//
//   - "inmemory" — thread-safe in-memory store (development/testing)
//
// # Related packages
//
//   - [github.com/lookatitude/beluga-ai/runtime/plugins] — AuditPlugin
//   - [github.com/lookatitude/beluga-ai/cost] — cost tracking
package audit
