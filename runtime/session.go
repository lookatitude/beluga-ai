package runtime

import (
	"time"
)

// Session holds runtime context for a single agent execution session.
// It carries the agent identity, session identifier, turn counter, and
// arbitrary state that persists across plugin calls within a session.
type Session struct {
	// AgentID is the unique identifier of the agent processing this session.
	AgentID string
	// SessionID is the unique identifier for this conversation session.
	SessionID string
	// TurnCount is the number of turns that have been processed so far.
	TurnCount int
	// StartedAt is the time the session was created.
	StartedAt time.Time
	// State holds arbitrary key-value state accessible to all plugins.
	State map[string]any
}

// NewSession creates a new Session with the given agent and session IDs.
func NewSession(agentID, sessionID string) *Session {
	return &Session{
		AgentID:   agentID,
		SessionID: sessionID,
		StartedAt: time.Now(),
		State:     make(map[string]any),
	}
}
