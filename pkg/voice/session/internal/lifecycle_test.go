package internal

import (
	"context"
	"testing"
	"time"

	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVoiceSessionImpl_Start(t *testing.T) {
	impl := createTestSessionImpl(t)

	ctx := context.Background()
	err := impl.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, impl.active)
	assert.Equal(t, sessioniface.SessionState("listening"), impl.GetState())
}

func TestVoiceSessionImpl_Start_AlreadyActive(t *testing.T) {
	impl := createTestSessionImpl(t)

	ctx := context.Background()
	err := impl.Start(ctx)
	require.NoError(t, err)

	// Try to start again
	err = impl.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already_active")
}

func TestVoiceSessionImpl_Start_FromEndedState(t *testing.T) {
	impl := createTestSessionImpl(t)

	ctx := context.Background()
	
	// Start and stop first
	err := impl.Start(ctx)
	require.NoError(t, err)
	err = impl.Stop(ctx)
	require.NoError(t, err)
	assert.Equal(t, sessioniface.SessionState("ended"), impl.GetState())

	// Start again from ended state
	err = impl.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, impl.active)
	assert.Equal(t, sessioniface.SessionState("listening"), impl.GetState())
}

func TestVoiceSessionImpl_Stop(t *testing.T) {
	impl := createTestSessionImpl(t)

	ctx := context.Background()
	err := impl.Start(ctx)
	require.NoError(t, err)

	err = impl.Stop(ctx)
	assert.NoError(t, err)
	assert.False(t, impl.active)
	assert.Equal(t, sessioniface.SessionState("ended"), impl.GetState())
}

func TestVoiceSessionImpl_Stop_NotActive(t *testing.T) {
	impl := createTestSessionImpl(t)

	ctx := context.Background()
	err := impl.Stop(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not_active")
}

func TestVoiceSessionImpl_GetSessionID(t *testing.T) {
	impl := createTestSessionImpl(t)
	sessionID := impl.GetSessionID()
	assert.NotEmpty(t, sessionID)
}

func TestVoiceSessionImpl_GetState(t *testing.T) {
	impl := createTestSessionImpl(t)
	state := impl.GetState()
	assert.Equal(t, sessioniface.SessionState("initial"), state)
}

// Helper function to create a test session implementation
func createTestSessionImpl(t *testing.T) *VoiceSessionImpl {
	config := &Config{
		SessionID: "test-session-123",
		Timeout:   30 * time.Minute,
	}
	
	opts := &VoiceOptions{
		STTProvider: nil, // Can be nil for basic tests
		TTSProvider: nil,
	}

	impl, err := NewVoiceSessionImpl(config, opts)
	require.NoError(t, err)
	require.NotNil(t, impl)
	
	return impl
}

