// Package audit provides the audit logging abstractions for recording agent
// turn lifecycle events. Implementations of Store persist audit entries to
// a backend such as a database, log stream, or cloud audit service.
package audit

import (
	"context"
	"time"
)

// Entry represents a single auditable event emitted during agent execution.
type Entry struct {
	// Action identifies the lifecycle phase (e.g., "agent.turn.start").
	Action string
	// AgentID identifies the agent that emitted this entry.
	AgentID string
	// SessionID identifies the session in which the event occurred.
	SessionID string
	// Timestamp is when the event occurred.
	Timestamp time.Time
	// Metadata holds arbitrary structured data associated with the event.
	Metadata map[string]any
}

// Store is the interface for persisting audit entries. Implementations must
// be safe for concurrent use by multiple goroutines.
type Store interface {
	// Log persists the given audit entry. Implementations must return an error
	// if the entry cannot be stored; they must not silently discard entries.
	Log(ctx context.Context, entry Entry) error
}

// Logger wraps a Store with convenience methods for the standard agent turn
// lifecycle actions.
type Logger struct {
	store Store
}

// NewLogger returns a Logger backed by the given Store.
func NewLogger(store Store) *Logger {
	return &Logger{store: store}
}

// Log writes the entry directly to the underlying store.
func (l *Logger) Log(ctx context.Context, entry Entry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	return l.store.Log(ctx, entry)
}

// TurnStart logs an "agent.turn.start" entry.
func (l *Logger) TurnStart(ctx context.Context, agentID, sessionID string, meta map[string]any) error {
	return l.Log(ctx, Entry{
		Action:    "agent.turn.start",
		AgentID:   agentID,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Metadata:  meta,
	})
}

// TurnEnd logs an "agent.turn.end" entry.
func (l *Logger) TurnEnd(ctx context.Context, agentID, sessionID string, meta map[string]any) error {
	return l.Log(ctx, Entry{
		Action:    "agent.turn.end",
		AgentID:   agentID,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Metadata:  meta,
	})
}

// TurnError logs an "agent.turn.error" entry.
func (l *Logger) TurnError(ctx context.Context, agentID, sessionID string, err error) error {
	meta := map[string]any{"error": err.Error()}
	return l.Log(ctx, Entry{
		Action:    "agent.turn.error",
		AgentID:   agentID,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Metadata:  meta,
	})
}
