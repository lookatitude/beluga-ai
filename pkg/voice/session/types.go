package session

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
)

// Re-export types from iface for convenience.
type (
	SessionState = iface.SessionState
	SayOptions   = iface.SayOptions
	SayHandle    = iface.SayHandle
)

// SessionState constants.
const (
	SessionStateInitial    = iface.SessionStateInitial
	SessionStateListening  = iface.SessionStateListening
	SessionStateProcessing = iface.SessionStateProcessing
	SessionStateSpeaking   = iface.SessionStateSpeaking
	SessionStateAway       = iface.SessionStateAway
	SessionStateEnded      = iface.SessionStateEnded
)

// VoiceOptions represents options for configuring a VoiceSession.
type VoiceOptions struct {
	// STTProvider specifies the STT provider to use
	STTProvider iface.STTProvider

	// TTSProvider specifies the TTS provider to use
	TTSProvider iface.TTSProvider

	// VADProvider specifies the VAD provider to use
	VADProvider iface.VADProvider

	// TurnDetector specifies the turn detection provider to use
	TurnDetector iface.TurnDetector

	// Transport specifies the transport provider to use
	Transport iface.Transport

	// NoiseCancellation specifies the noise cancellation provider to use
	NoiseCancellation iface.NoiseCancellation

	// AgentCallback is called when user input is detected
	AgentCallback func(ctx context.Context, transcript string) (string, error)

	// OnStateChanged callback for state changes
	OnStateChanged func(state SessionState)

	// Config for session configuration
	Config *Config
}

// VoiceOption is a functional option for configuring VoiceSession.
type VoiceOption func(*VoiceOptions)

// WithSTTProvider sets the STT provider.
func WithSTTProvider(provider iface.STTProvider) VoiceOption {
	return func(opts *VoiceOptions) {
		opts.STTProvider = provider
	}
}

// WithTTSProvider sets the TTS provider.
func WithTTSProvider(provider iface.TTSProvider) VoiceOption {
	return func(opts *VoiceOptions) {
		opts.TTSProvider = provider
	}
}

// WithVADProvider sets the VAD provider.
func WithVADProvider(provider iface.VADProvider) VoiceOption {
	return func(opts *VoiceOptions) {
		opts.VADProvider = provider
	}
}

// WithTurnDetector sets the turn detector.
func WithTurnDetector(detector iface.TurnDetector) VoiceOption {
	return func(opts *VoiceOptions) {
		opts.TurnDetector = detector
	}
}

// WithTransport sets the transport provider.
func WithTransport(transport iface.Transport) VoiceOption {
	return func(opts *VoiceOptions) {
		opts.Transport = transport
	}
}

// WithNoiseCancellation sets the noise cancellation provider.
func WithNoiseCancellation(noiseCancellation iface.NoiseCancellation) VoiceOption {
	return func(opts *VoiceOptions) {
		opts.NoiseCancellation = noiseCancellation
	}
}

// WithAgentCallback sets the agent callback.
func WithAgentCallback(callback func(ctx context.Context, transcript string) (string, error)) VoiceOption {
	return func(opts *VoiceOptions) {
		opts.AgentCallback = callback
	}
}

// WithOnStateChanged sets the state change callback.
func WithOnStateChanged(callback func(state SessionState)) VoiceOption {
	return func(opts *VoiceOptions) {
		opts.OnStateChanged = callback
	}
}

// WithConfig sets the session configuration.
func WithConfig(config *Config) VoiceOption {
	return func(opts *VoiceOptions) {
		opts.Config = config
	}
}
