package internal

import (
	"context"
	"errors"

	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
)

// ProcessAudio processes incoming audio data.
func (s *VoiceSessionImpl) ProcessAudio(ctx context.Context, audio []byte) error {
	s.mu.RLock()
	active := s.active
	state := s.state
	s.mu.RUnlock()

	if !active {
		return newSessionError("ProcessAudio", "session_not_active",
			errors.New("session is not active"))
	}

	// Apply noise cancellation if available
	if s.noiseCancellation != nil {
		cleaned, err := s.noiseCancellation.Process(ctx, audio)
		if err != nil {
			// Log error but continue processing
			_ = err
		} else {
			audio = cleaned
		}
	}

	// Route to S2S provider if available, otherwise use STT+TTS pipeline
	s.mu.RLock()
	s2sIntegration := s.s2sIntegration
	s.mu.RUnlock()

	if s2sIntegration != nil {
		// S2S mode: Process audio directly through S2S provider
		return s.processAudioWithS2S(ctx, audio)
	}

	// Traditional STT+TTS mode
	// TODO: Implement actual audio processing pipeline:
	// 1. Apply VAD to detect speech
	// 2. Use turn detection to identify user turns
	// 3. Send audio to STT provider for transcription
	// 4. Process transcript through agent callback
	// 5. Generate response via TTS
	// 6. Play response audio

	// Placeholder: Transition to processing state if not already
	if state == sessioniface.SessionState("listening") {
		s.mu.Lock()
		s.stateMachine.SetState(sessioniface.SessionState("processing"))
		s.state = s.stateMachine.GetState()
		if s.stateChangeCallback != nil {
			s.stateChangeCallback(s.state)
		}
		s.mu.Unlock()
	}

	return nil
}

// processAudioWithS2S processes audio using the S2S provider.
// If external reasoning mode is enabled and agent integration is available,
// audio is routed through the agent. Otherwise, it uses built-in reasoning.
func (s *VoiceSessionImpl) processAudioWithS2S(ctx context.Context, audio []byte) error {
	s.mu.RLock()
	s2sAgentIntegration := s.s2sAgentIntegration
	s2sIntegration := s.s2sIntegration
	sessionID := s.sessionID
	s.mu.RUnlock()

	var output []byte
	var err error

	// Use S2S agent integration if available (external reasoning mode)
	if s2sAgentIntegration != nil {
		output, err = s2sAgentIntegration.ProcessAudioWithAgent(ctx, audio, sessionID)
	} else {
		// Use direct S2S integration (built-in reasoning mode)
		output, err = s2sIntegration.ProcessAudioWithSessionID(ctx, audio, sessionID)
	}

	if err != nil {
		return newSessionError("ProcessAudio", "s2s_error",
			errors.New("failed to process audio with S2S provider"))
	}

	// Send output audio through transport if available
	s.mu.RLock()
	transport := s.transport
	s.mu.RUnlock()

	if transport != nil && output != nil && len(output) > 0 {
		if err := transport.SendAudio(ctx, output); err != nil {
			return newSessionError("ProcessAudio", "transport_error",
				errors.New("failed to send audio through transport"))
		}
	}

	// Transition to processing state
	s.mu.Lock()
	if !s.stateMachine.SetState(sessioniface.SessionState("processing")) {
		s.mu.Unlock()
		return newSessionError("ProcessAudio", "invalid_state", nil)
	}
	s.state = s.stateMachine.GetState()
	if s.stateChangeCallback != nil {
		s.stateChangeCallback(s.state)
	}
	s.mu.Unlock()

	// Transition back to listening state after processing
	s.mu.Lock()
	s.stateMachine.SetState(sessioniface.SessionState("listening"))
	s.state = s.stateMachine.GetState()
	if s.stateChangeCallback != nil {
		s.stateChangeCallback(s.state)
	}
	s.mu.Unlock()

	return nil
}
