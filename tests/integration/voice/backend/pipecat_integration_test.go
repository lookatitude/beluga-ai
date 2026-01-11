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
	_ "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/pipecat"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/mock"
)

// TestPipecatIntegration tests basic Pipecat provider integration (T262).
func TestPipecatIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Pipecat integration test in short mode")
	}

	ctx := context.Background()

	// Create Pipecat backend configuration
	config := backend.DefaultConfig()
	config.Provider = "pipecat"
	config.ProviderConfig = make(map[string]any)
	config.ProviderConfig["daily_api_key"] = "test-key"
	config.ProviderConfig["daily_api_url"] = "https://api.daily.co"
	config.ProviderConfig["pipecat_server_url"] = "ws://localhost:8080/ws"
	config.MaxConcurrentSessions = 10
	config.LatencyTarget = 500 * time.Millisecond

	// Try to create backend
	voiceBackend, err := backend.NewBackend(ctx, "pipecat", config)
	if err != nil {
		t.Skipf("Skipping Pipecat integration test: %v (Pipecat server may not be available)", err)
		return
	}

	// Start backend
	err = voiceBackend.Start(ctx)
	if err != nil {
		t.Skipf("Skipping Pipecat integration test: failed to start: %v (Pipecat server may not be available)", err)
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
			return "Hello from Pipecat", nil
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

// TestPipecatHealthCheck tests health check functionality for Pipecat.
func TestPipecatHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping health check test in short mode")
	}

	ctx := context.Background()

	config := backend.DefaultConfig()
	config.Provider = "pipecat"
	config.ProviderConfig = make(map[string]any)
	config.ProviderConfig["daily_api_key"] = "test-key"
	config.ProviderConfig["daily_api_url"] = "https://api.daily.co"
	config.ProviderConfig["pipecat_server_url"] = "ws://localhost:8080/ws"

	voiceBackend, err := backend.NewBackend(ctx, "pipecat", config)
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

// TestPipecatGracefulShutdown tests graceful shutdown functionality.
func TestPipecatGracefulShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping graceful shutdown test in short mode")
	}

	ctx := context.Background()

	config := backend.DefaultConfig()
	config.Provider = "pipecat"
	config.ProviderConfig = make(map[string]any)
	config.ProviderConfig["daily_api_key"] = "test-key"
	config.ProviderConfig["daily_api_url"] = "https://api.daily.co"
	config.ProviderConfig["pipecat_server_url"] = "ws://localhost:8080/ws"

	voiceBackend, err := backend.NewBackend(ctx, "pipecat", config)
	if err != nil {
		t.Skipf("Skipping graceful shutdown test: %v", err)
		return
	}

	err = voiceBackend.Start(ctx)
	if err != nil {
		t.Skipf("Skipping graceful shutdown test: failed to start: %v", err)
		return
	}

	// Create a session
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

	// Stop backend gracefully (should complete in-flight conversations)
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = voiceBackend.Stop(shutdownCtx)
	require.NoError(t, err, "Graceful shutdown should complete successfully")

	// Verify backend is disconnected
	state := voiceBackend.GetConnectionState()
	assert.Equal(t, vbiface.ConnectionStateDisconnected, state, "Backend should be disconnected after shutdown")
}
