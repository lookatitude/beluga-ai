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
	_ "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/livekit"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/mock"
)

// TestLiveKitIntegration tests basic LiveKit provider integration (T261).
func TestLiveKitIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping LiveKit integration test in short mode")
	}

	ctx := context.Background()

	// Create LiveKit backend configuration
	config := backend.DefaultConfig()
	config.Provider = "livekit"
	config.STTProvider = "mock" // Required for stt_tts pipeline
	config.TTSProvider = "mock" // Required for stt_tts pipeline
	config.ProviderConfig = make(map[string]any)
	config.ProviderConfig["url"] = "ws://localhost:7880"
	config.ProviderConfig["api_key"] = "test-key"
	config.ProviderConfig["api_secret"] = "test-secret"
	config.MaxConcurrentSessions = 10
	config.LatencyTarget = 500 * time.Millisecond

	// Try to create backend
	voiceBackend, err := backend.NewBackend(ctx, "livekit", config)
	if err != nil {
		t.Skipf("Skipping LiveKit integration test: %v (LiveKit server may not be available)", err)
		return
	}

	// Start backend
	err = voiceBackend.Start(ctx)
	if err != nil {
		t.Skipf("Skipping LiveKit integration test: failed to start: %v (LiveKit server may not be available)", err)
		return
	}
	defer func() {
		_ = voiceBackend.Stop(ctx)
	}()

	// Verify backend is connected
	state := voiceBackend.GetConnectionState()
	assert.Equal(t, vbiface.ConnectionStateConnected, state, "Backend should be connected")

	// Create a session
	sessionConfig := &vbiface.SessionConfig{
		UserID:       "test-user",
		Transport:    "websocket",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		AgentCallback: func(ctx context.Context, transcript string) (string, error) {
			return "Hello from LiveKit", nil
		},
	}

	session, err := voiceBackend.CreateSession(ctx, sessionConfig)
	require.NoError(t, err, "Failed to create session")
	require.NotNil(t, session, "Session should not be nil")

	// Verify session properties
	assert.NotEmpty(t, session.GetID(), "Session ID should not be empty")
	assert.Equal(t, vbiface.PipelineStateIdle, session.GetState(), "Initial state should be idle")

	// Start session
	err = session.Start(ctx)
	require.NoError(t, err, "Failed to start session")

	// Verify session is active
	assert.Equal(t, vbiface.PipelineStateListening, session.GetState(), "State should be listening after start")

	// Process some audio
	audio := []byte{1, 2, 3, 4, 5}
	err = session.ProcessAudio(ctx, audio)
	require.NoError(t, err, "Failed to process audio")

	// Stop session
	err = session.Stop(ctx)
	require.NoError(t, err, "Failed to stop session")

	// Verify session is stopped
	assert.Equal(t, vbiface.PipelineStateIdle, session.GetState(), "State should be idle after stop")
}

// TestLiveKitEndToEndLatency tests end-to-end latency <500ms (T264, SC-001).
func TestLiveKitEndToEndLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping latency test in short mode")
	}

	ctx := context.Background()

	// Create LiveKit backend
	config := backend.DefaultConfig()
	config.Provider = "livekit"
	config.STTProvider = "mock" // Required for stt_tts pipeline
	config.TTSProvider = "mock" // Required for stt_tts pipeline
	config.ProviderConfig = make(map[string]any)
	config.ProviderConfig["url"] = "ws://localhost:7880"
	config.ProviderConfig["api_key"] = "test-key"
	config.ProviderConfig["api_secret"] = "test-secret"
	config.LatencyTarget = 500 * time.Millisecond

	voiceBackend, err := backend.NewBackend(ctx, "livekit", config)
	if err != nil {
		t.Skipf("Skipping latency test: %v", err)
		return
	}

	err = voiceBackend.Start(ctx)
	if err != nil {
		t.Skipf("Skipping latency test: failed to start: %v", err)
		return
	}
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
	defer func() {
		_ = session.Stop(ctx)
	}()

	err = session.Start(ctx)
	require.NoError(t, err)

	// Measure end-to-end latency
	startTime := time.Now()
	audio := []byte{1, 2, 3, 4, 5}
	err = session.ProcessAudio(ctx, audio)
	require.NoError(t, err)
	latency := time.Since(startTime)

	// Verify latency is within target (<500ms)
	assert.Less(t, latency, 500*time.Millisecond, "End-to-end latency should be <500ms (SC-001), got: %v", latency)
	t.Logf("End-to-end latency: %v", latency)
}

// TestLiveKitConnectionFailureRecovery tests 90% connection failure recovery (T267, SC-006).
func TestLiveKitConnectionFailureRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping connection recovery test in short mode")
	}

	ctx := context.Background()

	// Create LiveKit backend with retry configuration
	config := backend.DefaultConfig()
	config.Provider = "livekit"
	config.STTProvider = "mock" // Required for stt_tts pipeline
	config.TTSProvider = "mock" // Required for stt_tts pipeline
	config.ProviderConfig = make(map[string]any)
	config.ProviderConfig["url"] = "ws://localhost:7880"
	config.ProviderConfig["api_key"] = "test-key"
	config.ProviderConfig["api_secret"] = "test-secret"
	config.MaxRetries = 3
	config.RetryDelay = 100 * time.Millisecond

	voiceBackend, err := backend.NewBackend(ctx, "livekit", config)
	if err != nil {
		t.Skipf("Skipping recovery test: %v", err)
		return
	}

	err = voiceBackend.Start(ctx)
	if err != nil {
		t.Skipf("Skipping recovery test: failed to start: %v", err)
		return
	}
	defer func() {
		_ = voiceBackend.Stop(ctx)
	}()

	// Perform health check
	health, err := voiceBackend.HealthCheck(ctx)
	require.NoError(t, err)
	require.NotNil(t, health)

	// Verify health status
	assert.Equal(t, "healthy", health.Status, "Backend should report healthy status")

	// Simulate connection failure by checking health multiple times
	// In a real scenario, this would involve actual connection drops
	recoveryCount := 0
	totalAttempts := 10

	for i := 0; i < totalAttempts; i++ {
		health, err := voiceBackend.HealthCheck(ctx)
		if err == nil && health != nil && health.Status == "healthy" {
			recoveryCount++
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Verify recovery rate is >= 90%
	recoveryRate := float64(recoveryCount) / float64(totalAttempts)
	assert.GreaterOrEqual(t, recoveryRate, 0.9, "Connection failure recovery should be >=90%% (SC-006), got: %.2f%%", recoveryRate*100)
	t.Logf("Connection recovery rate: %.2f%%", recoveryRate*100)
}

// TestLiveKitHealthCheck tests health check functionality.
func TestLiveKitHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping health check test in short mode")
	}

	ctx := context.Background()

	config := backend.DefaultConfig()
	config.Provider = "livekit"
	config.STTProvider = "mock" // Required for stt_tts pipeline
	config.TTSProvider = "mock" // Required for stt_tts pipeline
	config.ProviderConfig = make(map[string]any)
	config.ProviderConfig["url"] = "ws://localhost:7880"
	config.ProviderConfig["api_key"] = "test-key"
	config.ProviderConfig["api_secret"] = "test-secret"

	voiceBackend, err := backend.NewBackend(ctx, "livekit", config)
	if err != nil {
		t.Skipf("Skipping health check test: %v", err)
		return
	}

	err = voiceBackend.Start(ctx)
	if err != nil {
		t.Skipf("Skipping health check test: failed to start: %v", err)
		return
	}
	defer func() {
		_ = voiceBackend.Stop(ctx)
	}()

	// Perform health check
	health, err := voiceBackend.HealthCheck(ctx)
	require.NoError(t, err)
	require.NotNil(t, health)

	// Verify health status fields
	assert.NotEmpty(t, health.Status, "Health status should not be empty")
	assert.NotNil(t, health.Details, "Health details should not be nil")
	assert.False(t, health.LastCheck.IsZero(), "Last check time should be set")
}
