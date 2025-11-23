package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVoiceSessionImpl(t *testing.T) {
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
	assert.NotNil(t, impl)
	assert.NotEmpty(t, impl.sessionID)
	assert.False(t, impl.active)
}

func TestNewVoiceSessionImpl_WithConfig(t *testing.T) {
	config := &Config{
		SessionID:         "test-session-123",
		Timeout:           30 * time.Minute,
		EnableKeepAlive:   true,
		KeepAliveInterval: 30 * time.Second,
		MaxRetries:        3,
		RetryDelay:        1 * time.Second,
	}

	opts := &VoiceOptions{
		STTProvider:  nil,
		TTSProvider:  nil,
		VADProvider:  nil,
		TurnDetector: nil,
		Transport:    nil,
		Config:       config,
	}

	impl, err := NewVoiceSessionImpl(config, opts)
	require.NoError(t, err)
	assert.NotNil(t, impl)
	assert.Equal(t, "test-session-123", impl.sessionID)
}

func TestNewVoiceSessionImpl_GenerateSessionID(t *testing.T) {
	opts := &VoiceOptions{
		STTProvider:  nil,
		TTSProvider:  nil,
		VADProvider:  nil,
		TurnDetector: nil,
		Transport:    nil,
		Config:       nil,
	}

	impl1, err := NewVoiceSessionImpl(nil, opts)
	require.NoError(t, err)

	impl2, err := NewVoiceSessionImpl(nil, opts)
	require.NoError(t, err)

	// Session IDs should be different
	assert.NotEqual(t, impl1.sessionID, impl2.sessionID)
}

func TestNewVoiceSessionImpl_WithDefaults(t *testing.T) {
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
	assert.NotNil(t, impl)
	assert.NotNil(t, impl.config)
	assert.Equal(t, 30*time.Minute, impl.config.Timeout)
	assert.True(t, impl.config.EnableKeepAlive)
	assert.Equal(t, 30*time.Second, impl.config.KeepAliveInterval)
	assert.Equal(t, 3, impl.config.MaxRetries)
	assert.Equal(t, 1*time.Second, impl.config.RetryDelay)
}

