package session

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
	"github.com/stretchr/testify/assert"
)

func TestSession_Start(t *testing.T) {
	tests := []struct {
		name          string
		session       iface.VoiceSession
		expectedError bool
	}{
		{
			name: "successful start",
			session: NewAdvancedMockSession("test-session",
				WithActive(false),
			),
			expectedError: false,
		},
		{
			name: "error on start",
			session: NewAdvancedMockSession("test-session",
				WithError(NewSessionError("Start", ErrCodeInternalError, nil)),
			),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := tt.session.Start(ctx)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				state := tt.session.GetState()
				assert.NotEqual(t, "ended", string(state))
			}
		})
	}
}

func TestSession_Stop(t *testing.T) {
	tests := []struct {
		name          string
		session       iface.VoiceSession
		expectedError bool
	}{
		{
			name: "successful stop",
			session: NewAdvancedMockSession("test-session",
				WithActive(true),
			),
			expectedError: false,
		},
		{
			name: "error on stop",
			session: NewAdvancedMockSession("test-session",
				WithError(NewSessionError("Stop", ErrCodeInternalError, nil)),
			),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := tt.session.Stop(ctx)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				state := tt.session.GetState()
				assert.Equal(t, "ended", string(state))
			}
		})
	}
}

func TestSession_GetSessionID(t *testing.T) {
	session := NewAdvancedMockSession("test-session-123")
	assert.Equal(t, "test-session-123", session.GetSessionID())
}

func TestSession_GetState(t *testing.T) {
	tests := []struct {
		name          string
		active        bool
		expectedState string
	}{
		{
			name:          "active session",
			active:        true,
			expectedState: "listening",
		},
		{
			name:          "inactive session",
			active:        false,
			expectedState: "initial",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := NewAdvancedMockSession("test-session", WithActive(tt.active))
			assert.Equal(t, tt.expectedState, string(session.GetState()))
		})
	}
}

func TestSession_InterfaceCompliance(t *testing.T) {
	session := NewAdvancedMockSession("test-session")
	AssertSessionInterface(t, session)
}
