package internal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// SessionManager manages voice session lifecycle and state.
// It ensures thread-safe access and session isolation (T169, T171).
type SessionManager struct {
	store    PersistenceStore
	sessions map[string]vbiface.VoiceSession
	config   *vbiface.Config
	mu       sync.RWMutex
}

// NewSessionManager creates a new session manager.
func NewSessionManager(config *vbiface.Config) *SessionManager {
	// Use in-memory persistence store by default
	// In production, this can be replaced with a database-backed store
	store := NewInMemoryPersistenceStore()
	return &SessionManager{
		sessions: make(map[string]vbiface.VoiceSession),
		config:   config,
		store:    store,
	}
}

// SetPersistenceStore sets the persistence store for session persistence (T305, T306).
func (sm *SessionManager) SetPersistenceStore(store PersistenceStore) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.store = store
}

// CreateSession creates a new voice session with session ID generation and validation.
func (sm *SessionManager) CreateSession(ctx context.Context, sessionConfig *vbiface.SessionConfig) (vbiface.VoiceSession, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check session limit
	if sm.config.MaxConcurrentSessions > 0 {
		if len(sm.sessions) >= sm.config.MaxConcurrentSessions {
			return nil, backend.NewBackendError("CreateSession", backend.ErrCodeSessionLimitExceeded,
				fmt.Errorf("maximum concurrent sessions (%d) exceeded", sm.config.MaxConcurrentSessions))
		}
	}

	// Validate session config
	if err := backend.ValidateSessionConfig(sessionConfig); err != nil {
		return nil, backend.NewBackendError("CreateSession", backend.ErrCodeInvalidConfig, err)
	}

	// Create session (will be implemented by providers)
	// For now, return an error indicating this needs provider implementation
	// The actual session creation will be done by the provider's CreateSession method
	return nil, errors.New("session creation must be implemented by provider")
}

// GetSession retrieves a session by ID with thread-safe access.
func (sm *SessionManager) GetSession(sessionID string) (vbiface.VoiceSession, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, backend.NewBackendError("GetSession", backend.ErrCodeSessionNotFound,
			fmt.Errorf("session '%s' not found", sessionID))
	}

	return session, nil
}

// ListSessions returns all active sessions.
func (sm *SessionManager) ListSessions() []vbiface.VoiceSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]vbiface.VoiceSession, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// CloseSession closes a session with cleanup and state transition to Completed.
func (sm *SessionManager) CloseSession(ctx context.Context, sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return backend.NewBackendError("CloseSession", backend.ErrCodeSessionNotFound,
			fmt.Errorf("session '%s' not found", sessionID))
	}

	// Stop the session
	if err := session.Stop(ctx); err != nil {
		return backend.WrapError("CloseSession", err)
	}

	// Remove from active sessions
	delete(sm.sessions, sessionID)

	return nil
}

// AddSession adds a session to the manager (called by providers after creating session).
func (sm *SessionManager) AddSession(sessionID string, session vbiface.VoiceSession) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check session limit
	if sm.config.MaxConcurrentSessions > 0 {
		if len(sm.sessions) >= sm.config.MaxConcurrentSessions {
			return backend.NewBackendError("AddSession", backend.ErrCodeSessionLimitExceeded,
				fmt.Errorf("maximum concurrent sessions (%d) exceeded", sm.config.MaxConcurrentSessions))
		}
	}

	sm.sessions[sessionID] = session
	return nil
}

// RemoveSession removes a session from the manager.
func (sm *SessionManager) RemoveSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, sessionID)
}

// GetActiveSessionCount returns the number of active sessions.
func (sm *SessionManager) GetActiveSessionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// PersistActiveSessions persists active sessions (T305, FR-025).
// Active sessions should persist to survive restarts.
func (sm *SessionManager) PersistActiveSessions(ctx context.Context) error {
	sm.mu.RLock()
	sessions := make([]vbiface.VoiceSession, 0, len(sm.sessions))
	store := sm.store
	for _, session := range sm.sessions {
		if session.GetPersistenceStatus() == vbiface.PersistenceStatusActive {
			sessions = append(sessions, session)
		}
	}
	sm.mu.RUnlock()

	if store == nil {
		// No persistence store configured, skip persistence
		return nil
	}

	// Persist each active session
	for _, session := range sessions {
		if err := PersistSession(ctx, store, session); err != nil {
			backend.LogWithOTELContext(ctx, slog.LevelError, "Failed to persist session",
				"session_id", session.GetID(), "error", err)
			// Continue with other sessions even if one fails
		}
	}

	return nil
}

// RecoverActiveSessions recovers active sessions after restart (T306, FR-025).
func (sm *SessionManager) RecoverActiveSessions(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.store == nil {
		// No persistence store configured, nothing to recover
		return nil
	}

	// List all active sessions from persistence store
	sessionIDs, err := sm.store.ListActiveSessions(ctx)
	if err != nil {
		return backend.WrapError("RecoverActiveSessions", err)
	}

	// Load and restore each session
	// Note: In a full implementation, this would recreate the actual session objects
	// from the persisted state. For now, we just load the state.
	// The actual session recreation would need to be done by the backend provider
	// since it knows how to create sessions.
	for _, sessionID := range sessionIDs {
		state, err := sm.store.LoadSession(ctx, sessionID)
		if err != nil {
			backend.LogWithOTELContext(ctx, slog.LevelWarn, "Failed to load session state",
				"session_id", sessionID, "error", err)
			continue
		}

		// In a full implementation, we would:
		// 1. Recreate the session object from state
		// 2. Restore session connections
		// 3. Add session back to session manager
		_ = state // For now, just load the state
	}

	return nil
}

// CleanupCompletedSessions cleans up completed sessions (T307, FR-026).
// Completed sessions are ephemeral and should be cleaned up.
func (sm *SessionManager) CleanupCompletedSessions(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sessionsToRemove := []string{}
	cleanupThreshold := 5 * time.Minute // Cleanup threshold for completed sessions

	for sessionID, session := range sm.sessions {
		if session.GetPersistenceStatus() == vbiface.PersistenceStatusCompleted {
			// In a full implementation, we would check the completion time
			// For now, remove completed sessions immediately
			sessionsToRemove = append(sessionsToRemove, sessionID)
		}
	}

	// Remove completed sessions from memory
	for _, sessionID := range sessionsToRemove {
		delete(sm.sessions, sessionID)
	}

	// Also remove from persistence store if configured
	if sm.store != nil {
		for _, sessionID := range sessionsToRemove {
			if err := sm.store.DeleteSession(ctx, sessionID); err != nil {
				backend.LogWithOTELContext(ctx, slog.LevelWarn, "Failed to delete session from persistence store",
					"session_id", sessionID, "error", err)
			}
		}
	}

	_ = cleanupThreshold // Will be used in full implementation

	return nil
}
