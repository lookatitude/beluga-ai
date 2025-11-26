package internal

import (
	"context"
	"sync"
	"testing"
	"time"

	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVoiceSessionImpl_OnStateChanged(t *testing.T) {
	opts := &VoiceOptions{
		STTProvider:  nil,
		TTSProvider:  nil,
		VADProvider:  nil,
		TurnDetector: nil,
		Transport:    nil,
		Config:       nil,
	}

	impl, err := NewVoiceSessionImpl(nil, opts)
	require.NoError(t, err)
	require.NotNil(t, impl)

	// Set callback
	called := false
	var receivedState sessioniface.SessionState
	impl.OnStateChanged(func(state sessioniface.SessionState) {
		called = true
		receivedState = state
	})

	// Change state through Start (which triggers callback)
	ctx := context.Background()
	err = impl.Start(ctx)
	require.NoError(t, err)

	// Callback should have been called
	assert.True(t, called)
	assert.Equal(t, sessioniface.SessionState("listening"), receivedState)
}

func TestVoiceSessionImpl_OnStateChanged_MultipleChanges(t *testing.T) {
	opts := &VoiceOptions{
		STTProvider:  nil,
		TTSProvider:  nil,
		VADProvider:  nil,
		TurnDetector: nil,
		Transport:    nil,
		Config:       nil,
	}

	impl, err := NewVoiceSessionImpl(nil, opts)
	require.NoError(t, err)
	require.NotNil(t, impl)

	// Track all state changes
	var mu sync.Mutex
	states := make([]sessioniface.SessionState, 0)
	impl.OnStateChanged(func(state sessioniface.SessionState) {
		mu.Lock()
		states = append(states, state)
		mu.Unlock()
	})

	// Change states through actual methods (which trigger callbacks)
	ctx := context.Background()
	_ = impl.Start(ctx) // Should trigger callback for "listening"

	// Use Say to trigger speaking state
	_, _ = impl.Say(ctx, "test") // Should trigger callback for "speaking"

	// Wait a bit for async state change
	time.Sleep(200 * time.Millisecond)

	// Should have received all state changes
	mu.Lock()
	stateCount := len(states)
	mu.Unlock()
	assert.GreaterOrEqual(t, stateCount, 3)
}

func TestVoiceSessionImpl_OnStateChanged_NilCallback(t *testing.T) {
	opts := &VoiceOptions{
		STTProvider:  nil,
		TTSProvider:  nil,
		VADProvider:  nil,
		TurnDetector: nil,
		Transport:    nil,
		Config:       nil,
	}

	impl, err := NewVoiceSessionImpl(nil, opts)
	require.NoError(t, err)
	require.NotNil(t, impl)

	// Set nil callback (should not panic)
	impl.OnStateChanged(nil)

	// Change state through Start (should not panic)
	ctx := context.Background()
	err = impl.Start(ctx)
	require.NoError(t, err)
}

func TestVoiceSessionImpl_OnStateChanged_ReplaceCallback(t *testing.T) {
	opts := &VoiceOptions{
		STTProvider:  nil,
		TTSProvider:  nil,
		VADProvider:  nil,
		TurnDetector: nil,
		Transport:    nil,
		Config:       nil,
	}

	impl, err := NewVoiceSessionImpl(nil, opts)
	require.NoError(t, err)
	require.NotNil(t, impl)

	// Set first callback
	firstCalled := false
	impl.OnStateChanged(func(state sessioniface.SessionState) {
		firstCalled = true
	})

	// Replace with second callback
	secondCalled := false
	impl.OnStateChanged(func(state sessioniface.SessionState) {
		secondCalled = true
	})

	// Change state through Start
	ctx := context.Background()
	err = impl.Start(ctx)
	require.NoError(t, err)

	// Only second callback should be called
	assert.False(t, firstCalled)
	assert.True(t, secondCalled)
}
