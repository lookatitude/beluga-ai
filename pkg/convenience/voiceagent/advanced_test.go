package voiceagent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// TestVoiceAgent_ProcessAudio tests audio processing.
func TestVoiceAgent_ProcessAudio(t *testing.T) {
	tests := []struct {
		name        string
		setupAgent  func() (VoiceAgent, error)
		audio       []byte
		expectError bool
	}{
		{
			name: "basic audio processing",
			setupAgent: func() (VoiceAgent, error) {
				mockSTT := NewMockSTT()
				mockSTT.SetTranscription("Hello world")
				mockTTS := NewMockTTS()
				mockTTS.SetGenerateResponse([]byte("audio-response"))

				return NewBuilder().
					WithSTTInstance(mockSTT).
					WithTTSInstance(mockTTS).
					Build(context.Background())
			},
			audio:       []byte("test audio data"),
			expectError: false,
		},
		{
			name: "with LLM processing",
			setupAgent: func() (VoiceAgent, error) {
				mockSTT := NewMockSTT()
				mockSTT.SetTranscription("What is AI?")
				mockTTS := NewMockTTS()
				mockLLM := NewMockChatModel()
				mockLLM.SetGenerateResponse("AI stands for Artificial Intelligence.")

				return NewBuilder().
					WithSTTInstance(mockSTT).
					WithTTSInstance(mockTTS).
					WithLLMInstance(mockLLM).
					Build(context.Background())
			},
			audio:       []byte("test audio"),
			expectError: false,
		},
		{
			name: "STT error",
			setupAgent: func() (VoiceAgent, error) {
				mockSTT := NewMockSTT()
				mockSTT.TranscribeFunc = func(_ context.Context, _ []byte) (string, error) {
					return "", errors.New("transcription failed")
				}
				mockTTS := NewMockTTS()

				return NewBuilder().
					WithSTTInstance(mockSTT).
					WithTTSInstance(mockTTS).
					Build(context.Background())
			},
			audio:       []byte("test audio"),
			expectError: true,
		},
		{
			name: "TTS error",
			setupAgent: func() (VoiceAgent, error) {
				mockSTT := NewMockSTT()
				mockTTS := NewMockTTS()
				mockTTS.GenerateSpeechFunc = func(_ context.Context, _ string) ([]byte, error) {
					return nil, errors.New("synthesis failed")
				}

				return NewBuilder().
					WithSTTInstance(mockSTT).
					WithTTSInstance(mockTTS).
					Build(context.Background())
			},
			audio:       []byte("test audio"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := tt.setupAgent()
			if err != nil {
				t.Fatalf("failed to setup agent: %v", err)
			}
			defer agent.Shutdown()

			result, err := agent.ProcessAudio(context.Background(), tt.audio)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result) == 0 {
				t.Error("expected non-empty audio response")
			}
		})
	}
}

// TestVoiceAgent_ProcessText tests text processing.
func TestVoiceAgent_ProcessText(t *testing.T) {
	tests := []struct {
		name        string
		setupAgent  func() (VoiceAgent, error)
		text        string
		expectError bool
	}{
		{
			name: "echo without LLM",
			setupAgent: func() (VoiceAgent, error) {
				return NewBuilder().
					WithSTTInstance(NewMockSTT()).
					WithTTSInstance(NewMockTTS()).
					Build(context.Background())
			},
			text:        "Hello world",
			expectError: false,
		},
		{
			name: "with LLM processing",
			setupAgent: func() (VoiceAgent, error) {
				mockLLM := NewMockChatModel()
				mockLLM.SetGenerateResponse("This is the AI response.")

				return NewBuilder().
					WithSTTInstance(NewMockSTT()).
					WithTTSInstance(NewMockTTS()).
					WithLLMInstance(mockLLM).
					Build(context.Background())
			},
			text:        "What is the weather?",
			expectError: false,
		},
		{
			name: "LLM error",
			setupAgent: func() (VoiceAgent, error) {
				mockLLM := NewMockChatModel()
				mockLLM.GenerateFunc = func(_ context.Context, _ []schema.Message, _ ...core.Option) (schema.Message, error) {
					return nil, errors.New("LLM failed")
				}

				return NewBuilder().
					WithSTTInstance(NewMockSTT()).
					WithTTSInstance(NewMockTTS()).
					WithLLMInstance(mockLLM).
					Build(context.Background())
			},
			text:        "Hello",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent, err := tt.setupAgent()
			if err != nil {
				t.Fatalf("failed to setup agent: %v", err)
			}
			defer agent.Shutdown()

			result, err := agent.ProcessText(context.Background(), tt.text)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == "" {
				t.Error("expected non-empty response")
			}
		})
	}
}

// TestVoiceAgent_WithMemory tests memory integration.
func TestVoiceAgent_WithMemory(t *testing.T) {
	mockSTT := NewMockSTT()
	mockTTS := NewMockTTS()
	mockLLM := NewMockChatModel()
	mockMemory := NewMockMemory()

	agent, err := NewBuilder().
		WithSTTInstance(mockSTT).
		WithTTSInstance(mockTTS).
		WithLLMInstance(mockLLM).
		WithMemoryInstance(mockMemory).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build agent: %v", err)
	}
	defer agent.Shutdown()

	// Process first message
	_, err = agent.ProcessText(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("first ProcessText failed: %v", err)
	}

	// Process second message
	_, err = agent.ProcessText(context.Background(), "How are you?")
	if err != nil {
		t.Fatalf("second ProcessText failed: %v", err)
	}

	// Verify memory was used
	if mockMemory.SaveCalls < 2 {
		t.Errorf("expected at least 2 SaveContext calls, got %d", mockMemory.SaveCalls)
	}

	// Verify history is accumulated
	if len(mockMemory.History) < 4 {
		t.Errorf("expected at least 4 messages in history, got %d", len(mockMemory.History))
	}
}

// TestVoiceAgent_Callbacks tests callback invocations.
func TestVoiceAgent_Callbacks(t *testing.T) {
	var (
		transcriptCalled bool
		transcriptText   string
		responseCalled   bool
		responseText     string
		errorCalled      bool
	)

	mockSTT := NewMockSTT()
	mockSTT.SetTranscription("Test transcript")
	mockTTS := NewMockTTS()
	mockLLM := NewMockChatModel()
	mockLLM.SetGenerateResponse("Test response")

	agent, err := NewBuilder().
		WithSTTInstance(mockSTT).
		WithTTSInstance(mockTTS).
		WithLLMInstance(mockLLM).
		WithOnTranscript(func(text string, isFinal bool) {
			transcriptCalled = true
			transcriptText = text
		}).
		WithOnResponse(func(text string) {
			responseCalled = true
			responseText = text
		}).
		WithOnError(func(err error) {
			errorCalled = true
		}).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build agent: %v", err)
	}
	defer agent.Shutdown()

	// Process audio
	_, err = agent.ProcessAudio(context.Background(), []byte("audio"))
	if err != nil {
		t.Fatalf("ProcessAudio failed: %v", err)
	}

	if !transcriptCalled {
		t.Error("expected transcript callback to be called")
	}
	if transcriptText != "Test transcript" {
		t.Errorf("expected transcript 'Test transcript', got %s", transcriptText)
	}
	if !responseCalled {
		t.Error("expected response callback to be called")
	}
	if responseText != "Test response" {
		t.Errorf("expected response 'Test response', got %s", responseText)
	}
	if errorCalled {
		t.Error("expected error callback not to be called")
	}
}

// TestVoiceAgent_Session tests session management.
func TestVoiceAgent_Session(t *testing.T) {
	mockSTT := NewMockSTT()
	mockSTT.SetTranscription("Hello")
	mockTTS := NewMockTTS()
	mockTTS.SetGenerateResponse([]byte("audio"))

	agent, err := NewBuilder().
		WithSTTInstance(mockSTT).
		WithTTSInstance(mockTTS).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build agent: %v", err)
	}
	defer agent.Shutdown()

	// Start session
	session, err := agent.StartSession(context.Background())
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	// Verify session properties
	if session.ID() == "" {
		t.Error("expected non-empty session ID")
	}
	if !session.IsActive() {
		t.Error("expected session to be active")
	}

	// Send audio
	err = session.SendAudio(context.Background(), []byte("test audio"))
	if err != nil {
		t.Fatalf("SendAudio failed: %v", err)
	}

	// Give time for processing
	time.Sleep(100 * time.Millisecond)

	// Stop session
	err = session.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if session.IsActive() {
		t.Error("expected session to be inactive after stop")
	}
}

// TestVoiceAgent_SessionTranscript tests transcript accumulation.
func TestVoiceAgent_SessionTranscript(t *testing.T) {
	transcriptions := []string{"Hello", "How are you", "Goodbye"}
	transcriptIndex := 0
	var mu sync.Mutex

	mockSTT := NewMockSTT()
	mockSTT.TranscribeFunc = func(_ context.Context, _ []byte) (string, error) {
		mu.Lock()
		defer mu.Unlock()
		if transcriptIndex >= len(transcriptions) {
			return "", errors.New("no more transcriptions")
		}
		text := transcriptions[transcriptIndex]
		transcriptIndex++
		return text, nil
	}
	mockTTS := NewMockTTS()

	agent, err := NewBuilder().
		WithSTTInstance(mockSTT).
		WithTTSInstance(mockTTS).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build agent: %v", err)
	}
	defer agent.Shutdown()

	session, err := agent.StartSession(context.Background())
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	// Send multiple audio chunks
	for i := 0; i < len(transcriptions); i++ {
		err = session.SendAudio(context.Background(), []byte(fmt.Sprintf("audio %d", i)))
		if err != nil {
			t.Fatalf("SendAudio failed: %v", err)
		}
		time.Sleep(50 * time.Millisecond) // Allow processing
	}

	// Wait for processing to complete
	time.Sleep(200 * time.Millisecond)

	transcript := session.GetTranscript()
	if transcript == "" {
		t.Error("expected non-empty transcript")
	}

	session.Stop()
}

// TestVoiceAgent_VADIntegration tests VAD filtering.
func TestVoiceAgent_VADIntegration(t *testing.T) {
	mockSTT := NewMockSTT()
	mockTTS := NewMockTTS()
	mockVAD := NewMockVAD()
	mockVAD.SetVoiceDetection(false) // No voice detected

	agent, err := NewBuilder().
		WithSTTInstance(mockSTT).
		WithTTSInstance(mockTTS).
		WithVADInstance(mockVAD).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build agent: %v", err)
	}
	defer agent.Shutdown()

	session, err := agent.StartSession(context.Background())
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	// Send audio (should be filtered by VAD)
	err = session.SendAudio(context.Background(), []byte("silence"))
	if err != nil {
		t.Fatalf("SendAudio failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// STT should not be called when VAD detects no voice
	if mockSTT.TranscribeCalls > 0 {
		t.Errorf("expected no transcribe calls when VAD detects no voice, got %d", mockSTT.TranscribeCalls)
	}

	session.Stop()
}

// TestVoiceAgent_Concurrent tests concurrent operations.
func TestVoiceAgent_Concurrent(t *testing.T) {
	mockSTT := NewMockSTT()
	mockTTS := NewMockTTS()
	mockLLM := NewMockChatModel()

	agent, err := NewBuilder().
		WithSTTInstance(mockSTT).
		WithTTSInstance(mockTTS).
		WithLLMInstance(mockLLM).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build agent: %v", err)
	}
	defer agent.Shutdown()

	const numGoroutines = 10
	var wg sync.WaitGroup
	errs := make(chan error, numGoroutines*2)

	// Concurrent ProcessText calls
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := agent.ProcessText(context.Background(), fmt.Sprintf("message %d", id))
			if err != nil {
				errs <- err
			}
		}(i)
	}

	// Concurrent ProcessAudio calls
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := agent.ProcessAudio(context.Background(), []byte(fmt.Sprintf("audio %d", id)))
			if err != nil {
				errs <- err
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent operation failed: %v", err)
	}
}

// TestVoiceAgent_MultipleSessions tests multiple concurrent sessions.
func TestVoiceAgent_MultipleSessions(t *testing.T) {
	mockSTT := NewMockSTT()
	mockTTS := NewMockTTS()

	agent, err := NewBuilder().
		WithSTTInstance(mockSTT).
		WithTTSInstance(mockTTS).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build agent: %v", err)
	}
	defer agent.Shutdown()

	const numSessions = 5
	sessions := make([]Session, numSessions)

	// Start multiple sessions
	for i := 0; i < numSessions; i++ {
		session, err := agent.StartSession(context.Background())
		if err != nil {
			t.Fatalf("StartSession %d failed: %v", i, err)
		}
		sessions[i] = session
	}

	// Verify all sessions are active with unique IDs
	ids := make(map[string]bool)
	for i, session := range sessions {
		if !session.IsActive() {
			t.Errorf("session %d is not active", i)
		}
		if ids[session.ID()] {
			t.Errorf("duplicate session ID: %s", session.ID())
		}
		ids[session.ID()] = true
	}

	// Stop all sessions
	for i, session := range sessions {
		err := session.Stop()
		if err != nil {
			t.Errorf("Stop session %d failed: %v", i, err)
		}
	}
}

// TestVoiceAgent_Shutdown tests graceful shutdown.
func TestVoiceAgent_Shutdown(t *testing.T) {
	mockSTT := NewMockSTT()
	mockTTS := NewMockTTS()

	agent, err := NewBuilder().
		WithSTTInstance(mockSTT).
		WithTTSInstance(mockTTS).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build agent: %v", err)
	}

	// Start some sessions
	sessions := make([]Session, 3)
	for i := 0; i < 3; i++ {
		session, err := agent.StartSession(context.Background())
		if err != nil {
			t.Fatalf("StartSession failed: %v", err)
		}
		sessions[i] = session
	}

	// Shutdown agent
	err = agent.Shutdown()
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Verify sessions are stopped
	for i, session := range sessions {
		if session.IsActive() {
			t.Errorf("session %d should be inactive after shutdown", i)
		}
	}
}

// TestSession_SendAudioAfterStop tests sending audio to a stopped session.
func TestSession_SendAudioAfterStop(t *testing.T) {
	mockSTT := NewMockSTT()
	mockTTS := NewMockTTS()

	agent, err := NewBuilder().
		WithSTTInstance(mockSTT).
		WithTTSInstance(mockTTS).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build agent: %v", err)
	}
	defer agent.Shutdown()

	session, err := agent.StartSession(context.Background())
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	// Stop session
	session.Stop()

	// Try to send audio
	err = session.SendAudio(context.Background(), []byte("audio"))
	if err == nil {
		t.Error("expected error when sending audio to stopped session")
	}
}

// TestVoiceAgent_Timeout tests operation timeout.
func TestVoiceAgent_Timeout(t *testing.T) {
	mockSTT := NewMockSTT()
	mockSTT.TranscribeFunc = func(ctx context.Context, _ []byte) (string, error) {
		// Simulate slow transcription
		select {
		case <-time.After(5 * time.Second):
			return "transcription", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	mockTTS := NewMockTTS()

	agent, err := NewBuilder().
		WithSTTInstance(mockSTT).
		WithTTSInstance(mockTTS).
		WithTimeout(100 * time.Millisecond).
		Build(context.Background())
	if err != nil {
		t.Fatalf("failed to build agent: %v", err)
	}
	defer agent.Shutdown()

	// This should timeout
	_, err = agent.ProcessAudio(context.Background(), []byte("audio"))
	if err == nil {
		t.Error("expected timeout error")
	}
}
