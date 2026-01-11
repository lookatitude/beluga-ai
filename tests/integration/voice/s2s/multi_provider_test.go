package s2s

import (
	"context"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	s2siface "github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiProvider tests multi-provider configuration and fallback scenarios.
// This test validates User Story 2: Multi-Provider Support and Fallback (P2)
func TestMultiProvider(t *testing.T) {
	ctx := context.Background()

	// Create multiple mock providers
	primaryProvider := s2s.NewAdvancedMockS2SProvider("primary",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "primary", 100*time.Millisecond)))

	fallbackProvider1 := s2s.NewAdvancedMockS2SProvider("fallback1",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{4, 5, 6}, "fallback1", 150*time.Millisecond)))

	fallbackProvider2 := s2s.NewAdvancedMockS2SProvider("fallback2",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{7, 8, 9}, "fallback2", 200*time.Millisecond)))

	t.Run("create provider manager with fallbacks", func(t *testing.T) {
		fallbacks := []s2siface.S2SProvider{fallbackProvider1, fallbackProvider2}
		manager, err := s2s.NewProviderManager(primaryProvider, fallbacks)
		require.NoError(t, err)
		assert.NotNil(t, manager)
		assert.Equal(t, primaryProvider, manager.GetPrimaryProvider())
		assert.Equal(t, fallbacks, manager.GetFallbackProviders())
	})

	t.Run("provider manager uses primary provider", func(t *testing.T) {
		fallbacks := []s2siface.S2SProvider{fallbackProvider1, fallbackProvider2}
		manager, err := s2s.NewProviderManager(primaryProvider, fallbacks)
		require.NoError(t, err)

		input := s2s.NewAudioInput([]byte{1, 2, 3}, "test-session")
		convCtx := s2s.NewConversationContext("test-session")

		output, err := manager.Process(ctx, input, convCtx)
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.False(t, manager.IsUsingFallback())
		assert.Equal(t, "primary", manager.GetCurrentProviderName())
	})

	t.Run("provider manager falls back on primary failure", func(t *testing.T) {
		// Create primary that fails
		failingPrimary := s2s.NewAdvancedMockS2SProvider("primary",
			s2s.WithError(assert.AnError))

		fallbacks := []s2siface.S2SProvider{fallbackProvider1, fallbackProvider2}
		manager, err := s2s.NewProviderManager(failingPrimary, fallbacks)
		require.NoError(t, err)

		input := s2s.NewAudioInput([]byte{1, 2, 3}, "test-session")
		convCtx := s2s.NewConversationContext("test-session")

		output, err := manager.Process(ctx, input, convCtx)
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.True(t, manager.IsUsingFallback())
		assert.Equal(t, "fallback1", manager.GetCurrentProviderName())
	})

	t.Run("provider manager falls back through multiple providers", func(t *testing.T) {
		// Create providers that fail in sequence
		failingPrimary := s2s.NewAdvancedMockS2SProvider("primary",
			s2s.WithError(assert.AnError))
		failingFallback1 := s2s.NewAdvancedMockS2SProvider("fallback1",
			s2s.WithError(assert.AnError))

		fallbacks := []s2siface.S2SProvider{failingFallback1, fallbackProvider2}
		manager, err := s2s.NewProviderManager(failingPrimary, fallbacks)
		require.NoError(t, err)

		input := s2s.NewAudioInput([]byte{1, 2, 3}, "test-session")
		convCtx := s2s.NewConversationContext("test-session")

		output, err := manager.Process(ctx, input, convCtx)
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.True(t, manager.IsUsingFallback())
		assert.Equal(t, "fallback2", manager.GetCurrentProviderName())
	})

	t.Run("provider manager fails when all providers fail", func(t *testing.T) {
		// Create all providers that fail
		failingPrimary := s2s.NewAdvancedMockS2SProvider("primary",
			s2s.WithError(assert.AnError))
		failingFallback1 := s2s.NewAdvancedMockS2SProvider("fallback1",
			s2s.WithError(assert.AnError))
		failingFallback2 := s2s.NewAdvancedMockS2SProvider("fallback2",
			s2s.WithError(assert.AnError))

		fallbacks := []s2siface.S2SProvider{failingFallback1, failingFallback2}
		manager, err := s2s.NewProviderManager(failingPrimary, fallbacks)
		require.NoError(t, err)

		input := s2s.NewAudioInput([]byte{1, 2, 3}, "test-session")
		convCtx := s2s.NewConversationContext("test-session")

		output, err := manager.Process(ctx, input, convCtx)
		require.Error(t, err)
		assert.Nil(t, output)
		assert.Contains(t, err.Error(), "all S2S providers failed")
	})
}

// TestMultiProvider_SessionIntegration tests multi-provider fallback in session context.
func TestMultiProvider_SessionIntegration(t *testing.T) {
	ctx := context.Background()

	// Create providers
	primaryProvider := s2s.NewAdvancedMockS2SProvider("primary",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{1, 2, 3}, "primary", 100*time.Millisecond)))

	fallbackProvider := s2s.NewAdvancedMockS2SProvider("fallback1",
		s2s.WithAudioOutputs(s2s.NewAudioOutput([]byte{4, 5, 6}, "fallback1", 150*time.Millisecond)))

	t.Run("session with primary provider", func(t *testing.T) {
		voiceSession, err := session.NewVoiceSession(ctx,
			session.WithS2SProvider(primaryProvider),
		)
		require.NoError(t, err)
		assert.NotNil(t, voiceSession)

		err = voiceSession.Start(ctx)
		require.NoError(t, err)

		audio := []byte{1, 2, 3, 4, 5}
		err = voiceSession.ProcessAudio(ctx, audio)
		require.NoError(t, err)

		err = voiceSession.Stop(ctx)
		require.NoError(t, err)
	})

	t.Run("session with provider manager fallback", func(t *testing.T) {
		// Create provider manager with fallback
		fallbacks := []s2siface.S2SProvider{fallbackProvider}
		manager, err := s2s.NewProviderManager(primaryProvider, fallbacks)
		require.NoError(t, err)

		// Note: Session integration with provider manager would require
		// updating session to accept ProviderManager or creating a wrapper
		// For now, we test the manager directly
		input := s2s.NewAudioInput([]byte{1, 2, 3}, "test-session")
		convCtx := s2s.NewConversationContext("test-session")

		output, err := manager.Process(ctx, input, convCtx)
		require.NoError(t, err)
		assert.NotNil(t, output)
	})
}
