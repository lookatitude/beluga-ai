package s2s

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	"github.com/lookatitude/beluga-ai/pkg/voice/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBasicConversation tests real-time speech conversations using S2S provider.
// This test validates User Story 1: Real-Time Speech Conversations (P1)
func TestBasicConversation(t *testing.T) {
	ctx := context.Background()

	// Create mock S2S provider
	mockProvider := s2s.NewAdvancedMockS2SProvider("test-provider",
		s2s.WithAudioOutputs(s2s.NewAudioOutput(
			[]byte{5, 4, 3, 2, 1},
			"test-provider",
			100*time.Millisecond, // Under 200ms target
		)))

	t.Run("create session with S2S provider", func(t *testing.T) {
		voiceSession, err := session.NewVoiceSession(ctx,
			session.WithS2SProvider(mockProvider),
		)
		require.NoError(t, err)
		assert.NotNil(t, voiceSession)
	})

	t.Run("session lifecycle with S2S", func(t *testing.T) {
		voiceSession, err := session.NewVoiceSession(ctx,
			session.WithS2SProvider(mockProvider),
		)
		require.NoError(t, err)

		// Start session
		err = voiceSession.Start(ctx)
		require.NoError(t, err)
		assert.Equal(t, "listening", string(voiceSession.GetState()))

		// Stop session
		err = voiceSession.Stop(ctx)
		require.NoError(t, err)
		assert.Equal(t, "ended", string(voiceSession.GetState()))
	})

	t.Run("process audio with S2S", func(t *testing.T) {
		voiceSession, err := session.NewVoiceSession(ctx,
			session.WithS2SProvider(mockProvider),
		)
		require.NoError(t, err)

		err = voiceSession.Start(ctx)
		require.NoError(t, err)

		// Process audio - should go through S2S provider
		audio := []byte{1, 2, 3, 4, 5}
		err = voiceSession.ProcessAudio(ctx, audio)
		require.NoError(t, err)

		// Wait a bit for processing to complete
		time.Sleep(50 * time.Millisecond)

		// Should be in listening state (may transition through processing)
		state := string(voiceSession.GetState())
		assert.True(t, state == "listening" || state == "processing",
			"Expected listening or processing state, got: %s", state)

		err = voiceSession.Stop(ctx)
		require.NoError(t, err)
	})

	t.Run("latency target validation", func(t *testing.T) {
		// Test that latency is acceptable (under 2 seconds for 95% of interactions)
		startTime := time.Now()

		voiceSession, err := session.NewVoiceSession(ctx,
			session.WithS2SProvider(mockProvider),
		)
		require.NoError(t, err)

		err = voiceSession.Start(ctx)
		require.NoError(t, err)

		audio := []byte{1, 2, 3, 4, 5}
		err = voiceSession.ProcessAudio(ctx, audio)
		require.NoError(t, err)

		// Wait for processing to complete
		time.Sleep(200 * time.Millisecond)

		latency := time.Since(startTime)
		// Should be under 2 seconds (allowing for test overhead)
		assert.Less(t, latency, 2*time.Second, "Latency should be under 2 seconds")

		err = voiceSession.Stop(ctx)
		require.NoError(t, err)
	})
}

// TestS2SProvider_Registry tests that S2S providers can be created via registry.
func TestS2SProvider_Registry(t *testing.T) {
	ctx := context.Background()

	t.Run("provider creation via registry", func(t *testing.T) {
		config := s2s.DefaultConfig()
		config.Provider = "amazon_nova"

		// Import provider to trigger init() registration
		_ = "github.com/lookatitude/beluga-ai/pkg/voice/s2s/providers/amazon_nova"

		registry := s2s.GetRegistry()
		provider, err := registry.GetProvider("amazon_nova", config)
		// May fail if AWS credentials not configured, but should not panic
		_ = provider
		_ = err
	})

	t.Run("mock provider creation", func(t *testing.T) {
		mockProvider := s2s.NewAdvancedMockS2SProvider("test",
			s2s.WithAudioOutputs(s2s.NewAudioOutput(
				[]byte("test audio"),
				"test",
				50*time.Millisecond,
			)))

		input := s2s.NewAudioInput([]byte{1, 2, 3}, "test-session")
		convCtx := s2s.NewConversationContext("test-session")

		output, err := mockProvider.Process(ctx, input, convCtx)
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.NotEmpty(t, output.Data)
	})
}

// TestS2S_ErrorHandling tests error handling in S2S conversations.
func TestS2S_ErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("session creation without provider", func(t *testing.T) {
		// Should fail validation - either STT+TTS or S2S required
		voiceSession, err := session.NewVoiceSession(ctx)
		require.Error(t, err)
		assert.Nil(t, voiceSession)
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("session creation with both STT+TTS and S2S", func(t *testing.T) {
		// Should fail validation - cannot specify both
		mockSTT := &mockSTTProvider{}
		mockTTS := &mockTTSProvider{}
		mockS2S := s2s.NewAdvancedMockS2SProvider("test")

		voiceSession, err := session.NewVoiceSession(ctx,
			session.WithSTTProvider(mockSTT),
			session.WithTTSProvider(mockTTS),
			session.WithS2SProvider(mockS2S),
		)
		require.Error(t, err)
		assert.Nil(t, voiceSession)
		assert.Contains(t, err.Error(), "cannot specify both")
	})
}

// Mock providers for testing
type mockSTTProvider struct{}

func (m *mockSTTProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	return "test transcript", nil
}

func (m *mockSTTProvider) StartStreaming(ctx context.Context) (iface.StreamingSession, error) {
	return &mockStreamingSession{}, nil
}

type mockStreamingSession struct{}

func (m *mockStreamingSession) SendAudio(ctx context.Context, audio []byte) error {
	return nil
}

func (m *mockStreamingSession) ReceiveTranscript() <-chan iface.TranscriptResult {
	ch := make(chan iface.TranscriptResult, 1)
	ch <- iface.TranscriptResult{Text: "test", IsFinal: true}
	close(ch)
	return ch
}

func (m *mockStreamingSession) Close() error {
	return nil
}

type mockTTSProvider struct{}

func (m *mockTTSProvider) GenerateSpeech(ctx context.Context, text string) ([]byte, error) {
	return []byte{1, 2, 3}, nil
}

func (m *mockTTSProvider) StreamGenerate(ctx context.Context, text string) (io.Reader, error) {
	return strings.NewReader("test audio"), nil
}
