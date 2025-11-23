package quickstart

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderSetup_Validation validates provider setup from quickstart
func TestProviderSetup_Validation(t *testing.T) {
	ctx := context.Background()

	// Test STT provider setup
	sttProvider := &mockSTTProvider{}
	assert.NotNil(t, sttProvider)

	// Test TTS provider setup
	ttsProvider := &mockTTSProvider{}
	assert.NotNil(t, ttsProvider)

	// Test session creation with providers
	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
	)
	require.NoError(t, err)
	assert.NotNil(t, voiceSession)

	// Verify session has providers configured
	sessionID := voiceSession.GetSessionID()
	assert.NotEmpty(t, sessionID)
}

// TestProviderConfiguration_Validation validates provider configuration
func TestProviderConfiguration_Validation(t *testing.T) {
	ctx := context.Background()

	// Test with default config
	config := session.DefaultConfig()
	assert.NotNil(t, config)

	sttProvider := &mockSTTProvider{}
	ttsProvider := &mockTTSProvider{}

	voiceSession, err := session.NewVoiceSession(ctx,
		session.WithSTTProvider(sttProvider),
		session.WithTTSProvider(ttsProvider),
		session.WithConfig(config),
	)
	require.NoError(t, err)
	assert.NotNil(t, voiceSession)
}
