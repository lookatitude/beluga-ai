package runtime

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/schema"
)

// Session holds the full state for a single agent conversation, including
// the ordered turn history, arbitrary key-value state, and lifecycle
// timestamps.
type Session struct {
	// ID is the unique, crypto-random identifier for this session.
	ID string

	// AgentID is the identifier of the agent that owns this session.
	AgentID string

	// TenantID scopes the session to a specific tenant for multi-tenancy.
	TenantID string

	// State holds arbitrary session-level data accessible across turns.
	State map[string]any

	// Turns contains the ordered sequence of conversation turns.
	Turns []schema.Turn

	// CreatedAt is the timestamp when the session was first created.
	CreatedAt time.Time

	// UpdatedAt is the timestamp when the session was last modified.
	UpdatedAt time.Time

	// ExpiresAt is the timestamp after which the session may be evicted.
	// A zero value means the session does not expire.
	ExpiresAt time.Time
}

// NewSession creates a Session with the given ID and agent ID, initializing
// timestamps and an empty state map.
func NewSession(id, agentID string) *Session {
	now := time.Now()
	return &Session{
		ID:        id,
		AgentID:   agentID,
		State:     make(map[string]any),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SessionService manages session lifecycle: creation, retrieval, mutation,
// and deletion. Implementations must be safe for concurrent use.
type SessionService interface {
	// Create allocates a new Session for the given agentID and persists it.
	// It returns the newly created session or an error.
	Create(ctx context.Context, agentID string) (*Session, error)

	// Get retrieves the Session identified by sessionID. It returns a
	// core.Error with code ErrNotFound if no such session exists.
	Get(ctx context.Context, sessionID string) (*Session, error)

	// Update persists the current state of an existing session. It returns a
	// core.Error with code ErrNotFound if the session does not exist.
	Update(ctx context.Context, session *Session) error

	// Delete removes the session identified by sessionID. It returns a
	// core.Error with code ErrNotFound if no such session exists.
	Delete(ctx context.Context, sessionID string) error
}
