package internal

import (
	"sync"

	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
)

// StateMachine manages session state transitions
type StateMachine struct {
	mu    sync.RWMutex
	state sessioniface.SessionState
}

// NewStateMachine creates a new state machine
func NewStateMachine() *StateMachine {
	return &StateMachine{
		state: sessioniface.SessionState("initial"),
	}
}

// GetState returns the current state
func (sm *StateMachine) GetState() sessioniface.SessionState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

// SetState sets the state (with validation)
func (sm *StateMachine) SetState(newState sessioniface.SessionState) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validate state transition
	if !isValidTransition(sm.state, newState) {
		return false
	}

	sm.state = newState
	return true
}

// isValidTransition checks if a state transition is valid
func isValidTransition(from, to sessioniface.SessionState) bool {
	// Define valid transitions
	validTransitions := map[sessioniface.SessionState][]sessioniface.SessionState{
		sessioniface.SessionState("initial"): {
			sessioniface.SessionState("listening"),
			sessioniface.SessionState("ended"),
		},
		sessioniface.SessionState("listening"): {
			sessioniface.SessionState("processing"),
			sessioniface.SessionState("speaking"),
			sessioniface.SessionState("away"),
			sessioniface.SessionState("ended"),
		},
		sessioniface.SessionState("processing"): {
			sessioniface.SessionState("speaking"),
			sessioniface.SessionState("listening"),
			sessioniface.SessionState("ended"),
		},
		sessioniface.SessionState("speaking"): {
			sessioniface.SessionState("listening"),
			sessioniface.SessionState("processing"),
			sessioniface.SessionState("ended"),
		},
		sessioniface.SessionState("away"): {
			sessioniface.SessionState("listening"),
			sessioniface.SessionState("ended"),
		},
		sessioniface.SessionState("ended"): {
			// Allow restarting from ended state
			sessioniface.SessionState("initial"),
			sessioniface.SessionState("listening"),
		},
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowedState := range allowed {
		if allowedState == to {
			return true
		}
	}

	return false
}
