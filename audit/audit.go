package audit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Entry is a single structured audit log record describing one action taken
// within the system.
type Entry struct {
	// ID is a unique identifier for this entry. If empty when passed to Log,
	// an implementation may generate one automatically.
	ID string

	// Timestamp records when the action occurred. If zero when passed to Log,
	// an implementation may set it to the current time.
	Timestamp time.Time

	// TenantID identifies the tenant that owns this entry.
	TenantID string

	// AgentID identifies the agent that performed the action.
	AgentID string

	// SessionID identifies the session in which the action occurred.
	SessionID string

	// Action is a short dot-separated identifier describing what happened,
	// e.g. "tool.execute" or "llm.generate".
	Action string

	// Input holds the input data associated with the action.
	// Callers MUST redact sensitive data such as PII, API keys, secrets, and
	// passwords before logging to prevent information disclosure.
	Input any

	// Output holds the output data produced by the action.
	// Callers MUST redact sensitive data such as PII, API keys, secrets, and
	// passwords before logging to prevent information disclosure.
	Output any

	// Metadata carries arbitrary key-value string annotations.
	Metadata map[string]string

	// Error holds a string description of any error that occurred during the
	// action. Empty when the action succeeded.
	Error string

	// Duration is the elapsed time of the action.
	Duration time.Duration
}

// Logger is the write-only interface for emitting audit entries.
// Implementations must be safe for concurrent use.
type Logger interface {
	// Log records a single audit entry. Implementations may enrich the entry
	// with a generated ID and current timestamp if those fields are empty.
	Log(ctx context.Context, entry Entry) error
}

// Store extends Logger with the ability to query historical entries.
// Implementations must be safe for concurrent use.
type Store interface {
	Logger

	// Query returns entries matching the given filter. An empty filter
	// returns all entries up to Filter.Limit (or all entries when Limit ≤ 0).
	Query(ctx context.Context, filter Filter) ([]Entry, error)
}

// Filter constrains a [Store.Query] call. All fields are optional; zero values
// are treated as "match any".
type Filter struct {
	// TenantID, when non-empty, restricts results to this tenant.
	TenantID string

	// AgentID, when non-empty, restricts results to this agent.
	AgentID string

	// SessionID, when non-empty, restricts results to this session.
	SessionID string

	// Action, when non-empty, restricts results to this action string.
	Action string

	// Since, when non-zero, excludes entries with Timestamp before this time.
	Since time.Time

	// Until, when non-zero, excludes entries with Timestamp after this time.
	Until time.Time

	// Limit caps the number of returned entries. Zero or negative means no cap.
	Limit int
}

// generateID returns a cryptographically random 16-byte hex string suitable
// for use as an entry ID.
func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// enrichEntry fills in the ID and Timestamp of e when those fields are zero,
// returning the enriched copy.
func enrichEntry(e Entry) (Entry, error) {
	if e.ID == "" {
		id, err := generateID()
		if err != nil {
			return Entry{}, err
		}
		e.ID = id
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	return e, nil
}
