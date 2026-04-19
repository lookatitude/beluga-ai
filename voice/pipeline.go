package voice

import (
	"context"
	"iter"

	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/internal/hookutil"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// Transcriber is a local interface for speech-to-text processors.
// This avoids importing the voice/stt package directly.
type Transcriber interface {
	FrameProcessor
}

// Synthesizer is a local interface for text-to-speech processors.
// This avoids importing the voice/tts package directly.
type Synthesizer interface {
	FrameProcessor
}

// LLMProcessor is a local interface for LLM integration in the voice pipeline.
// It wraps an LLM as a FrameProcessor that converts text frames to text frames.
type LLMProcessor interface {
	FrameProcessor
}

// Transport is a local interface for audio input/output transport.
// Concrete implementations live in voice/transport/.
type Transport interface {
	// Recv returns an iterator of incoming audio frames from the client.
	// Transport-level errors are delivered via the iterator's second element;
	// a non-nil error terminates the stream.
	Recv(ctx context.Context) iter.Seq2[Frame, error]

	// Send writes an outgoing audio frame to the client.
	Send(ctx context.Context, frame Frame) error

	// Close shuts down the transport connection.
	Close() error
}

// Hooks provides optional callback functions invoked at various points during
// pipeline execution. All fields are optional; nil hooks are skipped.
type Hooks struct {
	// OnSpeechStart is called when VAD detects the start of speech.
	OnSpeechStart func(ctx context.Context)

	// OnSpeechEnd is called when VAD detects the end of speech.
	OnSpeechEnd func(ctx context.Context)

	// OnTranscript is called with the transcribed text from STT.
	OnTranscript func(ctx context.Context, text string)

	// OnResponse is called with the LLM-generated response text.
	OnResponse func(ctx context.Context, text string)

	// OnError is called when a pipeline error occurs. Returning a non-nil
	// error propagates it; returning nil suppresses the error.
	OnError func(ctx context.Context, err error) error
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
// Callbacks are called in the order the hooks were provided.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		OnSpeechStart: hookutil.ComposeVoid0(h, func(hk Hooks) func(context.Context) {
			return hk.OnSpeechStart
		}),
		OnSpeechEnd: hookutil.ComposeVoid0(h, func(hk Hooks) func(context.Context) {
			return hk.OnSpeechEnd
		}),
		OnTranscript: hookutil.ComposeVoid1(h, func(hk Hooks) func(context.Context, string) {
			return hk.OnTranscript
		}),
		OnResponse: hookutil.ComposeVoid1(h, func(hk Hooks) func(context.Context, string) {
			return hk.OnResponse
		}),
		OnError: hookutil.ComposeErrorPassthrough(h, func(hk Hooks) func(context.Context, error) error {
			return hk.OnError
		}),
	}
}

// PipelineConfig holds the configuration for a VoicePipeline.
type PipelineConfig struct {
	Transport Transport
	VAD       ActivityDetector
	STT       Transcriber
	LLM       LLMProcessor
	TTS       Synthesizer
	Hooks     Hooks
	Session   *VoiceSession

	// ChannelBufferSize is retained for backward compatibility with callers
	// that previously configured inter-processor channel buffer sizes. The
	// iter.Seq2-based pipeline does not use intermediate channels, so this
	// field has no runtime effect.
	ChannelBufferSize int
}

// PipelineOption configures a VoicePipeline.
type PipelineOption func(*PipelineConfig)

// WithTransport sets the audio transport for the pipeline.
func WithTransport(t Transport) PipelineOption {
	return func(cfg *PipelineConfig) {
		cfg.Transport = t
	}
}

// WithVAD sets the voice activity detector for the pipeline.
func WithVAD(v ActivityDetector) PipelineOption {
	return func(cfg *PipelineConfig) {
		cfg.VAD = v
	}
}

// WithSTT sets the speech-to-text processor for the pipeline.
func WithSTT(stt Transcriber) PipelineOption {
	return func(cfg *PipelineConfig) {
		cfg.STT = stt
	}
}

// WithLLM sets the LLM processor for the pipeline.
func WithLLM(llm LLMProcessor) PipelineOption {
	return func(cfg *PipelineConfig) {
		cfg.LLM = llm
	}
}

// WithTTS sets the text-to-speech processor for the pipeline.
func WithTTS(tts Synthesizer) PipelineOption {
	return func(cfg *PipelineConfig) {
		cfg.TTS = tts
	}
}

// WithHooks sets the pipeline hooks.
func WithHooks(h Hooks) PipelineOption {
	return func(cfg *PipelineConfig) {
		cfg.Hooks = h
	}
}

// WithSession sets the voice session for state tracking.
func WithSession(s *VoiceSession) PipelineOption {
	return func(cfg *PipelineConfig) {
		cfg.Session = s
	}
}

// WithChannelBufferSize is retained for backward compatibility; it has no
// effect on the iter.Seq2-based pipeline.
func WithChannelBufferSize(size int) PipelineOption {
	return func(cfg *PipelineConfig) {
		cfg.ChannelBufferSize = size
	}
}

// VoicePipeline implements the cascading voice pipeline: STT → LLM → TTS.
// Each stage is a FrameProcessor composed lazily over iter.Seq2 streams.
type VoicePipeline struct {
	config PipelineConfig
}

// NewPipeline creates a new VoicePipeline with the given options.
func NewPipeline(opts ...PipelineOption) *VoicePipeline {
	cfg := PipelineConfig{
		ChannelBufferSize: 64,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &VoicePipeline{config: cfg}
}

// Run starts the cascading pipeline and blocks until ctx is cancelled or an
// error occurs. The pipeline connects Transport → VAD → STT → LLM → TTS →
// Transport by composing FrameProcessors over an iter.Seq2 stream.
func (p *VoicePipeline) Run(ctx context.Context) error {
	if p.config.Transport == nil {
		return core.Errorf(core.ErrInvalidInput, "voice: pipeline requires a transport")
	}

	// Build the processor chain from available components.
	var processors []FrameProcessor

	if p.config.VAD != nil {
		processors = append(processors, p.vadProcessor())
	}
	if p.config.STT != nil {
		processors = append(processors, p.config.STT)
	}
	if p.config.LLM != nil {
		processors = append(processors, p.config.LLM)
	}
	if p.config.TTS != nil {
		processors = append(processors, p.config.TTS)
	}

	if len(processors) == 0 {
		return core.Errorf(core.ErrInvalidInput, "voice: pipeline has no processors")
	}

	// Receive audio frames from transport as an iter.Seq2 stream. Any
	// transport-level dial failure is delivered as the first yielded pair.
	incoming := p.config.Transport.Recv(ctx)

	// Compose the pipeline lazily over the input stream.
	chain := Chain(processors...)
	output := chain.Process(ctx, incoming)

	// Drain the composed stream, forwarding frames through transport.
	for frame, err := range output {
		if err != nil {
			// Distinguish transport dial errors from downstream failures by
			// wrapping untyped errors with ErrProviderDown only when they
			// arrived on the first iteration with a zero frame (common case
			// for Transport.Recv early failure). Downstream stages already
			// return typed core errors, so pass those through unchanged.
			return err
		}
		if sendErr := p.config.Transport.Send(ctx, frame); sendErr != nil {
			return core.Errorf(core.ErrProviderDown, "voice: transport send: %w", sendErr)
		}
	}

	return ctx.Err()
}

// processVADResult emits control frames for VAD state transitions and any
// speech audio frame to the output slice.
func (p *VoicePipeline) processVADResult(ctx context.Context, result ActivityResult, frame Frame) []Frame {
	var out []Frame
	switch result.EventType {
	case VADSpeechStart:
		if p.config.Hooks.OnSpeechStart != nil {
			p.config.Hooks.OnSpeechStart(ctx)
		}
		out = append(out, NewControlFrame(SignalStart))
	case VADSpeechEnd:
		if p.config.Hooks.OnSpeechEnd != nil {
			p.config.Hooks.OnSpeechEnd(ctx)
		}
		out = append(out, NewControlFrame(SignalEndOfUtterance))
	}
	if result.IsSpeech {
		out = append(out, frame)
	}
	return out
}

// handleVADFrame processes a single audio frame through VAD and returns any
// resulting output frames. A non-nil error stops the processor; hook errors
// are the only errors propagated as fatal (VAD provider errors are reported
// to the OnError hook and otherwise suppressed).
func (p *VoicePipeline) handleVADFrame(ctx context.Context, frame Frame) ([]Frame, error) {
	if frame.Type != FrameAudio {
		return []Frame{frame}, nil
	}

	result, err := p.config.VAD.DetectActivity(ctx, frame.Data)
	if err != nil {
		if p.config.Hooks.OnError != nil {
			if hookErr := p.config.Hooks.OnError(ctx, err); hookErr != nil {
				return nil, hookErr
			}
		}
		return nil, nil
	}

	return p.processVADResult(ctx, result, frame), nil
}

// vadProcessor creates a FrameProcessor that runs VAD on audio frames and
// injects control frames for speech start/end events.
func (p *VoicePipeline) vadProcessor() FrameProcessor {
	return FrameLoop(func(ctx context.Context, frame Frame) ([]Frame, error) {
		return p.handleVADFrame(ctx, frame)
	})
}

// Events returns an iterator of pipeline events for streaming consumers.
// This is a convenience wrapper that collects events from hooks.
func (p *VoicePipeline) Events(ctx context.Context) iter.Seq2[schema.AgentEvent, error] {
	return func(yield func(schema.AgentEvent, error) bool) {
		// Run pipeline; events are delivered through hooks.
		err := p.Run(ctx)
		if err != nil {
			yield(schema.AgentEvent{}, err)
		}
	}
}
