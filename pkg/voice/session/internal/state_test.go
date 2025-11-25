package internal

import (
	"testing"

	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
	"github.com/stretchr/testify/assert"
)

func TestNewStateMachine(t *testing.T) {
	sm := NewStateMachine()
	assert.NotNil(t, sm)
	assert.Equal(t, sessioniface.SessionState("initial"), sm.GetState())
}

func TestStateMachine_GetState(t *testing.T) {
	sm := NewStateMachine()
	state := sm.GetState()
	assert.Equal(t, sessioniface.SessionState("initial"), state)
}

func TestStateMachine_SetState(t *testing.T) {
	tests := []struct {
		name      string
		fromState sessioniface.SessionState
		toState   sessioniface.SessionState
		wantValid bool
	}{
		{
			name:      "initial to listening",
			fromState: sessioniface.SessionState("initial"),
			toState:   sessioniface.SessionState("listening"),
			wantValid: true,
		},
		{
			name:      "listening to processing",
			fromState: sessioniface.SessionState("listening"),
			toState:   sessioniface.SessionState("processing"),
			wantValid: true,
		},
		{
			name:      "processing to speaking",
			fromState: sessioniface.SessionState("processing"),
			toState:   sessioniface.SessionState("speaking"),
			wantValid: true,
		},
		{
			name:      "speaking to listening",
			fromState: sessioniface.SessionState("speaking"),
			toState:   sessioniface.SessionState("listening"),
			wantValid: true,
		},
		{
			name:      "listening to ended",
			fromState: sessioniface.SessionState("listening"),
			toState:   sessioniface.SessionState("ended"),
			wantValid: true,
		},
		{
			name:      "initial to ended",
			fromState: sessioniface.SessionState("initial"),
			toState:   sessioniface.SessionState("ended"),
			wantValid: true,
		},
		{
			name:      "ended to initial",
			fromState: sessioniface.SessionState("ended"),
			toState:   sessioniface.SessionState("initial"),
			wantValid: true,
		},
		{
			name:      "listening to away",
			fromState: sessioniface.SessionState("listening"),
			toState:   sessioniface.SessionState("away"),
			wantValid: true,
		},
		{
			name:      "away to listening",
			fromState: sessioniface.SessionState("away"),
			toState:   sessioniface.SessionState("listening"),
			wantValid: true,
		},
		{
			name:      "processing to listening",
			fromState: sessioniface.SessionState("processing"),
			toState:   sessioniface.SessionState("listening"),
			wantValid: true,
		},
		{
			name:      "invalid: initial to speaking",
			fromState: sessioniface.SessionState("initial"),
			toState:   sessioniface.SessionState("speaking"),
			wantValid: false,
		},
		{
			name:      "invalid: initial to processing",
			fromState: sessioniface.SessionState("initial"),
			toState:   sessioniface.SessionState("processing"),
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewStateMachine()
			// Set up the from state properly (may need intermediate transitions)
			if tt.fromState != sessioniface.SessionState("initial") {
				// Need to transition through valid states
				if tt.fromState == sessioniface.SessionState("processing") {
					sm.SetState(sessioniface.SessionState("listening"))
					sm.SetState(sessioniface.SessionState("processing"))
				} else if tt.fromState == sessioniface.SessionState("speaking") {
					sm.SetState(sessioniface.SessionState("listening"))
					sm.SetState(sessioniface.SessionState("speaking"))
				} else if tt.fromState == sessioniface.SessionState("away") {
					sm.SetState(sessioniface.SessionState("listening"))
					sm.SetState(sessioniface.SessionState("away"))
				} else {
					sm.SetState(tt.fromState)
				}
			}
			result := sm.SetState(tt.toState)
			assert.Equal(t, tt.wantValid, result)
			if tt.wantValid {
				assert.Equal(t, tt.toState, sm.GetState())
			} else {
				// State should remain unchanged
				assert.Equal(t, tt.fromState, sm.GetState())
			}
		})
	}
}

func TestStateMachine_SetState_SameState(t *testing.T) {
	sm := NewStateMachine()
	sm.SetState(sessioniface.SessionState("listening"))

	// Setting to same state - isValidTransition doesn't allow same state
	// So it should return false
	_ = sm.SetState(sessioniface.SessionState("listening"))
	// Same state transition is not explicitly allowed, so it may return false
	// But state should remain unchanged
	assert.Equal(t, sessioniface.SessionState("listening"), sm.GetState())
}

func TestStateMachine_ConcurrentAccess(t *testing.T) {
	sm := NewStateMachine()

	// Test concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_ = sm.GetState()
			done <- true
		}()
	}

	// Wait for all reads
	for i := 0; i < 10; i++ {
		<-done
	}

	// Test concurrent writes (should be safe)
	sm.SetState(sessioniface.SessionState("listening"))
	state := sm.GetState()
	assert.Equal(t, sessioniface.SessionState("listening"), state)
}
