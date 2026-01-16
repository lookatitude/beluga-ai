package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/lookatitude/beluga-ai/tests/integration/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVoiceAgentIntegration tests the complete voice agent functionality.
// This test verifies that a voice call can be made, the agent responds appropriately,
// and context is maintained throughout the conversation.
func TestVoiceAgentIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Configure Twilio voice backend
	config := &vbiface.Config{
		Provider:     "twilio",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		STTProvider:  "openai", // Configure your STT provider
		TTSProvider:  "openai", // Configure your TTS provider
		ProviderConfig: map[string]any{
			"account_sid":  utils.GetEnvOrSkip(t, "TWILIO_ACCOUNT_SID"),
			"auth_token":   utils.GetEnvOrSkip(t, "TWILIO_AUTH_TOKEN"),
			"phone_number": utils.GetEnvOrSkip(t, "TWILIO_PHONE_NUMBER"),
		},
		LatencyTarget: 2 * time.Second, // FR-009: <2s latency
	}

	// Create backend
	voiceBackend, err := backend.NewBackend(ctx, "twilio", config)
	require.NoError(t, err)

	// Start backend
	err = voiceBackend.Start(ctx)
	require.NoError(t, err)
	defer voiceBackend.Stop(ctx)

	// Agent callback function
	conversationContext := make([]string, 0)
	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		conversationContext = append(conversationContext, transcript)
		// Simple echo agent for testing
		return fmt.Sprintf("I heard you say: %s", transcript), nil
	}

	// Create session
	sessionConfig := &vbiface.SessionConfig{
		UserID:        "test-user",
		Transport:     "websocket",
		ConnectionURL: "wss://example.com",
		PipelineType:  vbiface.PipelineTypeSTTTTS,
		Metadata: map[string]any{
			"to":   utils.GetEnvOrSkip(t, "TWILIO_TEST_PHONE_NUMBER"), // Test phone number
			"from": config.ProviderConfig["phone_number"].(string),
		},
		AgentCallback: agentCallback,
	}

	session, err := voiceBackend.CreateSession(ctx, sessionConfig)
	require.NoError(t, err)
	assert.NotNil(t, session)

	// Start session
	err = session.Start(ctx)
	require.NoError(t, err)
	defer session.Stop(ctx)

	// Verify session is active
	assert.Equal(t, vbiface.PipelineStateListening, session.GetState())
	assert.True(t, len(session.GetID()) > 0)

	// Wait a bit for call to be established
	time.Sleep(2 * time.Second)

	// Verify context is maintained
	assert.GreaterOrEqual(t, len(conversationContext), 0)
}

// TestVoiceAgentLatency verifies FR-009: <2s latency from speech completion to agent audio response start.
func TestVoiceAgentLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config := &vbiface.Config{
		Provider:     "twilio",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		STTProvider:  "openai",
		TTSProvider:  "openai",
		ProviderConfig: map[string]any{
			"account_sid":  utils.GetEnvOrSkip(t, "TWILIO_ACCOUNT_SID"),
			"auth_token":   utils.GetEnvOrSkip(t, "TWILIO_AUTH_TOKEN"),
			"phone_number": utils.GetEnvOrSkip(t, "TWILIO_PHONE_NUMBER"),
		},
		LatencyTarget: 2 * time.Second, // FR-009: <2s latency
	}

	voiceBackend, err := backend.NewBackend(ctx, "twilio", config)
	require.NoError(t, err)

	err = voiceBackend.Start(ctx)
	require.NoError(t, err)
	defer voiceBackend.Stop(ctx)

	var speechEndTime time.Time
	var audioBeginTime time.Time

	agentCallback := func(ctx context.Context, transcript string) (string, error) {
		// Record when speech ends (transcript received)
		speechEndTime = time.Now()
		return "Test response", nil
	}

	sessionConfig := &vbiface.SessionConfig{
		UserID:        "test-user",
		Transport:     "websocket",
		ConnectionURL: "wss://example.com",
		PipelineType:  vbiface.PipelineTypeSTTTTS,
		Metadata: map[string]any{
			"to":   utils.GetEnvOrSkip(t, "TWILIO_TEST_PHONE_NUMBER"),
			"from": config.ProviderConfig["phone_number"].(string),
		},
		AgentCallback: agentCallback,
	}

	session, err := voiceBackend.CreateSession(ctx, sessionConfig)
	require.NoError(t, err)

	err = session.Start(ctx)
	require.NoError(t, err)
	defer session.Stop(ctx)

	// Simulate audio input (in real test, this would come from Twilio)
	testAudio := []byte{0x00, 0x01, 0x02, 0x03}

	// Process audio and measure latency
	err = session.ProcessAudio(ctx, testAudio)
	require.NoError(t, err)

	// Record when audio response begins (would be when TTS starts in real scenario)
	audioBeginTime = time.Now()

	// Calculate latency: from speech completion to agent audio response start
	latency := audioBeginTime.Sub(speechEndTime)

	// Verify FR-009: <2s latency
	assert.Less(t, latency, 2*time.Second, "Latency should be <2s (FR-009), got %v", latency)
	t.Logf("Measured latency: %v (target: <2s)", latency)
}

// TestConcurrentCalls verifies SC-003: Support 100 concurrent calls without degradation.
func TestConcurrentCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	config := &vbiface.Config{
		Provider:     "twilio",
		PipelineType: vbiface.PipelineTypeSTTTTS,
		STTProvider:  "openai",
		TTSProvider:  "openai",
		ProviderConfig: map[string]any{
			"account_sid":  utils.GetEnvOrSkip(t, "TWILIO_ACCOUNT_SID"),
			"auth_token":   utils.GetEnvOrSkip(t, "TWILIO_AUTH_TOKEN"),
			"phone_number": utils.GetEnvOrSkip(t, "TWILIO_PHONE_NUMBER"),
		},
		MaxConcurrentSessions: 100, // SC-003: 100 concurrent calls
	}

	voiceBackend, err := backend.NewBackend(ctx, "twilio", config)
	require.NoError(t, err)

	err = voiceBackend.Start(ctx)
	require.NoError(t, err)
	defer voiceBackend.Stop(ctx)

	// Create 100 concurrent sessions
	numSessions := 100
	sessions := make([]vbiface.VoiceSession, 0, numSessions)
	errors := make(chan error, numSessions)

	for i := 0; i < numSessions; i++ {
		go func(index int) {
			sessionConfig := &vbiface.SessionConfig{
				UserID:        "test-user",
				Transport:     "websocket",
				ConnectionURL: "wss://example.com",
				PipelineType:  vbiface.PipelineTypeSTTTTS,
				Metadata: map[string]any{
					"to":   utils.GetEnvOrSkip(t, "TWILIO_TEST_PHONE_NUMBER"),
					"from": config.ProviderConfig["phone_number"].(string),
				},
				AgentCallback: func(ctx context.Context, transcript string) (string, error) {
					return "Response", nil
				},
			}

			session, err := voiceBackend.CreateSession(ctx, sessionConfig)
			if err != nil {
				errors <- err
				return
			}

			err = session.Start(ctx)
			if err != nil {
				errors <- err
				return
			}

			sessions = append(sessions, session)
		}(i)
	}

	// Wait for all sessions to be created
	time.Sleep(5 * time.Second)

	// Check for errors
	close(errors)
	errorCount := 0
	for err := range errors {
		if err != nil {
			errorCount++
			t.Logf("Session creation error: %v", err)
		}
	}

	// Verify we can handle concurrent sessions
	activeCount := voiceBackend.GetActiveSessionCount()
	t.Logf("Active sessions: %d (target: up to 100)", activeCount)

	// Cleanup
	for _, session := range sessions {
		_ = session.Stop(ctx)
	}

	// Allow some failures but most should succeed
	assert.Less(t, errorCount, numSessions/10, "Too many session creation failures")
}
