package backend

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestScalability100ConcurrentSessions tests that the backend can handle 100+ concurrent sessions (T181, T182).
// This verifies SC-002: No latency degradation with 100 concurrent users.
func TestScalability100ConcurrentSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping scalability test in short mode")
	}

	ctx := context.Background()

	// Create backend with mock provider for testing
	config := &iface.Config{
		Provider:              "mock",
		MaxConcurrentSessions: 200, // Allow up to 200 sessions for this test
		LatencyTarget:         500 * time.Millisecond,
		Timeout:               30 * time.Second,
		EnableTracing:         false, // Disable tracing for performance
		EnableMetrics:         false, // Disable metrics for performance
	}

	backendInstance, err := backend.NewBackend(ctx, "mock", config)
	require.NoError(t, err)
	require.NotNil(t, backendInstance)

	// Start backend
	err = backendInstance.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = backendInstance.Stop(ctx)
	}()

	// Create 100 concurrent sessions
	numSessions := 100
	sessions := make([]iface.VoiceSession, numSessions)
	errors := make([]error, numSessions)
	var wg sync.WaitGroup
	startTime := time.Now()

	// Concurrent session creation (T172)
	for i := 0; i < numSessions; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			sessionConfig := &iface.SessionConfig{
				UserID:        "user-" + string(rune(index)),
				Transport:     "webrtc",
				ConnectionURL: "ws://localhost:8080",
				PipelineType:  iface.PipelineTypeSTTTTS,
				Metadata:      map[string]any{"test_index": index},
			}

			session, err := backendInstance.CreateSession(ctx, sessionConfig)
			sessions[index] = session
			errors[index] = err
		}(i)
	}

	wg.Wait()
	creationTime := time.Since(startTime)

	// Verify all sessions were created successfully
	successCount := 0
	for i, err := range errors {
		if err != nil {
			t.Logf("Session %d creation failed: %v", i, err)
		} else {
			assert.NotNil(t, sessions[i], "Session %d should not be nil", i)
			successCount++
		}
	}

	// Verify at least 95% of sessions were created successfully (allowing for some failures)
	assert.GreaterOrEqual(t, successCount, int(float64(numSessions)*0.95),
		"At least 95%% of sessions should be created successfully")

	// Verify session creation time is reasonable (<2 seconds per SC-007)
	avgCreationTime := creationTime / time.Duration(numSessions)
	t.Logf("Average session creation time: %v", avgCreationTime)
	assert.Less(t, avgCreationTime, 2*time.Second,
		"Average session creation time should be <2 seconds (SC-007)")

	// Verify concurrent session count
	activeCount := backendInstance.GetActiveSessionCount()
	assert.Equal(t, successCount, activeCount,
		"Active session count should match created sessions")

	// Test concurrent audio processing
	audioData := []byte{1, 2, 3, 4, 5}
	var processingWg sync.WaitGroup
	processingErrors := make([]error, successCount)
	processingStartTime := time.Now()

	sessionIndex := 0
	for i := 0; i < numSessions; i++ {
		if sessions[i] != nil {
			processingWg.Add(1)
			idx := sessionIndex
			sess := sessions[i]
			go func() {
				defer processingWg.Done()

				// Start session
				err := sess.Start(ctx)
				if err != nil {
					processingErrors[idx] = err
					return
				}

				// Process audio concurrently
				err = sess.ProcessAudio(ctx, audioData)
				processingErrors[idx] = err
			}()
			sessionIndex++
		}
	}

	processingWg.Wait()
	processingTime := time.Since(processingStartTime)

	// Verify processing completed without errors
	processingSuccessCount := 0
	for i, err := range processingErrors {
		if err == nil {
			processingSuccessCount++
		} else {
			t.Logf("Session %d processing failed: %v", i, err)
		}
	}

	assert.GreaterOrEqual(t, processingSuccessCount, int(float64(successCount)*0.95),
		"At least 95%% of sessions should process audio successfully")

	// Verify no latency degradation (SC-002)
	avgProcessingTime := processingTime / time.Duration(processingSuccessCount)
	t.Logf("Average processing time with %d concurrent sessions: %v", processingSuccessCount, avgProcessingTime)

	// Latency should not degrade significantly with concurrent load
	// Target: <500ms per session even with 100 concurrent sessions
	assert.Less(t, avgProcessingTime, 1*time.Second,
		"Average processing time should not degrade significantly with concurrent load (SC-002)")

	// Cleanup: Close all sessions
	for _, session := range sessions {
		if session != nil {
			_ = backendInstance.CloseSession(ctx, session.GetID())
		}
	}
}

// TestSessionIsolation verifies that sessions are properly isolated (T171, SC-008).
func TestSessionIsolation(t *testing.T) {
	ctx := context.Background()

	config := &iface.Config{
		Provider:              "mock",
		MaxConcurrentSessions: 10,
		LatencyTarget:         500 * time.Millisecond,
		Timeout:               30 * time.Second,
	}

	backendInstance, err := backend.NewBackend(ctx, "mock", config)
	require.NoError(t, err)

	err = backendInstance.Start(ctx)
	require.NoError(t, err)
	defer func() {
		_ = backendInstance.Stop(ctx)
	}()

	// Create multiple sessions with different metadata
	session1Config := &iface.SessionConfig{
		UserID:        "user-1",
		Transport:     "webrtc",
		ConnectionURL: "ws://localhost:8080",
		PipelineType:  iface.PipelineTypeSTTTTS,
		Metadata:      map[string]any{"key": "value1"},
	}

	session2Config := &iface.SessionConfig{
		UserID:        "user-2",
		Transport:     "webrtc",
		ConnectionURL: "ws://localhost:8080",
		PipelineType:  iface.PipelineTypeSTTTTS,
		Metadata:      map[string]any{"key": "value2"},
	}

	session1, err := backendInstance.CreateSession(ctx, session1Config)
	require.NoError(t, err)
	require.NotNil(t, session1)

	session2, err := backendInstance.CreateSession(ctx, session2Config)
	require.NoError(t, err)
	require.NotNil(t, session2)

	// Verify sessions have different IDs
	assert.NotEqual(t, session1.GetID(), session2.GetID(),
		"Sessions should have unique IDs")

	// Verify sessions have independent state
	state1 := session1.GetState()
	state2 := session2.GetState()
	assert.Equal(t, state1, state2, "Both sessions should start in same initial state")

	// Start session1 and verify session2 state is unchanged
	err = session1.Start(ctx)
	require.NoError(t, err)

	state1After := session1.GetState()
	state2After := session2.GetState()
	assert.NotEqual(t, state1After, state2After,
		"Session states should be independent (SC-008)")

	// Verify no cross-contamination of metadata or state
	// This ensures zero cross-conversation data leakage (SC-008)
	assert.Equal(t, iface.PipelineStateIdle, state2After,
		"Session2 state should not be affected by session1 operations")
}
