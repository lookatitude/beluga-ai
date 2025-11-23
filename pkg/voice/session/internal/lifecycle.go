package internal

import (
	"context"
	"time"

	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
)

// Start starts the voice session
func (s *VoiceSessionImpl) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		return newSessionError("Start", "session_already_active",
			nil)
	}

	startTime := time.Now()

	// If state is "ended", reset to "initial" first to allow restart
	currentState := s.stateMachine.GetState()
	if currentState == sessioniface.SessionState("ended") {
		s.stateMachine.SetState(sessioniface.SessionState("initial"))
		s.state = sessioniface.SessionState("initial")
	}

	// Transition to listening state
	if !s.stateMachine.SetState(sessioniface.SessionState("listening")) {
		return newSessionError("Start", "invalid_state",
			nil)
	}

	s.state = s.stateMachine.GetState()
	s.active = true

	// Notify state change
	if s.stateChangeCallback != nil {
		s.stateChangeCallback(s.state)
	}

	// Record metrics (metrics would be passed in or accessed differently to avoid import cycle)
	_ = startTime

	return nil
}

// Stop stops the voice session gracefully
func (s *VoiceSessionImpl) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return newSessionError("Stop", "session_not_active",
			nil)
	}

	stopTime := time.Now()

	// Transition to ended state
	s.stateMachine.SetState(sessioniface.SessionState("ended"))
	s.state = s.stateMachine.GetState()
	s.active = false

	// Notify state change
	if s.stateChangeCallback != nil {
		s.stateChangeCallback(s.state)
	}

	// Record metrics (metrics would be passed in or accessed differently to avoid import cycle)
	_ = stopTime

	return nil
}

// GetSessionID returns the session identifier
func (s *VoiceSessionImpl) GetSessionID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessionID
}

// GetState returns the current session state
func (s *VoiceSessionImpl) GetState() sessioniface.SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}
