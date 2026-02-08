// Package s2s provides the speech-to-speech (S2S) interface and provider
// registry for the Beluga AI voice pipeline. S2S providers handle native
// audio-in/audio-out via their own transport (WebRTC, WebSocket), bypassing
// the STT → LLM → TTS cascade for lower latency.
//
// Usage:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/s2s/providers/openai_realtime"
//
//	engine, err := s2s.New("openai_realtime", s2s.Config{Voice: "alloy"})
//	session, err := engine.Start(ctx)
//	defer session.Close()
//
//	session.SendAudio(ctx, audioChunk)
//	for event := range session.Recv() {
//	    switch event.Type {
//	    case s2s.EventAudioOutput:
//	        playAudio(event.Audio)
//	    case s2s.EventToolCall:
//	        handleToolCall(event.ToolCall)
//	    }
//	}
package s2s

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/voice"
)

// SessionEventType identifies the type of S2S session event.
type SessionEventType string

const (
	// EventAudioOutput indicates the session produced audio output.
	EventAudioOutput SessionEventType = "audio_output"

	// EventTextOutput indicates the session produced text output.
	EventTextOutput SessionEventType = "text_output"

	// EventTranscript indicates a user speech transcript.
	EventTranscript SessionEventType = "transcript"

	// EventToolCall indicates the model wants to invoke a tool.
	EventToolCall SessionEventType = "tool_call"

	// EventTurnEnd indicates the end of a conversational turn.
	EventTurnEnd SessionEventType = "turn_end"

	// EventError indicates an error occurred.
	EventError SessionEventType = "error"
)

// SessionEvent represents an event from an active S2S session.
type SessionEvent struct {
	// Type identifies the event kind.
	Type SessionEventType

	// Audio carries audio data for AudioOutput events.
	Audio []byte

	// Text carries text data for TextOutput and Transcript events.
	Text string

	// ToolCall carries tool call data for ToolCall events.
	ToolCall *schema.ToolCall

	// Error carries error information for Error events.
	Error error
}

// S2S is the speech-to-speech interface. Implementations provide bidirectional
// audio streaming via native provider protocols (e.g., OpenAI Realtime API,
// Gemini Live, Amazon Nova).
type S2S interface {
	// Start initiates a new S2S session with the provider.
	Start(ctx context.Context, opts ...Option) (Session, error)
}

// Session represents an active bidirectional audio session with an S2S provider.
type Session interface {
	// SendAudio sends an audio chunk to the provider.
	SendAudio(ctx context.Context, audio []byte) error

	// SendText sends a text message to the provider (for instructions or prompts).
	SendText(ctx context.Context, text string) error

	// SendToolResult sends a tool execution result back to the model.
	SendToolResult(ctx context.Context, result schema.ToolResult) error

	// Recv returns a channel of session events. The channel is closed
	// when the session ends.
	Recv() <-chan SessionEvent

	// Interrupt signals that the user has interrupted the model's output.
	Interrupt(ctx context.Context) error

	// Close terminates the session and releases resources.
	Close() error
}

// Config holds configuration options for S2S sessions.
type Config struct {
	// Voice is the voice identifier for the S2S session (provider-specific).
	Voice string

	// Model is the S2S model to use (provider-specific).
	Model string

	// Instructions is the system prompt or instructions for the session.
	Instructions string

	// Tools is the set of tool definitions available to the S2S session.
	Tools []schema.ToolDefinition

	// SampleRate is the audio sample rate in Hz.
	SampleRate int

	// Extra holds provider-specific configuration.
	Extra map[string]any
}

// Option configures an S2S session.
type Option func(*Config)

// WithVoice sets the voice for the session.
func WithVoice(voice string) Option {
	return func(cfg *Config) {
		cfg.Voice = voice
	}
}

// WithModel sets the S2S model.
func WithModel(model string) Option {
	return func(cfg *Config) {
		cfg.Model = model
	}
}

// WithInstructions sets the system instructions for the session.
func WithInstructions(instructions string) Option {
	return func(cfg *Config) {
		cfg.Instructions = instructions
	}
}

// WithTools sets the tools available to the session.
func WithTools(tools []schema.ToolDefinition) Option {
	return func(cfg *Config) {
		cfg.Tools = tools
	}
}

// WithSampleRate sets the audio sample rate.
func WithSampleRate(rate int) Option {
	return func(cfg *Config) {
		cfg.SampleRate = rate
	}
}

// ApplyOptions applies the given options to a Config and returns it.
func ApplyOptions(opts ...Option) Config {
	var cfg Config
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// Hooks provides optional callback functions for S2S operations.
type Hooks struct {
	// OnTurn is called when a conversational turn completes.
	OnTurn func(ctx context.Context, userText, agentText string)

	// OnInterrupt is called when the user interrupts the model.
	OnInterrupt func(ctx context.Context)

	// OnToolCall is called when the model requests a tool call.
	OnToolCall func(ctx context.Context, call schema.ToolCall)

	// OnError is called when an error occurs. Returning nil suppresses it.
	OnError func(ctx context.Context, err error) error
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
func ComposeHooks(hooks ...Hooks) Hooks {
	return Hooks{
		OnTurn: func(ctx context.Context, userText, agentText string) {
			for _, h := range hooks {
				if h.OnTurn != nil {
					h.OnTurn(ctx, userText, agentText)
				}
			}
		},
		OnInterrupt: func(ctx context.Context) {
			for _, h := range hooks {
				if h.OnInterrupt != nil {
					h.OnInterrupt(ctx)
				}
			}
		},
		OnToolCall: func(ctx context.Context, call schema.ToolCall) {
			for _, h := range hooks {
				if h.OnToolCall != nil {
					h.OnToolCall(ctx, call)
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

// AsFrameProcessor wraps an S2S engine as a voice.FrameProcessor.
// It creates a session, forwards audio frames, and emits output audio frames.
func AsFrameProcessor(engine S2S, opts ...Option) voice.FrameProcessor {
	return voice.FrameProcessorFunc(func(ctx context.Context, in <-chan voice.Frame, out chan<- voice.Frame) error {
		defer close(out)

		session, err := engine.Start(ctx, opts...)
		if err != nil {
			return fmt.Errorf("s2s: start session: %w", err)
		}
		defer session.Close()

		// Forward output events to out channel.
		done := make(chan error, 1)
		go func() {
			defer close(done)
			for event := range session.Recv() {
				switch event.Type {
				case EventAudioOutput:
					sampleRate := 24000 // default S2S sample rate
					out <- voice.NewAudioFrame(event.Audio, sampleRate)
				case EventTextOutput:
					out <- voice.NewTextFrame(event.Text)
				case EventTurnEnd:
					out <- voice.NewControlFrame(voice.SignalEndOfUtterance)
				case EventError:
					if event.Error != nil {
						done <- event.Error
						return
					}
				}
			}
		}()

		// Forward input audio frames to session.
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case frame, ok := <-in:
				if !ok {
					return <-done
				}
				switch frame.Type {
				case voice.FrameAudio:
					if sendErr := session.SendAudio(ctx, frame.Data); sendErr != nil {
						return fmt.Errorf("s2s: send audio: %w", sendErr)
					}
				case voice.FrameText:
					if sendErr := session.SendText(ctx, frame.Text()); sendErr != nil {
						return fmt.Errorf("s2s: send text: %w", sendErr)
					}
				case voice.FrameControl:
					if frame.Signal() == voice.SignalInterrupt {
						if intErr := session.Interrupt(ctx); intErr != nil {
							return fmt.Errorf("s2s: interrupt: %w", intErr)
						}
					}
				}
			}
		}
	})
}
