package internal

import (
	"context"
	"errors"
	"sync"
	"time"

	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
)

// SayHandleImpl implements the SayHandle interface.
type SayHandleImpl struct {
	canceled bool
	mu       sync.RWMutex
}

// WaitForPlayout waits for audio to finish playing.
func (h *SayHandleImpl) WaitForPlayout(ctx context.Context) error {
	// TODO: Implement actual wait logic
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1 * time.Second): // Placeholder
		return nil
	}
}

// Cancel cancels the Say operation.
func (h *SayHandleImpl) Cancel() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.canceled = true
	return nil
}

// Say converts text to speech and plays it.
func (s *VoiceSessionImpl) Say(ctx context.Context, text string) (sessioniface.SayHandle, error) {
	return s.SayWithOptions(ctx, text, sessioniface.SayOptions{})
}

// SayWithOptions converts text to speech with options and plays it.
func (s *VoiceSessionImpl) SayWithOptions(ctx context.Context, text string, options sessioniface.SayOptions) (sessioniface.SayHandle, error) {
	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()

	if !active {
		return nil, newSessionError("SayWithOptions", "session_not_active",
			errors.New("session is not active"))
	}

	// Transition to speaking state
	s.mu.Lock()
	if !s.stateMachine.SetState(sessioniface.SessionState("speaking")) {
		s.mu.Unlock()
		return nil, newSessionError("SayWithOptions", "invalid_state",
			nil)
	}
	s.state = s.stateMachine.GetState()
	if s.stateChangeCallback != nil {
		s.stateChangeCallback(s.state)
	}
	s.mu.Unlock()

	// TODO: Implement actual TTS and playback
	// 1. Call TTS provider to synthesize speech
	// 2. Play audio through transport
	// 3. Handle interruptions if AllowInterruptions is true
	// 4. Transition back to listening state when done

	// Placeholder: Create handle
	handle := &SayHandleImpl{}

	// Transition back to listening state after "playback"
	go func() {
		// Simulate playback completion
		time.Sleep(100 * time.Millisecond)
		s.mu.Lock()
		s.stateMachine.SetState(sessioniface.SessionState("listening"))
		s.state = s.stateMachine.GetState()
		if s.stateChangeCallback != nil {
			s.stateChangeCallback(s.state)
		}
		s.mu.Unlock()
	}()

	return handle, nil
}
