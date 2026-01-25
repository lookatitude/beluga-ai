package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// SessionState represents the serializable state of a voice session (T305, T306).
type SessionState struct {
	CreatedAt         time.Time                 `json:"created_at"`
	LastActivity      time.Time                 `json:"last_activity"`
	Metadata          map[string]any            `json:"metadata"`
	Config            *SessionConfigState       `json:"config"`
	ID                string                    `json:"id"`
	UserID            string                    `json:"user_id"`
	State             vbiface.PipelineState     `json:"state"`
	PersistenceStatus vbiface.PersistenceStatus `json:"persistence_status"`
}

// SessionConfigState represents the serializable session configuration.
type SessionConfigState struct {
	UserID       string `json:"user_id"`
	Transport    string `json:"transport"`
	PipelineType string `json:"pipeline_type"`
	// Note: AgentCallback and AgentInstance are not serializable
	// They will need to be restored from the backend configuration
}

// PersistenceStore defines the interface for session persistence storage (T305, T306).
type PersistenceStore interface {
	// SaveSession saves a session state to persistent storage.
	SaveSession(ctx context.Context, state *SessionState) error

	// LoadSession loads a session state from persistent storage.
	LoadSession(ctx context.Context, sessionID string) (*SessionState, error)

	// ListActiveSessions returns all active session IDs.
	ListActiveSessions(ctx context.Context) ([]string, error)

	// DeleteSession deletes a session from persistent storage.
	DeleteSession(ctx context.Context, sessionID string) error
}

// InMemoryPersistenceStore is an in-memory implementation of PersistenceStore (T305, T306).
// This is suitable for development and testing. Production deployments should use
// a persistent storage backend (database, file system, etc.).
type InMemoryPersistenceStore struct {
	sessions map[string]*SessionState
	mu       sync.RWMutex
}

// NewInMemoryPersistenceStore creates a new in-memory persistence store.
func NewInMemoryPersistenceStore() *InMemoryPersistenceStore {
	return &InMemoryPersistenceStore{
		sessions: make(map[string]*SessionState),
	}
}

// SaveSession saves a session state to in-memory storage.
func (s *InMemoryPersistenceStore) SaveSession(ctx context.Context, state *SessionState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state.LastActivity = time.Now()
	s.sessions[state.ID] = state
	return nil
}

// LoadSession loads a session state from in-memory storage.
func (s *InMemoryPersistenceStore) LoadSession(ctx context.Context, sessionID string) (*SessionState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, exists := s.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	// Return a copy to prevent external modifications
	stateCopy := *state
	return &stateCopy, nil
}

// ListActiveSessions returns all active session IDs.
func (s *InMemoryPersistenceStore) ListActiveSessions(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sessionIDs []string
	for id, state := range s.sessions {
		if state.PersistenceStatus == vbiface.PersistenceStatusActive {
			sessionIDs = append(sessionIDs, id)
		}
	}
	return sessionIDs, nil
}

// DeleteSession deletes a session from in-memory storage.
func (s *InMemoryPersistenceStore) DeleteSession(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, sessionID)
	return nil
}

// serializeSessionState serializes a voice session to SessionState (T305).
func serializeSessionState(session vbiface.VoiceSession) (*SessionState, error) {
	sessionID := session.GetID()
	state := session.GetState()
	persistenceStatus := session.GetPersistenceStatus()

	// Extract metadata - in a full implementation, sessions would expose GetMetadata()
	// For now, use empty metadata as sessions don't expose metadata retrieval
	metadata := make(map[string]any)

	// Create session state
	sessionState := &SessionState{
		ID:                sessionID,
		State:             state,
		PersistenceStatus: persistenceStatus,
		Metadata:          metadata,
		CreatedAt:         time.Now(), // In a full implementation, this would be stored
		LastActivity:      time.Now(),
	}

	return sessionState, nil
}

// PersistSession persists a single session to storage (T305).
func PersistSession(ctx context.Context, store PersistenceStore, session vbiface.VoiceSession) error {
	if session.GetPersistenceStatus() != vbiface.PersistenceStatusActive {
		// Only persist active sessions
		return nil
	}

	state, err := serializeSessionState(session)
	if err != nil {
		return backend.WrapError("PersistSession", err)
	}

	return store.SaveSession(ctx, state)
}
