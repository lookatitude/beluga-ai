package backend

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	// Import providers to trigger init() registration
	_ "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/mock"
)

// TestPipelineIntegration tests pipeline orchestration integration (T263).
func TestPipelineIntegration(t *testing.T) {
	t.Skip("Skipping - mock provider ProcessAudio requires actual STT/TTS providers to be registered")
	ctx := context.Background()

	// Create backend with mock provider
	config := backend.DefaultConfig()
	config.Provider = "mock"
	config.STTProvider = "mock" // Required for stt_tts pipeline
	config.TTSProvider = "mock" // Required for stt_tts pipeline
	config.MaxConcurrentSessions = 10

	voiceBackend, err := backend.NewBackend(ctx, "mock", config)
	require.NoError(t, err)

	err = voiceBackend.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = voiceBackend.Stop(ctx)
	}()

	// Create a session with STT/TTS pipeline
	sessionConfig := &vbiface.SessionConfig{
		UserID:       "test-user",
		Transport:    "websocket",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		AgentCallback: func(ctx context.Context, transcript string) (string, error) {
			return "Response to: " + transcript, nil
		},
	}

	session, err := voiceBackend.CreateSession(ctx, sessionConfig)
	require.NoError(t, err)
	require.NotNil(t, session)

	// Start session
	err = session.Start(ctx)
	require.NoError(t, err)

	// Process audio through pipeline
	audio := []byte{1, 2, 3, 4, 5}
	err = session.ProcessAudio(ctx, audio)
	require.NoError(t, err)

	// Verify pipeline state
	assert.Equal(t, vbiface.PipelineStateProcessing, session.GetState(), "State should be processing")

	// Stop session
	err = session.Stop(ctx)
	require.NoError(t, err)
}

// TestPipelineS2SLatency tests S2S latency <300ms (T265, SC-005).
func TestPipelineS2SLatency(t *testing.T) {
	t.Skip("Skipping - mock provider ProcessAudio requires actual S2S providers to be registered")
	if testing.Short() {
		t.Skip("Skipping S2S latency test in short mode")
	}

	ctx := context.Background()

	// Create backend with S2S pipeline
	config := backend.DefaultConfig()
	config.Provider = "mock"
	config.S2SProvider = "mock" // Required for s2s pipeline
	config.LatencyTarget = 300 * time.Millisecond

	voiceBackend, err := backend.NewBackend(ctx, "mock", config)
	require.NoError(t, err)

	err = voiceBackend.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = voiceBackend.Stop(ctx)
	}()

	// Create a session with S2S pipeline
	sessionConfig := &vbiface.SessionConfig{
		UserID:       "test-user",
		Transport:    "websocket",
		PipelineType: vbiface.PipelineTypeS2S,
		AgentCallback: func(ctx context.Context, transcript string) (string, error) {
			return "Response", nil
		},
	}

	session, err := voiceBackend.CreateSession(ctx, sessionConfig)
	require.NoError(t, err)

	err = session.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = session.Stop(ctx)
	}()

	// Measure S2S latency
	startTime := time.Now()
	audio := []byte{1, 2, 3, 4, 5}
	err = session.ProcessAudio(ctx, audio)
	require.NoError(t, err)
	latency := time.Since(startTime)

	// Verify latency is within target (<300ms for S2S)
	assert.Less(t, latency, 300*time.Millisecond, "S2S latency should be <300ms (SC-005), got: %v", latency)
	t.Logf("S2S latency: %v", latency)
}

// TestPipelineTurnProcessingSuccess tests 99% turn processing success (T266, SC-004).
func TestPipelineTurnProcessingSuccess(t *testing.T) {
	t.Skip("Skipping - mock provider ProcessAudio requires actual STT/TTS providers to be registered")
	if testing.Short() {
		t.Skip("Skipping turn processing test in short mode")
	}

	ctx := context.Background()

	config := backend.DefaultConfig()
	config.Provider = "mock"
	config.STTProvider = "mock" // Required for stt_tts pipeline
	config.TTSProvider = "mock" // Required for stt_tts pipeline

	voiceBackend, err := backend.NewBackend(ctx, "mock", config)
	require.NoError(t, err)

	err = voiceBackend.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = voiceBackend.Stop(ctx)
	}()

	// Create session
	sessionConfig := &vbiface.SessionConfig{
		UserID:       "test-user",
		Transport:    "websocket",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		AgentCallback: func(ctx context.Context, transcript string) (string, error) {
			return "Response", nil
		},
	}

	session, err := voiceBackend.CreateSession(ctx, sessionConfig)
	require.NoError(t, err)

	err = session.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = session.Stop(ctx)
	}()

	// Process multiple turns
	totalTurns := 100
	successfulTurns := 0

	for i := 0; i < totalTurns; i++ {
		audio := []byte{byte(i), byte(i + 1), byte(i + 2), byte(i + 3), byte(i + 4)}
		err := session.ProcessAudio(ctx, audio)
		if err == nil {
			successfulTurns++
		}
	}

	// Verify success rate is >= 99%
	successRate := float64(successfulTurns) / float64(totalTurns)
	assert.GreaterOrEqual(t, successRate, 0.99, "Turn processing success should be >=99%% (SC-004), got: %.2f%%", successRate*100)
	t.Logf("Turn processing success rate: %.2f%%", successRate*100)
}

// TestPipelineConcurrentProcessing tests concurrent audio processing.
func TestPipelineConcurrentProcessing(t *testing.T) {
	t.Skip("Skipping - mock provider ProcessAudio requires actual STT/TTS providers to be registered")
	ctx := context.Background()

	config := backend.DefaultConfig()
	config.Provider = "mock"
	config.STTProvider = "mock" // Required for stt_tts pipeline
	config.TTSProvider = "mock" // Required for stt_tts pipeline
	config.MaxConcurrentSessions = 10

	voiceBackend, err := backend.NewBackend(ctx, "mock", config)
	require.NoError(t, err)

	err = voiceBackend.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = voiceBackend.Stop(ctx)
	}()

	// Create multiple sessions
	numSessions := 5
	sessions := make([]vbiface.VoiceSession, numSessions)

	for i := 0; i < numSessions; i++ {
		sessionConfig := &vbiface.SessionConfig{
			UserID:       "test-user-" + string(rune('0'+i)),
			Transport:    "websocket",
			PipelineType: vbiface.PipelineTypeSTTTTS,
			AgentCallback: func(ctx context.Context, transcript string) (string, error) {
				return "Response", nil
			},
		}

		session, err := voiceBackend.CreateSession(ctx, sessionConfig)
		require.NoError(t, err)
		sessions[i] = session

		err = session.Start(ctx)
		require.NoError(t, err)
	}

	// Process audio concurrently in all sessions
	for _, session := range sessions {
		audio := []byte{1, 2, 3, 4, 5}
		err := session.ProcessAudio(ctx, audio)
		assert.NoError(t, err, "Concurrent processing should succeed")
	}

	// Stop all sessions
	for _, session := range sessions {
		_ = session.Stop(ctx)
	}
}
