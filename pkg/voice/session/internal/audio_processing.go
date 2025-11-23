package internal

import (
	"context"
	"fmt"

	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
)

// ProcessAudio processes incoming audio data
func (s *VoiceSessionImpl) ProcessAudio(ctx context.Context, audio []byte) error {
	s.mu.RLock()
	active := s.active
	state := s.state
	s.mu.RUnlock()

	if !active {
		return newSessionError("ProcessAudio", "session_not_active",
			fmt.Errorf("session is not active"))
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
