package voice

import (
	"context"
	"fmt"
)

// PipelineMode identifies the active mode in a hybrid pipeline.
type PipelineMode string

const (
	// ModeS2S indicates the speech-to-speech (native audio) mode.
	ModeS2S PipelineMode = "s2s"

	// ModeCascade indicates the cascading STT → LLM → TTS mode.
	ModeCascade PipelineMode = "cascade"
)

// PipelineState holds runtime state used by SwitchPolicy to decide whether
// the hybrid pipeline should switch modes.
type PipelineState struct {
	// ToolCallCount is the number of tool calls in the current session.
	ToolCallCount int

	// CurrentMode is the currently active pipeline mode.
	CurrentMode PipelineMode

	// TurnCount is the number of conversational turns completed.
	TurnCount int
}

// SwitchPolicy determines when a hybrid pipeline should switch between
// S2S and cascade modes.
type SwitchPolicy interface {
	// ShouldSwitch returns true if the pipeline should switch to the
	// other mode given the current state.
	ShouldSwitch(ctx context.Context, state PipelineState) bool
}

// SwitchPolicyFunc is an adapter that turns a plain function into a SwitchPolicy.
type SwitchPolicyFunc func(ctx context.Context, state PipelineState) bool

// ShouldSwitch calls f(ctx, state).
func (f SwitchPolicyFunc) ShouldSwitch(ctx context.Context, state PipelineState) bool {
	return f(ctx, state)
}

// DefaultSwitchPolicy switches from S2S to cascade when tool call count
// exceeds a threshold. This handles the common case where S2S providers
// struggle with complex multi-tool interactions.
type DefaultSwitchPolicy struct {
	// ToolCallThreshold is the number of tool calls after which the pipeline
	// switches to cascade mode. Defaults to 3 if zero.
	ToolCallThreshold int
}

// ShouldSwitch returns true when the tool call count exceeds the threshold
// and the current mode is S2S.
func (p *DefaultSwitchPolicy) ShouldSwitch(_ context.Context, state PipelineState) bool {
	threshold := p.ToolCallThreshold
	if threshold == 0 {
		threshold = 3
	}
	return state.CurrentMode == ModeS2S && state.ToolCallCount >= threshold
}

// OnToolOverload is a convenience SwitchPolicy that switches to cascade mode
// when 3 or more tool calls have been made.
var OnToolOverload SwitchPolicy = &DefaultSwitchPolicy{ToolCallThreshold: 3}

// S2SProcessor is a local interface for speech-to-speech processors.
// Concrete implementations live in voice/s2s/.
type S2SProcessor interface {
	FrameProcessor
}

// HybridPipelineConfig holds configuration for a HybridPipeline.
type HybridPipelineConfig struct {
	S2S          S2SProcessor
	Cascade      *VoicePipeline
	SwitchPolicy SwitchPolicy
	Session      *VoiceSession
}

// HybridPipelineOption configures a HybridPipeline.
type HybridPipelineOption func(*HybridPipelineConfig)

// WithS2S sets the speech-to-speech processor for the hybrid pipeline.
func WithS2S(s2s S2SProcessor) HybridPipelineOption {
	return func(cfg *HybridPipelineConfig) {
		cfg.S2S = s2s
	}
}

// WithCascade sets the cascading pipeline as fallback for the hybrid pipeline.
func WithCascade(cascade *VoicePipeline) HybridPipelineOption {
	return func(cfg *HybridPipelineConfig) {
		cfg.Cascade = cascade
	}
}

// WithSwitchPolicy sets the switch policy for mode transitions.
func WithSwitchPolicy(policy SwitchPolicy) HybridPipelineOption {
	return func(cfg *HybridPipelineConfig) {
		cfg.SwitchPolicy = policy
	}
}

// WithHybridSession sets the voice session for the hybrid pipeline.
func WithHybridSession(s *VoiceSession) HybridPipelineOption {
	return func(cfg *HybridPipelineConfig) {
		cfg.Session = s
	}
}

// HybridPipeline combines S2S and cascade pipelines, switching between them
// based on a configurable policy. By default it uses S2S for regular
// conversation and falls back to cascade for tool-heavy interactions.
type HybridPipeline struct {
	config HybridPipelineConfig
	state  PipelineState
}

// NewHybridPipeline creates a new HybridPipeline with the given options.
func NewHybridPipeline(opts ...HybridPipelineOption) *HybridPipeline {
	cfg := HybridPipelineConfig{
		SwitchPolicy: OnToolOverload,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &HybridPipeline{
		config: cfg,
		state: PipelineState{
			CurrentMode: ModeS2S,
		},
	}
}

// Run starts the hybrid pipeline. It begins in S2S mode and switches to
// cascade mode when the switch policy indicates. This is a stub
// implementation that will be expanded when S2S providers are implemented.
func (h *HybridPipeline) Run(ctx context.Context) error {
	if h.config.S2S == nil && h.config.Cascade == nil {
		return fmt.Errorf("voice: hybrid pipeline requires at least one of S2S or cascade")
	}

	// Check if we should switch modes.
	if h.config.SwitchPolicy != nil && h.config.SwitchPolicy.ShouldSwitch(ctx, h.state) {
		h.state.CurrentMode = ModeCascade
	}

	switch h.state.CurrentMode {
	case ModeS2S:
		if h.config.S2S == nil {
			// Fall back to cascade if S2S is not configured.
			h.state.CurrentMode = ModeCascade
			return h.runCascade(ctx)
		}
		return h.runS2S(ctx)
	case ModeCascade:
		return h.runCascade(ctx)
	default:
		return fmt.Errorf("voice: unknown pipeline mode %q", h.state.CurrentMode)
	}
}

// CurrentMode returns the currently active pipeline mode.
func (h *HybridPipeline) CurrentMode() PipelineMode {
	return h.state.CurrentMode
}

// UpdateState updates the pipeline state, allowing external code to inform
// the hybrid pipeline about tool calls and turn counts.
func (h *HybridPipeline) UpdateState(toolCalls, turnCount int) {
	h.state.ToolCallCount = toolCalls
	h.state.TurnCount = turnCount
}

// runS2S runs the S2S processor. The S2S processor handles its own
// bidirectional audio transport (WebRTC/WebSocket) and doesn't use the
// cascade transport system.
func (h *HybridPipeline) runS2S(ctx context.Context) error {
	if h.config.Session == nil {
		return fmt.Errorf("voice: S2S pipeline requires a session")
	}

	// S2S processors are self-contained FrameProcessors that manage their
	// own transport. Create dummy channels since S2S doesn't use the
	// cascade transport pattern.
	in := make(chan Frame)
	out := make(chan Frame)

	// Close input immediately - S2S manages its own audio I/O.
	close(in)

	// Drain output in case the processor produces any frames.
	go func() {
		for range out {
			// Discard - S2S handles its own output transport
		}
	}()

	// Run the S2S processor. It will manage its own WebSocket/WebRTC
	// connection internally and handle audio I/O.
	return h.config.S2S.Process(ctx, in, out)
}

// runCascade delegates to the cascade pipeline.
func (h *HybridPipeline) runCascade(ctx context.Context) error {
	if h.config.Cascade == nil {
		return fmt.Errorf("voice: cascade pipeline not configured")
	}
	return h.config.Cascade.Run(ctx)
}
