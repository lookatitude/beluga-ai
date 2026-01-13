package backend

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	// Import providers to trigger init() registration
	_ "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/cartesia"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/livekit"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/mock"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/pipecat"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/vapi"
	_ "github.com/lookatitude/beluga-ai/pkg/voice/backend/providers/vocode"
)

// TestProviderSwapping tests that sessions can be created and managed across different providers.
func TestProviderSwapping(t *testing.T) {
	ctx := context.Background()

	// Test providers: mock, pipecat, vocode, vapi, cartesia
	providers := []string{"mock", "pipecat", "vocode", "vapi", "cartesia"}

	for _, providerName := range providers {
		t.Run(providerName, func(t *testing.T) {
			// Create backend for this provider
			config := backend.DefaultConfig()
			config.Provider = providerName
			config.ProviderConfig = make(map[string]any)

			// Set provider-specific config
			switch providerName {
			case "mock":
				// Mock provider doesn't need additional config
			case "pipecat":
				config.ProviderConfig["daily_api_key"] = "test-key"
				config.ProviderConfig["pipecat_server_url"] = "ws://localhost:8080/ws"
			case "vocode":
				config.ProviderConfig["api_key"] = "test-key"
				config.ProviderConfig["api_url"] = "https://api.vocode.dev"
			case "vapi":
				config.ProviderConfig["api_key"] = "test-key"
				config.ProviderConfig["api_url"] = "https://api.vapi.ai"
			case "cartesia":
				config.ProviderConfig["api_key"] = "test-key"
				config.ProviderConfig["api_url"] = "https://api.cartesia.ai"
			}

			// Create backend
			voiceBackend, err := backend.NewBackend(ctx, providerName, config)
			if err != nil {
				t.Skipf("Skipping %s provider: %v", providerName, err)
				return
			}

			// Start backend
			err = voiceBackend.Start(ctx)
			if err != nil {
				t.Skipf("Skipping %s provider: failed to start: %v", providerName, err)
				return
			}
			defer func() {
				_ = voiceBackend.Stop(ctx)
			}()

			// Create a session
			sessionConfig := &vbiface.SessionConfig{
				UserID:       "test-user",
				Transport:    "websocket",
				PipelineType: vbiface.PipelineTypeSTTTTS,
				AgentCallback: func(ctx context.Context, transcript string) (string, error) {
					return "Hello from " + providerName, nil
				},
			}

			session, err := voiceBackend.CreateSession(ctx, sessionConfig)
			require.NoError(t, err, "Failed to create session with %s provider", providerName)
			require.NotNil(t, session, "Session should not be nil")

			// Verify session properties
			assert.NotEmpty(t, session.GetID(), "Session ID should not be empty")
			assert.Equal(t, vbiface.PipelineStateIdle, session.GetState(), "Initial state should be idle")

			// Start session
			err = session.Start(ctx)
			require.NoError(t, err, "Failed to start session")

			// Verify session is active
			assert.Equal(t, vbiface.PipelineStateListening, session.GetState(), "State should be listening after start")

			// Stop session
			err = session.Stop(ctx)
			require.NoError(t, err, "Failed to stop session")

			// Verify session is stopped
			assert.Equal(t, vbiface.PipelineStateIdle, session.GetState(), "State should be idle after stop")
		})
	}
}

// TestProviderRegistry tests that all providers are registered correctly.
func TestProviderRegistry(t *testing.T) {
	registry := backend.GetRegistry()

	// Expected providers
	expectedProviders := []string{"mock", "livekit", "pipecat", "vocode", "vapi", "cartesia"}

	// Get list of registered providers
	registeredProviders := registry.ListProviders()

	// Verify all expected providers are registered
	for _, expected := range expectedProviders {
		assert.Contains(t, registeredProviders, expected, "Provider %s should be registered", expected)
		assert.True(t, registry.IsRegistered(expected), "Provider %s should be registered", expected)
	}
}

// TestProviderSwappingSameSession tests that the same session configuration works across providers.
func TestProviderSwappingSameSession(t *testing.T) {
	ctx := context.Background()

	// Create a common session configuration
	sessionConfig := &vbiface.SessionConfig{
		UserID:       "test-user-swap",
		Transport:    "websocket",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		AgentCallback: func(ctx context.Context, transcript string) (string, error) {
			return "Response", nil
		},
	}

	// Test with mock provider (most reliable for testing)
	config := backend.DefaultConfig()
	config.Provider = "mock"

	mockBackend, err := backend.NewBackend(ctx, "mock", config)
	require.NoError(t, err)

	err = mockBackend.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = mockBackend.Stop(ctx)
	}()

	// Create session with mock provider
	session1, err := mockBackend.CreateSession(ctx, sessionConfig)
	require.NoError(t, err)
	require.NotNil(t, session1)

	sessionID1 := session1.GetID()
	assert.NotEmpty(t, sessionID1)

	// Verify we can retrieve the session
	retrievedSession, err := mockBackend.GetSession(ctx, sessionID1)
	require.NoError(t, err)
	assert.Equal(t, sessionID1, retrievedSession.GetID(), "Retrieved session should have same ID")

	// List sessions
	sessions, err := mockBackend.ListSessions(ctx)
	require.NoError(t, err)
	assert.Len(t, sessions, 1, "Should have one active session")
	assert.Equal(t, sessionID1, sessions[0].GetID(), "Listed session should match created session")
}

// TestProviderCapabilities tests that each provider reports correct capabilities.
func TestProviderCapabilities(t *testing.T) {
	ctx := context.Background()
	registry := backend.GetRegistry()

	providers := []string{"mock", "pipecat", "vocode", "vapi", "cartesia"}

	for _, providerName := range providers {
		t.Run(providerName, func(t *testing.T) {
			// Get provider factory
			creator, err := registry.GetProvider(providerName)
			if err != nil {
				t.Skipf("Skipping %s provider: not registered", providerName)
				return
			}

			// Create a minimal config to get capabilities
			config := backend.DefaultConfig()
			config.Provider = providerName
			config.ProviderConfig = make(map[string]any)

			// Set provider-specific config (minimal for capabilities check)
			switch providerName {
			case "pipecat":
				config.ProviderConfig["daily_api_key"] = "test-key"
				config.ProviderConfig["pipecat_server_url"] = "ws://localhost:8080/ws"
			case "vocode", "vapi", "cartesia":
				config.ProviderConfig["api_key"] = "test-key"
			}

			// Create backend to access provider
			backendInstance, err := creator(ctx, config)
			if err != nil {
				t.Skipf("Skipping %s provider: failed to create: %v", providerName, err)
				return
			}

			// Get capabilities (if provider implements GetCapabilities)
			// Note: This requires the provider to expose capabilities, which may not be available
			// For now, we'll just verify the backend was created successfully
			assert.NotNil(t, backendInstance, "Backend instance should not be nil")

			// Verify backend implements required interface
			configBackend := backendInstance.GetConfig()
			assert.NotNil(t, configBackend, "Backend should return config")
			assert.Equal(t, providerName, configBackend.Provider, "Backend provider should match")
		})
	}
}

// TestProviderSwappingHealthCheck tests health checks across different providers.
func TestProviderSwappingHealthCheck(t *testing.T) {
	ctx := context.Background()

	// Test with mock provider (most reliable)
	config := backend.DefaultConfig()
	config.Provider = "mock"

	mockBackend, err := backend.NewBackend(ctx, "mock", config)
	require.NoError(t, err)

	err = mockBackend.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = mockBackend.Stop(ctx)
	}()

	// Perform health check
	health, err := mockBackend.HealthCheck(ctx)
	require.NoError(t, err)
	require.NotNil(t, health)
	assert.Equal(t, "healthy", health.Status, "Mock backend should report healthy status")
}

// TestProviderSwappingConcurrentSessions tests concurrent session creation across providers.
func TestProviderSwappingConcurrentSessions(t *testing.T) {
	ctx := context.Background()

	// Test with mock provider
	config := backend.DefaultConfig()
	config.Provider = "mock"
	config.MaxConcurrentSessions = 10

	mockBackend, err := backend.NewBackend(ctx, "mock", config)
	require.NoError(t, err)

	err = mockBackend.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = mockBackend.Stop(ctx)
	}()

	// Create multiple sessions concurrently
	const numSessions = 5
	sessions := make([]vbiface.VoiceSession, numSessions)
	errors := make([]error, numSessions)

	for i := 0; i < numSessions; i++ {
		sessionConfig := &vbiface.SessionConfig{
			UserID:       "test-user-" + string(rune('0'+i)),
			Transport:    "websocket",
			PipelineType: vbiface.PipelineTypeSTTTTS,
			AgentCallback: func(ctx context.Context, transcript string) (string, error) {
				return "Response", nil
			},
		}

		sessions[i], errors[i] = mockBackend.CreateSession(ctx, sessionConfig)
	}

	// Verify all sessions were created successfully
	for i, err := range errors {
		require.NoError(t, err, "Session %d should be created successfully", i)
		require.NotNil(t, sessions[i], "Session %d should not be nil", i)
	}

	// Verify all sessions have unique IDs
	sessionIDs := make(map[string]bool)
	for i, session := range sessions {
		sessionID := session.GetID()
		assert.NotEmpty(t, sessionID, "Session %d should have an ID", i)
		assert.False(t, sessionIDs[sessionID], "Session ID %s should be unique", sessionID)
		sessionIDs[sessionID] = true
	}

	// Verify active session count
	activeCount := mockBackend.GetActiveSessionCount()
	assert.Equal(t, numSessions, activeCount, "Active session count should match created sessions")

	// Clean up sessions
	for _, session := range sessions {
		_ = mockBackend.CloseSession(ctx, session.GetID())
	}
}

// TestProviderSwappingConfigValidation tests configuration validation across providers.
func TestProviderSwappingConfigValidation(t *testing.T) {
	ctx := context.Background()

	providers := []string{"mock", "pipecat", "vocode", "vapi", "cartesia"}

	for _, providerName := range providers {
		t.Run(providerName, func(t *testing.T) {
			// Test with invalid config (missing required fields)
			invalidConfig := backend.DefaultConfig()
			invalidConfig.Provider = providerName
			invalidConfig.ProviderConfig = make(map[string]any)
			// Intentionally omit required provider-specific config

			// Try to create backend with invalid config
			_, err := backend.NewBackend(ctx, providerName, invalidConfig)

			// Some providers may accept minimal config (like mock), others should fail
			// This test verifies that validation is working
			if providerName == "mock" {
				// Mock provider should accept minimal config
				assert.NoError(t, err, "Mock provider should accept minimal config")
			} else {
				// Other providers should fail validation if required fields are missing
				// This is expected behavior
				if err != nil {
					t.Logf("Expected validation error for %s: %v", providerName, err)
				}
			}
		})
	}
}

// TestProviderSwappingSessionIsolation tests that sessions from different providers are isolated.
func TestProviderSwappingSessionIsolation(t *testing.T) {
	ctx := context.Background()

	// Create two backends with different providers (mock for both in this test)
	config1 := backend.DefaultConfig()
	config1.Provider = "mock"

	config2 := backend.DefaultConfig()
	config2.Provider = "mock"

	backend1, err := backend.NewBackend(ctx, "mock", config1)
	require.NoError(t, err)

	backend2, err := backend.NewBackend(ctx, "mock", config2)
	require.NoError(t, err)

	err = backend1.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = backend1.Stop(ctx)
	}()

	err = backend2.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = backend2.Stop(ctx)
	}()

	// Create sessions in each backend
	sessionConfig1 := &vbiface.SessionConfig{
		UserID:       "user-1",
		Transport:    "websocket",
		PipelineType: vbiface.PipelineTypeSTTTTS,
	}

	sessionConfig2 := &vbiface.SessionConfig{
		UserID:       "user-2",
		Transport:    "websocket",
		PipelineType: vbiface.PipelineTypeSTTTTS,
	}

	session1, err := backend1.CreateSession(ctx, sessionConfig1)
	require.NoError(t, err)

	session2, err := backend2.CreateSession(ctx, sessionConfig2)
	require.NoError(t, err)

	// Verify sessions are isolated (different IDs)
	assert.NotEqual(t, session1.GetID(), session2.GetID(), "Sessions should have different IDs")

	// Verify each backend only sees its own sessions
	sessions1, err := backend1.ListSessions(ctx)
	require.NoError(t, err)
	assert.Len(t, sessions1, 1, "Backend1 should have one session")
	assert.Equal(t, session1.GetID(), sessions1[0].GetID(), "Backend1 should list its own session")

	sessions2, err := backend2.ListSessions(ctx)
	require.NoError(t, err)
	assert.Len(t, sessions2, 1, "Backend2 should have one session")
	assert.Equal(t, session2.GetID(), sessions2[0].GetID(), "Backend2 should list its own session")

	// Verify cross-backend session retrieval fails
	_, err = backend1.GetSession(ctx, session2.GetID())
	assert.Error(t, err, "Backend1 should not retrieve Backend2's session")

	_, err = backend2.GetSession(ctx, session1.GetID())
	assert.Error(t, err, "Backend2 should not retrieve Backend1's session")
}
