package voice

import (
	"context"
	"fmt"
	"iter"

	"github.com/lookatitude/beluga-ai/schema"
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
	// Recv returns a channel of incoming audio frames from the client.
	Recv(ctx context.Context) (<-chan Frame, error)

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
	return Hooks{
		OnSpeechStart: func(ctx context.Context) {
			for _, h := range hooks {
				if h.OnSpeechStart != nil {
					h.OnSpeechStart(ctx)
				}
			}
		},
		OnSpeechEnd: func(ctx context.Context) {
			for _, h := range hooks {
				if h.OnSpeechEnd != nil {
					h.OnSpeechEnd(ctx)
				}
			}
		},
		OnTranscript: func(ctx context.Context, text string) {
			for _, h := range hooks {
				if h.OnTranscript != nil {
					h.OnTranscript(ctx, text)
				}
			}
		},
		OnResponse: func(ctx context.Context, text string) {
			for _, h := range hooks {
				if h.OnResponse != nil {
					h.OnResponse(ctx, text)
				}
			}
		},
		OnError: func(ctx context.Context, err error) error {
			for _, h := range hooks {
				if h.OnError != nil {
					if e := h.OnError(ctx, err); e != nil {
						return e
					}
				}
			}
			return err
		},
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

	// ChannelBufferSize is the buffer size for inter-processor channels.
	// Defaults to 64 if zero.
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

// WithChannelBufferSize sets the buffer size for inter-processor channels.
func WithChannelBufferSize(size int) PipelineOption {
	return func(cfg *PipelineConfig) {
		cfg.ChannelBufferSize = size
	}
}

// VoicePipeline implements the cascading voice pipeline: STT → LLM → TTS.
// Each stage is a FrameProcessor goroutine connected by channels.
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
// Transport in a chain of FrameProcessor goroutines.
func (p *VoicePipeline) Run(ctx context.Context) error {
	if p.config.Transport == nil {
		return fmt.Errorf("voice: pipeline requires a transport")
	}

	// Receive audio frames from transport.
	incoming, err := p.config.Transport.Recv(ctx)
	if err != nil {
		return fmt.Errorf("voice: transport recv: %w", err)
	}

	// Build the processor chain from available components.
	var processors []FrameProcessor

	// VAD processor: filters audio frames based on speech detection.
	if p.config.VAD != nil {
		processors = append(processors, p.vadProcessor())
	}

	// STT, LLM, TTS processors.
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
		return fmt.Errorf("voice: pipeline has no processors")
	}

	// Chain processors and run.
	chain := Chain(processors...)

	bufSize := p.config.ChannelBufferSize
	out := make(chan Frame, bufSize)

	// Process output frames back through transport.
	done := make(chan error, 1)
	go func() {
		defer close(done)
		for frame := range out {
			if sendErr := p.config.Transport.Send(ctx, frame); sendErr != nil {
				done <- fmt.Errorf("voice: transport send: %w", sendErr)
				return
			}
		}
	}()

	// Run the processor chain.
	if chainErr := chain.Process(ctx, incoming, out); chainErr != nil {
		return chainErr
	}

	// Wait for output drain.
	if sendErr := <-done; sendErr != nil {
		return sendErr
	}

	return nil
}

// processVADResult emits control frames for VAD state transitions and
// forwards speech audio frames to out.
func (p *VoicePipeline) processVADResult(ctx context.Context, result ActivityResult, frame Frame, out chan<- Frame) {
	switch result.EventType {
	case VADSpeechStart:
		if p.config.Hooks.OnSpeechStart != nil {
			p.config.Hooks.OnSpeechStart(ctx)
		}
		out <- NewControlFrame(SignalStart)
	case VADSpeechEnd:
		if p.config.Hooks.OnSpeechEnd != nil {
			p.config.Hooks.OnSpeechEnd(ctx)
		}
		out <- NewControlFrame(SignalEndOfUtterance)
	}

	if result.IsSpeech {
		out <- frame
	}
}

// vadProcessor creates a FrameProcessor that runs VAD on audio frames and
// injects control frames for speech start/end events.
func (p *VoicePipeline) vadProcessor() FrameProcessor {
	return FrameProcessorFunc(func(ctx context.Context, in <-chan Frame, out chan<- Frame) error {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case frame, ok := <-in:
				if !ok {
					return nil
				}
				if frame.Type != FrameAudio {
					out <- frame
					continue
				}

				result, err := p.config.VAD.DetectActivity(ctx, frame.Data)
				if err != nil {
					if p.config.Hooks.OnError != nil {
						if hookErr := p.config.Hooks.OnError(ctx, err); hookErr != nil {
							return hookErr
						}
					}
					continue
				}

				p.processVADResult(ctx, result, frame, out)
			}
		}
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
