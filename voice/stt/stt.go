package stt

import (
	"context"
	"fmt"
	"iter"
	"time"

	"github.com/lookatitude/beluga-ai/voice"
)

// TranscriptEvent represents a transcription result from a streaming STT
// engine. Partial results have IsFinal set to false and may be revised in
// subsequent events.
type TranscriptEvent struct {
	// Text is the transcribed text.
	Text string

	// IsFinal indicates whether this is a final (non-revisable) transcript.
	IsFinal bool

	// Confidence is the engine's confidence in the transcription (0.0 to 1.0).
	Confidence float64

	// Timestamp is the audio timestamp of this transcription event.
	Timestamp time.Duration

	// Language is the detected language code (e.g., "en", "es").
	Language string

	// Words holds word-level timing information when available.
	Words []Word
}

// Word represents a single word with timing information.
type Word struct {
	// Text is the word text.
	Text string

	// Start is the start time of the word in the audio.
	Start time.Duration

	// End is the end time of the word in the audio.
	End time.Duration

	// Confidence is the word-level confidence score.
	Confidence float64
}

// STT is the speech-to-text interface. Implementations convert audio data
// to text, supporting both batch and streaming modes.
type STT interface {
	// Transcribe converts a complete audio buffer to text.
	Transcribe(ctx context.Context, audio []byte, opts ...Option) (string, error)

	// TranscribeStream converts a streaming audio source to a stream of
	// transcript events. Partial results are emitted as they become available.
	TranscribeStream(ctx context.Context, audioStream iter.Seq2[[]byte, error], opts ...Option) iter.Seq2[TranscriptEvent, error]
}

// Config holds configuration options for STT operations.
type Config struct {
	// Language is the BCP-47 language code (e.g., "en-US", "es").
	Language string

	// Model is the STT model to use (provider-specific).
	Model string

	// Punctuation enables automatic punctuation insertion.
	Punctuation bool

	// Diarization enables speaker diarization (identification of different speakers).
	Diarization bool

	// SampleRate is the audio sample rate in Hz.
	SampleRate int

	// Encoding is the audio encoding format (e.g., "linear16", "opus").
	Encoding string

	// Extra holds provider-specific configuration.
	Extra map[string]any
}

// Option configures an STT operation.
type Option func(*Config)

// WithLanguage sets the language for transcription.
func WithLanguage(lang string) Option {
	return func(cfg *Config) {
		cfg.Language = lang
	}
}

// WithModel sets the STT model to use.
func WithModel(model string) Option {
	return func(cfg *Config) {
		cfg.Model = model
	}
}

// WithPunctuation enables or disables automatic punctuation.
func WithPunctuation(enabled bool) Option {
	return func(cfg *Config) {
		cfg.Punctuation = enabled
	}
}

// WithDiarization enables or disables speaker diarization.
func WithDiarization(enabled bool) Option {
	return func(cfg *Config) {
		cfg.Diarization = enabled
	}
}

// WithSampleRate sets the audio sample rate.
func WithSampleRate(rate int) Option {
	return func(cfg *Config) {
		cfg.SampleRate = rate
	}
}

// WithEncoding sets the audio encoding format.
func WithEncoding(encoding string) Option {
	return func(cfg *Config) {
		cfg.Encoding = encoding
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

// Hooks provides optional callback functions for STT operations.
type Hooks struct {
	// OnTranscript is called for each transcript event (interim and final).
	OnTranscript func(ctx context.Context, event TranscriptEvent)

	// OnUtterance is called when a complete utterance is finalized.
	OnUtterance func(ctx context.Context, text string)

	// OnError is called when an error occurs. Returning nil suppresses it.
	OnError func(ctx context.Context, err error) error
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
func ComposeHooks(hooks ...Hooks) Hooks {
	return Hooks{
		OnTranscript: func(ctx context.Context, event TranscriptEvent) {
			for _, h := range hooks {
				if h.OnTranscript != nil {
					h.OnTranscript(ctx, event)
				}
			}
		},
		OnUtterance: func(ctx context.Context, text string) {
			for _, h := range hooks {
				if h.OnUtterance != nil {
					h.OnUtterance(ctx, text)
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

// AsFrameProcessor wraps an STT engine as a voice.FrameProcessor.
// It reads audio frames from in, runs transcription, and emits text frames
// to out with transcription results.
func AsFrameProcessor(engine STT, opts ...Option) voice.FrameProcessor {
	return voice.FrameProcessorFunc(func(ctx context.Context, in <-chan voice.Frame, out chan<- voice.Frame) error {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case frame, ok := <-in:
				if !ok {
					return nil
				}
				// Pass through non-audio frames.
				if frame.Type != voice.FrameAudio {
					out <- frame
					continue
				}
				// Transcribe the audio chunk.
				text, err := engine.Transcribe(ctx, frame.Data, opts...)
				if err != nil {
					return fmt.Errorf("stt: transcribe: %w", err)
				}
				if text != "" {
					out <- voice.NewTextFrame(text)
				}
			}
		}
	})
}
