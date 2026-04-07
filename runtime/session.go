package runtime

import (
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/schema"
)

// Session holds runtime state for a single agent conversation.
// It carries the conversation turns and a thread-safe state map that
// plugins and the runner can read and write across turns.
type Session struct {
	mu sync.RWMutex

	// ID is the unique identifier for this session.
	ID string

	// AgentID identifies the agent that owns this session.
	AgentID string

	// Turns contains the ordered sequence of conversation turns recorded so far.
	Turns []schema.Turn

	// State holds arbitrary session-level state accessible across turns.
	State map[string]any

	// CreatedAt is when the session was created.
	CreatedAt time.Time

	// UpdatedAt is when the session was last modified.
	UpdatedAt time.Time
}

// NewSession constructs a new Session with the given ID and agent ID.
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

// Get retrieves a value from the session state. It is safe for concurrent use.
func (s *Session) Get(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.State[key]
	return v, ok
}

// Set writes a value into the session state. It is safe for concurrent use.
func (s *Session) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.State[key] = value
	s.UpdatedAt = time.Now()
}

// AddTurn appends a completed turn to the session history. It is safe for concurrent use.
func (s *Session) AddTurn(turn schema.Turn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Turns = append(s.Turns, turn)
	s.UpdatedAt = time.Now()
}
