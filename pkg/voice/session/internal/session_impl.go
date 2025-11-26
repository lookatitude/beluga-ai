package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
)

// Config and VoiceOptions are passed in, not imported to avoid cycle.
type Config struct {
	SessionID         string
	Timeout           time.Duration
	AutoStart         bool
	EnableKeepAlive   bool
	KeepAliveInterval time.Duration
	MaxRetries        int
	RetryDelay        time.Duration
}

type VoiceOptions struct {
	STTProvider       iface.STTProvider
	TTSProvider       iface.TTSProvider
	VADProvider       iface.VADProvider
	TurnDetector      iface.TurnDetector
	Transport         iface.Transport
	NoiseCancellation iface.NoiseCancellation
	AgentCallback     func(ctx context.Context, transcript string) (string, error)
	OnStateChanged    func(state sessioniface.SessionState)
	Config            *Config
}

// VoiceSessionImpl implements the VoiceSession interface.
type VoiceSessionImpl struct {
	turnDetector        iface.TurnDetector
	ttsProvider         iface.TTSProvider
	noiseCancellation   iface.NoiseCancellation
	vadProvider         iface.VADProvider
	sttProvider         iface.STTProvider
	transport           iface.Transport
	opts                *VoiceOptions
	config              *Config
	agentCallback       func(ctx context.Context, transcript string) (string, error)
	stateChangeCallback func(state sessioniface.SessionState)
	stateMachine        *StateMachine
	sessionID           string
	state               sessioniface.SessionState
	mu                  sync.RWMutex
	active              bool
}

// NewVoiceSessionImpl creates a new VoiceSessionImpl instance.
func NewVoiceSessionImpl(config *Config, opts *VoiceOptions) (*VoiceSessionImpl, error) {
	if config == nil {
		// Use defaults if config is nil
		config = &Config{
			Timeout:           30 * time.Minute,
			EnableKeepAlive:   true,
			KeepAliveInterval: 30 * time.Second,
			MaxRetries:        3,
			RetryDelay:        1 * time.Second,
		}
	}

	// Generate session ID if not provided
	sessionID := config.SessionID
	if sessionID == "" {
		sessionID = generateSessionID()
	}

	// Create state machine
	stateMachine := NewStateMachine()

	impl := &VoiceSessionImpl{
		config:              config,
		opts:                opts,
		sessionID:           sessionID,
		state:               sessioniface.SessionState("initial"),
		active:              false,
		sttProvider:         opts.STTProvider,
		ttsProvider:         opts.TTSProvider,
		vadProvider:         opts.VADProvider,
		turnDetector:        opts.TurnDetector,
		transport:           opts.Transport,
		noiseCancellation:   opts.NoiseCancellation,
		agentCallback:       opts.AgentCallback,
		stateChangeCallback: opts.OnStateChanged,
		stateMachine:        stateMachine,
	}

	return impl, nil
}

// generateSessionID generates a unique session ID.
func generateSessionID() string {
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}
