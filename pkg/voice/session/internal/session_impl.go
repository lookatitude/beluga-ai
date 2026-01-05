package internal

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/voice/iface"
	sessioniface "github.com/lookatitude/beluga-ai/pkg/voice/session/iface"
	s2siface "github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
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
	S2SProvider       s2siface.S2SProvider
	VADProvider       iface.VADProvider
	TurnDetector      iface.TurnDetector
	Transport         iface.Transport
	NoiseCancellation iface.NoiseCancellation
	AgentCallback     func(ctx context.Context, transcript string) (string, error)
	OnStateChanged    func(state sessioniface.SessionState)
	Config            *Config
	// Agent instance fields (passed from session package)
	AgentInstance agentsiface.StreamingAgent
	AgentConfig   *schema.AgentConfig
}

// VoiceSessionImpl implements the VoiceSession interface.
type VoiceSessionImpl struct {
	turnDetector        iface.TurnDetector
	ttsProvider         iface.TTSProvider
	noiseCancellation   iface.NoiseCancellation
	vadProvider         iface.VADProvider
	sttProvider         iface.STTProvider
	s2sProvider         s2siface.S2SProvider
	s2sIntegration      *S2SIntegration
	transport           iface.Transport
	agentCallback       func(ctx context.Context, transcript string) (string, error)
	config              *Config
	opts                *VoiceOptions
	stateChangeCallback func(state sessioniface.SessionState)
	stateMachine        *StateMachine
	agentIntegration    *AgentIntegration
	streamingAgent      *StreamingAgent
	s2sAgentIntegration *S2SAgentIntegration // S2S integration with agent support
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
		s2sProvider:         opts.S2SProvider,
		vadProvider:         opts.VADProvider,
		turnDetector:        opts.TurnDetector,
		transport:           opts.Transport,
		noiseCancellation:   opts.NoiseCancellation,
		agentCallback:       opts.AgentCallback,
		stateChangeCallback: opts.OnStateChanged,
		stateMachine:        stateMachine,
	}

	// Initialize S2S integration if S2S provider is set
	if opts.S2SProvider != nil {
		impl.s2sIntegration = NewS2SIntegration(opts.S2SProvider)
	}

	// Initialize agent integration
	var agentIntegration *AgentIntegration
	if opts.AgentCallback != nil || opts.AgentInstance != nil {
		if opts.AgentInstance != nil {
			// Validate that AgentInstance implements StreamingAgent interface
			// This is a compile-time check, but we verify at runtime for safety
			if opts.AgentInstance == nil {
				return nil, errors.New("agent instance validation failed: AgentInstance cannot be nil")
			}

			// Create agent instance-based integration
			var agentConfig schema.AgentConfig
			if opts.AgentConfig != nil {
				agentConfig = *opts.AgentConfig
			} else {
				// Set default agent config if not provided
				agentConfig = schema.AgentConfig{
					Name: "voice-agent",
				}
			}

			// Create agent instance
			agentInstance := NewAgentInstance(opts.AgentInstance, agentConfig)

			// Create agent integration with instance
			agentIntegration = NewAgentIntegrationWithInstance(opts.AgentInstance, agentConfig)
			agentIntegration.SetAgentInstance(agentInstance)
			impl.agentIntegration = agentIntegration

			// Create streaming agent if TTS provider is available
			if opts.TTSProvider != nil {
				streamingConfig := DefaultStreamingAgentConfig()
				impl.streamingAgent = NewStreamingAgent(agentInstance, opts.TTSProvider, streamingConfig)
			}
		} else {
			// Create callback-based integration (legacy)
			agentIntegration = NewAgentIntegration(opts.AgentCallback)
			impl.agentIntegration = agentIntegration
		}
	}

	// Initialize S2S agent integration if both S2S provider and agent are present
	// This enables external reasoning mode for S2S
	if opts.S2SProvider != nil && agentIntegration != nil {
		// Determine reasoning mode from S2S config (default to external when agent is present)
		reasoningMode := "external" // Default to external when agent is provided
		// TODO: Extract reasoning mode from S2S provider config if available
		impl.s2sAgentIntegration = NewS2SAgentIntegration(opts.S2SProvider, agentIntegration, reasoningMode)
	}

	return impl, nil
}

// generateSessionID generates a unique session ID.
func generateSessionID() string {
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}
