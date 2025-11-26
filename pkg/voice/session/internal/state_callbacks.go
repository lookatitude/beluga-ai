package internal

import (
	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
)

// OnStateChanged sets a callback function that is called when the session state changes.
func (s *VoiceSessionImpl) OnStateChanged(callback func(sessioniface.SessionState)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stateChangeCallback = callback
}

// notifyStateChange notifies the state change callback (internal use).
func (s *VoiceSessionImpl) notifyStateChange(newState sessioniface.SessionState) {
	s.mu.RLock()
	callback := s.stateChangeCallback
	s.mu.RUnlock()

	if callback != nil {
		callback(newState)
	}
}
