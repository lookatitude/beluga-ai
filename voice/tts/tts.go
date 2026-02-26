package tts

import (
	"context"
	"fmt"
	"iter"

	"github.com/lookatitude/beluga-ai/internal/hookutil"
	"github.com/lookatitude/beluga-ai/voice"
)

// AudioFormat identifies the encoding format of synthesized audio.
type AudioFormat string

const (
	// FormatPCM is raw 16-bit little-endian PCM audio.
	FormatPCM AudioFormat = "pcm"

	// FormatOpus is Opus-encoded audio.
	FormatOpus AudioFormat = "opus"

	// FormatMP3 is MP3-encoded audio.
	FormatMP3 AudioFormat = "mp3"

	// FormatWAV is WAV-encoded audio.
	FormatWAV AudioFormat = "wav"
)

// TTS is the text-to-speech interface. Implementations convert text to audio,
// supporting both batch and streaming modes.
type TTS interface {
	// Synthesize converts text to a complete audio buffer.
	Synthesize(ctx context.Context, text string, opts ...Option) ([]byte, error)

	// SynthesizeStream converts a streaming text source to a stream of audio
	// chunks. Audio chunks are emitted as they become available, enabling
	// low-latency playback.
	SynthesizeStream(ctx context.Context, textStream iter.Seq2[string, error], opts ...Option) iter.Seq2[[]byte, error]
}

// Config holds configuration options for TTS operations.
type Config struct {
	// Voice is the voice identifier (provider-specific, e.g., "rachel", "alloy").
	Voice string

	// Model is the TTS model to use (provider-specific).
	Model string

	// SampleRate is the output audio sample rate in Hz (e.g., 16000, 24000, 44100).
	SampleRate int

	// Format is the output audio format (PCM, Opus, MP3, WAV).
	Format AudioFormat

	// Speed is the speech rate multiplier (1.0 = normal, 0.5 = half speed, 2.0 = double).
	Speed float64

	// Pitch adjusts the voice pitch (-20.0 to 20.0, 0 = default).
	Pitch float64

	// Extra holds provider-specific configuration.
	Extra map[string]any
}

// Option configures a TTS operation.
type Option func(*Config)

// WithVoice sets the voice for synthesis.
func WithVoice(voice string) Option {
	return func(cfg *Config) {
		cfg.Voice = voice
	}
}

// WithModel sets the TTS model to use.
func WithModel(model string) Option {
	return func(cfg *Config) {
		cfg.Model = model
	}
}

// WithSampleRate sets the output audio sample rate in Hz.
func WithSampleRate(rate int) Option {
	return func(cfg *Config) {
		cfg.SampleRate = rate
	}
}

// WithFormat sets the output audio format.
func WithFormat(format AudioFormat) Option {
	return func(cfg *Config) {
		cfg.Format = format
	}
}

// WithSpeed sets the speech rate multiplier.
func WithSpeed(speed float64) Option {
	return func(cfg *Config) {
		cfg.Speed = speed
	}
}

// WithPitch sets the voice pitch adjustment.
func WithPitch(pitch float64) Option {
	return func(cfg *Config) {
		cfg.Pitch = pitch
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

// Hooks provides optional callback functions for TTS operations.
type Hooks struct {
	// BeforeSynthesize is called before synthesis starts with the input text.
	BeforeSynthesize func(ctx context.Context, text string)

	// OnAudioChunk is called for each audio chunk produced during streaming.
	OnAudioChunk func(ctx context.Context, chunk []byte)

	// OnError is called when an error occurs. Returning nil suppresses it.
	OnError func(ctx context.Context, err error) error
}

// ComposeHooks merges multiple Hooks into a single Hooks value.
func ComposeHooks(hooks ...Hooks) Hooks {
	h := append([]Hooks{}, hooks...)
	return Hooks{
		BeforeSynthesize: hookutil.ComposeVoid1(h, func(hk Hooks) func(context.Context, string) {
			return hk.BeforeSynthesize
		}),
		OnAudioChunk: hookutil.ComposeVoid1(h, func(hk Hooks) func(context.Context, []byte) {
			return hk.OnAudioChunk
		}),
		OnError: hookutil.ComposeErrorPassthrough(h, func(hk Hooks) func(context.Context, error) error {
			return hk.OnError
		}),
	}
}

// synthesizeFrame synthesizes a single text frame and sends the result to out.
// Non-text frames are passed through unchanged.
func synthesizeFrame(ctx context.Context, engine TTS, frame voice.Frame, sampleRate int, out chan<- voice.Frame, opts ...Option) error {
	if frame.Type != voice.FrameText {
		out <- frame
		return nil
	}
	audio, err := engine.Synthesize(ctx, frame.Text(), opts...)
	if err != nil {
		return fmt.Errorf("tts: synthesize: %w", err)
	}
	if len(audio) > 0 {
		out <- voice.NewAudioFrame(audio, sampleRate)
	}
	return nil
}

// AsFrameProcessor wraps a TTS engine as a voice.FrameProcessor.
// It reads text frames from in, runs synthesis, and emits audio frames
// to out with the synthesized audio.
func AsFrameProcessor(engine TTS, sampleRate int, opts ...Option) voice.FrameProcessor {
	return voice.FrameLoop(func(ctx context.Context, frame voice.Frame, out chan<- voice.Frame) error {
		return synthesizeFrame(ctx, engine, frame, sampleRate, out, opts...)
	})
}
