package internal

import (
	"context"
	"testing"
	"time"

	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVoiceSessionImpl_Say(t *testing.T) {
	impl := createTestSessionImpl(t)

	ctx := context.Background()
	err := impl.Start(ctx)
	require.NoError(t, err)

	handle, err := impl.Say(ctx, "Hello, world!")
	require.NoError(t, err)
	assert.NotNil(t, handle)

	// Should transition to speaking state
	assert.Equal(t, sessioniface.SessionState("speaking"), impl.GetState())
}

func TestVoiceSessionImpl_Say_NotActive(t *testing.T) {
	impl := createTestSessionImpl(t)

	ctx := context.Background()
	_, err := impl.Say(ctx, "Hello, world!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not_active")
}

func TestVoiceSessionImpl_SayWithOptions(t *testing.T) {
	impl := createTestSessionImpl(t)

	ctx := context.Background()
	err := impl.Start(ctx)
	require.NoError(t, err)

	options := sessioniface.SayOptions{
		AllowInterruptions: true,
		Voice:              "alloy",
		Speed:              1.0,
	}

	handle, err := impl.SayWithOptions(ctx, "Hello, world!", options)
	require.NoError(t, err)
	assert.NotNil(t, handle)
}

func TestVoiceSessionImpl_SayWithOptions_InvalidState(t *testing.T) {
	impl := createTestSessionImpl(t)

	ctx := context.Background()
	err := impl.Start(ctx)
	require.NoError(t, err)

	// Force invalid state by setting state machine to ended state
	impl.mu.Lock()
	impl.stateMachine.SetState(sessioniface.SessionState("ended"))
	impl.state = sessioniface.SessionState("ended")
	impl.mu.Unlock()

	// Try to say from ended state (should fail - ended to speaking is invalid)
	_, err = impl.SayWithOptions(ctx, "Hello", sessioniface.SayOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid_state")
}

func TestSayHandleImpl_WaitForPlayout(t *testing.T) {
	handle := &SayHandleImpl{}

	ctx := context.Background()
	err := handle.WaitForPlayout(ctx)
	require.NoError(t, err)
}

func TestSayHandleImpl_WaitForPlayout_ContextCancellation(t *testing.T) {
	handle := &SayHandleImpl{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := handle.WaitForPlayout(ctx)
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestSayHandleImpl_Cancel(t *testing.T) {
	handle := &SayHandleImpl{}

	err := handle.Cancel()
	require.NoError(t, err)
	assert.True(t, handle.canceled)
}

func TestSayHandleImpl_Cancel_MultipleTimes(t *testing.T) {
	handle := &SayHandleImpl{}

	err := handle.Cancel()
	require.NoError(t, err)

	// Cancel again should not error
	err = handle.Cancel()
	require.NoError(t, err)
	assert.True(t, handle.canceled)
}

func TestVoiceSessionImpl_Say_StateTransition(t *testing.T) {
	impl := createTestSessionImpl(t)

	ctx := context.Background()
	err := impl.Start(ctx)
	require.NoError(t, err)

	// Should be in listening state
	assert.Equal(t, sessioniface.SessionState("listening"), impl.GetState())

	// Say should transition to speaking
	handle, err := impl.Say(ctx, "Test")
	require.NoError(t, err)
	assert.Equal(t, sessioniface.SessionState("speaking"), impl.GetState())

	// Wait for transition back to listening
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, sessioniface.SessionState("listening"), impl.GetState())

	_ = handle
}
