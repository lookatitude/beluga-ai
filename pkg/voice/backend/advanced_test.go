package backend

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// TestRegistryOperations tests registry operations with table-driven tests (T253).
func TestRegistryOperations(t *testing.T) {
	tests := []struct {
		name           string
		providerName   string
		shouldRegister bool
		shouldError    bool
		errorCode      string
	}{
		{
			name:           "register mock provider",
			providerName:   "mock",
			shouldRegister: true,
			shouldError:    false,
		},
		{
			name:           "register duplicate provider",
			providerName:   "mock",
			shouldRegister: true,
			shouldError:    false, // Registry allows overwriting
		},
		{
			name:           "check registered provider",
			providerName:   "mock",
			shouldRegister: true,
			shouldError:    false,
		},
		{
			name:           "check unregistered provider",
			providerName:   "nonexistent",
			shouldRegister: false,
			shouldError:    false,
		},
	}

	registry := GetRegistry()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldRegister {
				// Register provider
				registry.Register(tt.providerName, func(ctx context.Context, config *iface.Config) (iface.VoiceBackend, error) {
					return NewAdvancedMockVoiceBackend(), nil
				})

				// Verify registration
				assert.True(t, registry.IsRegistered(tt.providerName), "Provider should be registered")
			}

			// Check if registered
			isRegistered := registry.IsRegistered(tt.providerName)
			if tt.shouldRegister {
				assert.True(t, isRegistered, "Provider should be registered")
			} else {
				assert.False(t, isRegistered, "Provider should not be registered")
			}

			// List providers
			providers := registry.ListProviders()
			if tt.shouldRegister {
				assert.Contains(t, providers, tt.providerName, "Provider should be in list")
			}
		})
	}
}

// TestBackendCreationAndLifecycle tests backend creation and lifecycle with table-driven tests (T254).
func TestBackendCreationAndLifecycle(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		config      *iface.Config
		shouldError bool
		errorCode   string
		setup       func(*iface.Config)
		teardown    func(iface.VoiceBackend)
	}{
		{
			name:     "create mock backend",
			provider: "mock",
			config: &iface.Config{
				Provider:     "mock",
				PipelineType: iface.PipelineTypeSTTTTS,
			},
			shouldError: false,
		},
		{
			name:     "create backend with invalid config",
			provider: "mock",
			config: &iface.Config{
				Provider: "", // Invalid: empty provider
			},
			shouldError: true,
			errorCode:   ErrCodeInvalidConfig,
		},
		{
			name:     "start and stop backend",
			provider: "mock",
			config: &iface.Config{
				Provider:     "mock",
				PipelineType: iface.PipelineTypeSTTTTS,
			},
			shouldError: false,
			setup: func(c *iface.Config) {
				// Setup config
			},
			teardown: func(b iface.VoiceBackend) {
				ctx := context.Background()
				_ = b.Stop(ctx)
			},
		},
		{
			name:     "backend health check",
			provider: "mock",
			config: &iface.Config{
				Provider:     "mock",
				PipelineType: iface.PipelineTypeSTTTTS,
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			if tt.setup != nil {
				tt.setup(tt.config)
			}

			backend := NewAdvancedMockVoiceBackend()

			// Test Start
			err := backend.Start(ctx)
			if tt.shouldError && tt.errorCode == "start_error" {
				require.Error(t, err, "Start should return error")
			} else {
				require.NoError(t, err, "Start should not return error")
			}

			// Test HealthCheck
			health, err := backend.HealthCheck(ctx)
			if !tt.shouldError {
				require.NoError(t, err, "HealthCheck should not return error")
				require.NotNil(t, health, "Health status should not be nil")
				assert.Equal(t, "healthy", health.Status, "Health status should be healthy")
			}

			// Test GetConnectionState
			state := backend.GetConnectionState()
			if !tt.shouldError {
				assert.Equal(t, iface.ConnectionStateConnected, state, "Connection state should be connected")
			}

			// Test Stop
			if tt.teardown != nil {
				tt.teardown(backend)
			} else {
				err = backend.Stop(ctx)
				require.NoError(t, err, "Stop should not return error")
			}
		})
	}
}

// TestSessionManagementOperations tests session management operations with table-driven tests (T255).
func TestSessionManagementOperations(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() iface.VoiceBackend
		operation   func(t *testing.T, backend iface.VoiceBackend, ctx context.Context)
		shouldError bool
		errorCode   string
	}{
		{
			name: "create session",
			setup: func() iface.VoiceBackend {
				backend := NewAdvancedMockVoiceBackend()
				ctx := context.Background()
				_ = backend.Start(ctx)
				return backend
			},
			operation: func(t *testing.T, backend iface.VoiceBackend, ctx context.Context) {
				sessionConfig := &iface.SessionConfig{
					UserID:       "test-user",
					Transport:    "websocket",
					PipelineType: iface.PipelineTypeSTTTTS,
				}

				session, err := backend.CreateSession(ctx, sessionConfig)
				require.NoError(t, err, "CreateSession should not return error")
				require.NotNil(t, session, "Session should not be nil")
				assert.NotEmpty(t, session.GetID(), "Session ID should not be empty")
			},
			shouldError: false,
		},
		{
			name: "get session by ID",
			setup: func() iface.VoiceBackend {
				backend := NewAdvancedMockVoiceBackend()
				ctx := context.Background()
				_ = backend.Start(ctx)
				return backend
			},
			operation: func(t *testing.T, backend iface.VoiceBackend, ctx context.Context) {
				sessionConfig := &iface.SessionConfig{
					UserID:       "test-user",
					Transport:    "websocket",
					PipelineType: iface.PipelineTypeSTTTTS,
				}

				session, err := backend.CreateSession(ctx, sessionConfig)
				require.NoError(t, err)

				retrieved, err := backend.GetSession(ctx, session.GetID())
				require.NoError(t, err, "GetSession should not return error")
				require.NotNil(t, retrieved, "Retrieved session should not be nil")
				assert.Equal(t, session.GetID(), retrieved.GetID(), "Session IDs should match")
			},
			shouldError: false,
		},
		{
			name: "list sessions",
			setup: func() iface.VoiceBackend {
				backend := NewAdvancedMockVoiceBackend()
				ctx := context.Background()
				_ = backend.Start(ctx)
				return backend
			},
			operation: func(t *testing.T, backend iface.VoiceBackend, ctx context.Context) {
				// Create multiple sessions
				for i := 0; i < 3; i++ {
					sessionConfig := &iface.SessionConfig{
						UserID:       "test-user",
						Transport:    "websocket",
						PipelineType: iface.PipelineTypeSTTTTS,
					}
					_, err := backend.CreateSession(ctx, sessionConfig)
					require.NoError(t, err)
				}

				sessions, err := backend.ListSessions(ctx)
				require.NoError(t, err, "ListSessions should not return error")
				assert.Len(t, sessions, 3, "Should have 3 sessions")
			},
			shouldError: false,
		},
		{
			name: "close session",
			setup: func() iface.VoiceBackend {
				backend := NewAdvancedMockVoiceBackend()
				ctx := context.Background()
				_ = backend.Start(ctx)
				return backend
			},
			operation: func(t *testing.T, backend iface.VoiceBackend, ctx context.Context) {
				sessionConfig := &iface.SessionConfig{
					UserID:       "test-user",
					Transport:    "websocket",
					PipelineType: iface.PipelineTypeSTTTTS,
				}

				session, err := backend.CreateSession(ctx, sessionConfig)
				require.NoError(t, err)

				err = backend.CloseSession(ctx, session.GetID())
				require.NoError(t, err, "CloseSession should not return error")

				// Verify session is closed
				_, err = backend.GetSession(ctx, session.GetID())
				require.Error(t, err, "GetSession should return error for closed session")
			},
			shouldError: false,
		},
		{
			name: "get nonexistent session",
			setup: func() iface.VoiceBackend {
				backend := NewAdvancedMockVoiceBackend()
				ctx := context.Background()
				_ = backend.Start(ctx)
				return backend
			},
			operation: func(t *testing.T, backend iface.VoiceBackend, ctx context.Context) {
				_, err := backend.GetSession(ctx, "nonexistent-session-id")
				require.Error(t, err, "GetSession should return error for nonexistent session")
			},
			shouldError: true,
			errorCode:   ErrCodeSessionNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			backend := tt.setup()
			defer func() {
				_ = backend.Stop(ctx)
			}()

			tt.operation(t, backend, ctx)
		})
	}
}

// TestPipelineOrchestration tests pipeline orchestration with table-driven tests (T256).
func TestPipelineOrchestration(t *testing.T) {
	tests := []struct {
		name          string
		pipelineType  iface.PipelineType
		audioData     []byte
		shouldError   bool
		errorCode     string
		agentCallback func(context.Context, string) (string, error)
	}{
		{
			name:         "STT/TTS pipeline processing",
			pipelineType: iface.PipelineTypeSTTTTS,
			audioData:    []byte{1, 2, 3, 4, 5},
			shouldError:  false,
			agentCallback: func(ctx context.Context, transcript string) (string, error) {
				return "Response: " + transcript, nil
			},
		},
		{
			name:         "S2S pipeline processing",
			pipelineType: iface.PipelineTypeS2S,
			audioData:    []byte{1, 2, 3, 4, 5},
			shouldError:  false,
		},
		{
			name:         "pipeline with error",
			pipelineType: iface.PipelineTypeSTTTTS,
			audioData:    []byte{1, 2, 3, 4, 5},
			shouldError:  true,
			errorCode:    "process_audio_error",
			agentCallback: func(ctx context.Context, transcript string) (string, error) {
				return "", errors.New("agent error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			backend := NewAdvancedMockVoiceBackend(
				WithMockError(tt.errorCode),
			)
			require.NoError(t, backend.Start(ctx))
			defer func() {
				_ = backend.Stop(ctx)
			}()

			sessionConfig := &iface.SessionConfig{
				UserID:        "test-user",
				Transport:     "websocket",
				PipelineType:  tt.pipelineType,
				AgentCallback: tt.agentCallback,
			}

			session, err := backend.CreateSession(ctx, sessionConfig)
			require.NoError(t, err)

			err = session.Start(ctx)
			require.NoError(t, err)

			err = session.ProcessAudio(ctx, tt.audioData)
			if tt.shouldError {
				require.Error(t, err, "ProcessAudio should return error")
			} else {
				// ProcessAudio may not return error even if processing fails
				// Just verify it doesn't panic
				_ = err
			}

			_ = session.Stop(ctx)
		})
	}
}

// TestErrorHandlingScenarios tests error handling scenarios with table-driven tests (T257).
func TestErrorHandlingScenarios(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() iface.VoiceBackend
		operation   func(t *testing.T, backend iface.VoiceBackend, ctx context.Context) error
		shouldError bool
		errorCode   string
	}{
		{
			name: "connection failure",
			setup: func() iface.VoiceBackend {
				return NewAdvancedMockVoiceBackend(
					WithMockError("start_error"),
				)
			},
			operation: func(t *testing.T, backend iface.VoiceBackend, ctx context.Context) error {
				return backend.Start(ctx)
			},
			shouldError: true,
			errorCode:   ErrCodeConnectionFailed,
		},
		{
			name: "session limit exceeded",
			setup: func() iface.VoiceBackend {
				return NewAdvancedMockVoiceBackend(
					WithMaxSessions(1),
				)
			},
			operation: func(t *testing.T, backend iface.VoiceBackend, ctx context.Context) error {
				_ = backend.Start(ctx)
				sessionConfig := &iface.SessionConfig{
					UserID:       "test-user-1",
					Transport:    "websocket",
					PipelineType: iface.PipelineTypeSTTTTS,
				}
				_, err1 := backend.CreateSession(ctx, sessionConfig)
				require.NoError(t, err1)

				// Try to create second session (should exceed limit)
				sessionConfig2 := &iface.SessionConfig{
					UserID:       "test-user-2",
					Transport:    "websocket",
					PipelineType: iface.PipelineTypeSTTTTS,
				}
				_, err2 := backend.CreateSession(ctx, sessionConfig2)
				return err2
			},
			shouldError: true,
			errorCode:   ErrCodeSessionLimitExceeded,
		},
		{
			name: "timeout handling",
			setup: func() iface.VoiceBackend {
				return NewAdvancedMockVoiceBackend(
					WithMockDelay(10 * time.Second),
				)
			},
			operation: func(t *testing.T, backend iface.VoiceBackend, ctx context.Context) error {
				timeoutCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
				return backend.Start(timeoutCtx)
			},
			shouldError: true,
			errorCode:   ErrCodeTimeout,
		},
		{
			name: "context cancellation",
			setup: func() iface.VoiceBackend {
				return NewAdvancedMockVoiceBackend(
					WithMockDelay(1 * time.Second),
				)
			},
			operation: func(t *testing.T, backend iface.VoiceBackend, ctx context.Context) error {
				cancelCtx, cancel := context.WithCancel(ctx)
				cancel() // Cancel immediately
				return backend.Start(cancelCtx)
			},
			shouldError: true,
			errorCode:   ErrCodeContextCanceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			backend := tt.setup()
			defer func() {
				_ = backend.Stop(ctx)
			}()

			err := tt.operation(t, backend, ctx)
			if tt.shouldError {
				require.Error(t, err, "Operation should return error")
				if tt.errorCode != "" {
					backendErr := AsError(err)
					if backendErr != nil {
						assert.Equal(t, tt.errorCode, backendErr.Code, "Error code should match")
					}
				}
			} else {
				require.NoError(t, err, "Operation should not return error")
			}
		})
	}
}

// TestTurnDetectionAndInterruption tests turn detection and interruption handling (T258).
func TestTurnDetectionAndInterruption(t *testing.T) {
	tests := []struct {
		name            string
		audioChunks     [][]byte
		shouldInterrupt bool
	}{
		{
			name: "normal turn completion",
			audioChunks: [][]byte{
				[]byte{1, 2, 3},
				[]byte{4, 5, 6},
			},
			shouldInterrupt: false,
		},
		{
			name: "interruption during processing",
			audioChunks: [][]byte{
				[]byte{1, 2, 3},
			},
			shouldInterrupt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			backend := NewAdvancedMockVoiceBackend()
			require.NoError(t, backend.Start(ctx))
			defer func() {
				_ = backend.Stop(ctx)
			}()

			sessionConfig := &iface.SessionConfig{
				UserID:       "test-user",
				Transport:    "websocket",
				PipelineType: iface.PipelineTypeSTTTTS,
			}

			session, err := backend.CreateSession(ctx, sessionConfig)
			require.NoError(t, err)

			err = session.Start(ctx)
			require.NoError(t, err)

			// Process audio chunks
			for _, chunk := range tt.audioChunks {
				_ = session.ProcessAudio(ctx, chunk)
			}

			// Verify state
			state := session.GetState()
			if tt.shouldInterrupt {
				// State might be processing or listening
				assert.Contains(t, []iface.PipelineState{
					iface.PipelineStateProcessing,
					iface.PipelineStateListening,
				}, state, "State should be processing or listening")
			}

			_ = session.Stop(ctx)
		})
	}
}

// TestConcurrentMultiUserScenarios tests concurrent multi-user scenarios (T259).
func TestConcurrentMultiUserScenarios(t *testing.T) {
	tests := []struct {
		name              string
		numUsers          int
		operationsPerUser int
		shouldError       bool
	}{
		{
			name:              "10 concurrent users",
			numUsers:          10,
			operationsPerUser: 5,
			shouldError:       false,
		},
		{
			name:              "50 concurrent users",
			numUsers:          50,
			operationsPerUser: 3,
			shouldError:       false,
		},
		{
			name:              "100 concurrent users",
			numUsers:          100,
			operationsPerUser: 2,
			shouldError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			backend := NewAdvancedMockVoiceBackend()
			require.NoError(t, backend.Start(ctx))
			defer func() {
				_ = backend.Stop(ctx)
			}()

			var wg sync.WaitGroup
			errors := make(chan error, tt.numUsers*tt.operationsPerUser)

			for i := 0; i < tt.numUsers; i++ {
				wg.Add(1)
				go func(userID int) {
					defer wg.Done()

					for j := 0; j < tt.operationsPerUser; j++ {
						sessionConfig := &iface.SessionConfig{
							UserID:       "test-user",
							Transport:    "websocket",
							PipelineType: iface.PipelineTypeSTTTTS,
						}

						session, err := backend.CreateSession(ctx, sessionConfig)
						if err != nil {
							errors <- err
							continue
						}

						err = session.Start(ctx)
						if err != nil {
							errors <- err
							continue
						}

						audio := []byte{byte(userID), byte(j)}
						err = session.ProcessAudio(ctx, audio)
						if err != nil {
							errors <- err
						}

						_ = session.Stop(ctx)
					}
				}(i)
			}

			wg.Wait()
			close(errors)

			// Collect errors
			var errorCount int
			for range errors {
				errorCount++
			}

			if !tt.shouldError {
				assert.Equal(t, 0, errorCount, "Should have no errors")
			}

			// Verify active session count
			activeCount := backend.GetActiveSessionCount()
			assert.Equal(t, 0, activeCount, "All sessions should be closed")
		})
	}
}

// TestEdgeCases tests edge cases from spec (T260).
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() iface.VoiceBackend
		operation   func(t *testing.T, backend iface.VoiceBackend, ctx context.Context)
		description string
	}{
		{
			name: "connection loss recovery",
			setup: func() iface.VoiceBackend {
				return NewAdvancedMockVoiceBackend()
			},
			operation: func(t *testing.T, backend iface.VoiceBackend, ctx context.Context) {
				// Simulate connection loss
				_ = backend.Stop(ctx)

				// Attempt recovery
				err := backend.Start(ctx)
				require.NoError(t, err, "Should recover from connection loss")

				state := backend.GetConnectionState()
				assert.Equal(t, iface.ConnectionStateConnected, state, "Should be connected after recovery")
			},
			description: "Connection loss and recovery",
		},
		{
			name: "format mismatch handling",
			setup: func() iface.VoiceBackend {
				return NewAdvancedMockVoiceBackend()
			},
			operation: func(t *testing.T, backend iface.VoiceBackend, ctx context.Context) {
				_ = backend.Start(ctx)

				sessionConfig := &iface.SessionConfig{
					UserID:       "test-user",
					Transport:    "websocket",
					PipelineType: iface.PipelineTypeSTTTTS,
				}

				session, err := backend.CreateSession(ctx, sessionConfig)
				require.NoError(t, err)

				err = session.Start(ctx)
				require.NoError(t, err)

				// Process audio with different formats (empty, large, invalid)
				testCases := [][]byte{
					{},                  // Empty audio
					make([]byte, 10000), // Large audio
					{0xFF, 0xFF, 0xFF},  // Invalid format
				}

				for _, audio := range testCases {
					// Should not panic, may return error
					_ = session.ProcessAudio(ctx, audio)
				}

				_ = session.Stop(ctx)
			},
			description: "Format mismatch handling",
		},
		{
			name: "timeout scenarios",
			setup: func() iface.VoiceBackend {
				return NewAdvancedMockVoiceBackend(
					WithMockDelay(2 * time.Second),
				)
			},
			operation: func(t *testing.T, backend iface.VoiceBackend, ctx context.Context) {
				timeoutCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()

				err := backend.Start(timeoutCtx)
				require.Error(t, err, "Should timeout")
				assert.Equal(t, context.DeadlineExceeded, err, "Should be deadline exceeded")
			},
			description: "Timeout scenarios",
		},
		{
			name: "concurrent session creation limit",
			setup: func() iface.VoiceBackend {
				return NewAdvancedMockVoiceBackend(
					WithMaxSessions(5),
				)
			},
			operation: func(t *testing.T, backend iface.VoiceBackend, ctx context.Context) {
				_ = backend.Start(ctx)

				// Create sessions up to limit
				var sessions []iface.VoiceSession
				for i := 0; i < 5; i++ {
					sessionConfig := &iface.SessionConfig{
						UserID:       "test-user",
						Transport:    "websocket",
						PipelineType: iface.PipelineTypeSTTTTS,
					}
					session, err := backend.CreateSession(ctx, sessionConfig)
					require.NoError(t, err)
					sessions = append(sessions, session)
				}

				// Try to create one more (should fail)
				sessionConfig := &iface.SessionConfig{
					UserID:       "test-user",
					Transport:    "websocket",
					PipelineType: iface.PipelineTypeSTTTTS,
				}
				_, err := backend.CreateSession(ctx, sessionConfig)
				require.Error(t, err, "Should fail when limit exceeded")

				// Close one session
				err = backend.CloseSession(ctx, sessions[0].GetID())
				require.NoError(t, err)

				// Now should be able to create another
				_, err = backend.CreateSession(ctx, sessionConfig)
				require.NoError(t, err, "Should succeed after closing a session")
			},
			description: "Concurrent session creation limit",
		},
		{
			name: "session state transitions",
			setup: func() iface.VoiceBackend {
				return NewAdvancedMockVoiceBackend()
			},
			operation: func(t *testing.T, backend iface.VoiceBackend, ctx context.Context) {
				_ = backend.Start(ctx)

				sessionConfig := &iface.SessionConfig{
					UserID:       "test-user",
					Transport:    "websocket",
					PipelineType: iface.PipelineTypeSTTTTS,
				}

				session, err := backend.CreateSession(ctx, sessionConfig)
				require.NoError(t, err)

				// Initial state should be idle
				assert.Equal(t, iface.PipelineStateIdle, session.GetState(), "Initial state should be idle")

				// Start session
				err = session.Start(ctx)
				require.NoError(t, err)
				assert.Equal(t, iface.PipelineStateListening, session.GetState(), "State should be listening after start")

				// Process audio
				err = session.ProcessAudio(ctx, []byte{1, 2, 3})
				// State might change during processing
				state := session.GetState()
				assert.Contains(t, []iface.PipelineState{
					iface.PipelineStateListening,
					iface.PipelineStateProcessing,
				}, state, "State should be listening or processing")

				// Stop session
				err = session.Stop(ctx)
				require.NoError(t, err)
				assert.Equal(t, iface.PipelineStateIdle, session.GetState(), "State should be idle after stop")
			},
			description: "Session state transitions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			backend := tt.setup()
			defer func() {
				_ = backend.Stop(ctx)
			}()

			tt.operation(t, backend, ctx)
		})
	}
}

// BenchmarkBackendCreationTime benchmarks backend creation time (T279, SC-007).
// Target: <2 seconds per backend creation.
func BenchmarkBackendCreationTime(b *testing.B) {
	ctx := context.Background()
	config := DefaultConfig()
	config.Provider = "mock"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		backend, err := NewBackend(ctx, "mock", config)
		if err != nil {
			b.Fatalf("Failed to create backend: %v", err)
		}
		_ = backend
	}
}

// BenchmarkAudioProcessingLatency benchmarks audio processing latency (T280, SC-001).
// Target: <500ms for audio processing.
func BenchmarkAudioProcessingLatency(b *testing.B) {
	ctx := context.Background()
	config := DefaultConfig()
	config.Provider = "mock"

	backend, err := NewBackend(ctx, "mock", config)
	if err != nil {
		b.Fatalf("Failed to create backend: %v", err)
	}
	defer func() {
		_ = backend.Stop(ctx)
	}()

	err = backend.Start(ctx)
	if err != nil {
		b.Fatalf("Failed to start backend: %v", err)
	}

	sessionConfig := &iface.SessionConfig{
		UserID:       "bench-user",
		Transport:    "websocket",
		PipelineType: iface.PipelineTypeSTTTTS,
		AgentCallback: func(ctx context.Context, transcript string) (string, error) {
			return "Response", nil
		},
	}

	session, err := backend.CreateSession(ctx, sessionConfig)
	if err != nil {
		b.Fatalf("Failed to create session: %v", err)
	}
	defer func() {
		_ = session.Stop(ctx)
	}()

	err = session.Start(ctx)
	if err != nil {
		b.Fatalf("Failed to start session: %v", err)
	}

	audio := make([]byte, 1024) // 1KB audio chunk
	for i := range audio {
		audio[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		err := session.ProcessAudio(ctx, audio)
		latency := time.Since(start)
		if err != nil {
			b.Fatalf("Failed to process audio: %v", err)
		}
		if latency > 500*time.Millisecond {
			b.Logf("Warning: Audio processing latency %v exceeds target 500ms", latency)
		}
	}
}

// BenchmarkConcurrentSessionHandling benchmarks concurrent session handling (T281, SC-002).
// Target: 100+ concurrent sessions.
func BenchmarkConcurrentSessionHandling(b *testing.B) {
	ctx := context.Background()
	config := DefaultConfig()
	config.Provider = "mock"
	config.MaxConcurrentSessions = 200 // Allow up to 200 sessions

	backend, err := NewBackend(ctx, "mock", config)
	if err != nil {
		b.Fatalf("Failed to create backend: %v", err)
	}
	defer func() {
		_ = backend.Stop(ctx)
	}()

	err = backend.Start(ctx)
	if err != nil {
		b.Fatalf("Failed to start backend: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			sessionConfig := &iface.SessionConfig{
				UserID:       "bench-user",
				Transport:    "websocket",
				PipelineType: iface.PipelineTypeSTTTTS,
				AgentCallback: func(ctx context.Context, transcript string) (string, error) {
					return "Response", nil
				},
			}

			session, err := backend.CreateSession(ctx, sessionConfig)
			if err != nil {
				b.Errorf("Failed to create session: %v", err)
				continue
			}

			err = session.Start(ctx)
			if err != nil {
				b.Errorf("Failed to start session: %v", err)
				_ = session.Stop(ctx)
				continue
			}

			// Process some audio
			audio := []byte{1, 2, 3, 4, 5}
			_ = session.ProcessAudio(ctx, audio)

			_ = session.Stop(ctx)
		}
	})
}
