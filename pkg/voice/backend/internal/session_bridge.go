package internal

import (
	"context"
	"fmt"
	"sync"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// SessionBridge maps provider-specific participants to Beluga AI voice sessions.
// This is a generic bridge that can be used by any provider.
type SessionBridge struct {
	participantToSession map[string]vbiface.VoiceSession
	sessionToParticipant map[string]string
	mu                   sync.RWMutex
}

// NewSessionBridge creates a new session bridge.
func NewSessionBridge() *SessionBridge {
	return &SessionBridge{
		participantToSession: make(map[string]vbiface.VoiceSession),
		sessionToParticipant: make(map[string]string),
	}
}

// CreateSessionForParticipant creates a voice session for a participant.
func (sb *SessionBridge) CreateSessionForParticipant(ctx context.Context, participantID string, config *vbiface.SessionConfig, voiceBackend vbiface.VoiceBackend) (vbiface.VoiceSession, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	// Check if session already exists
	if session, exists := sb.participantToSession[participantID]; exists {
		return session, nil
	}

	// Create new session
	session, err := voiceBackend.CreateSession(ctx, config)
	if err != nil {
		return nil, backend.WrapError("CreateSessionForParticipant", err)
	}

	// Map participant to session
	sb.participantToSession[participantID] = session
	sb.sessionToParticipant[session.GetID()] = participantID

	return session, nil
}

// GetSessionForParticipant retrieves the voice session for a participant.
func (sb *SessionBridge) GetSessionForParticipant(participantID string) (vbiface.VoiceSession, error) {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	session, exists := sb.participantToSession[participantID]
	if !exists {
		return nil, backend.NewBackendError("GetSessionForParticipant", backend.ErrCodeSessionNotFound,
			fmt.Errorf("no session found for participant '%s'", participantID))
	}

	return session, nil
}

// GetParticipantForSession retrieves the participant ID for a session.
func (sb *SessionBridge) GetParticipantForSession(sessionID string) (string, error) {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	participantID, exists := sb.sessionToParticipant[sessionID]
	if !exists {
		return "", backend.NewBackendError("GetParticipantForSession", backend.ErrCodeSessionNotFound,
			fmt.Errorf("no participant found for session '%s'", sessionID))
	}

	return participantID, nil
}

// RemoveParticipant removes the mapping for a participant.
func (sb *SessionBridge) RemoveParticipant(participantID string) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	if session, exists := sb.participantToSession[participantID]; exists {
		sessionID := session.GetID()
		delete(sb.participantToSession, participantID)
		delete(sb.sessionToParticipant, sessionID)
	}
}

// CloseSessionForParticipant closes the session for a participant.
func (sb *SessionBridge) CloseSessionForParticipant(ctx context.Context, participantID string, voiceBackend vbiface.VoiceBackend) error {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	session, exists := sb.participantToSession[participantID]
	if !exists {
		return nil // Session may have already been removed
	}

	sessionID := session.GetID()
	delete(sb.participantToSession, participantID)
	delete(sb.sessionToParticipant, sessionID)

	return voiceBackend.CloseSession(ctx, sessionID)
}
