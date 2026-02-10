// Package tts provides the text-to-speech (TTS) interface and provider registry
// for the Beluga AI voice pipeline. Providers implement the [TTS] interface and
// register themselves via init() for discovery.
//
// # Core Interface
//
// The [TTS] interface supports both batch and streaming synthesis:
//
//	type TTS interface {
//	    Synthesize(ctx context.Context, text string, opts ...Option) ([]byte, error)
//	    SynthesizeStream(ctx context.Context, textStream iter.Seq2[string, error], opts ...Option) iter.Seq2[[]byte, error]
//	}
//
// # Audio Formats
//
// Supported output formats are defined as [AudioFormat] constants:
// [FormatPCM], [FormatOpus], [FormatMP3], and [FormatWAV].
//
// # Registry Pattern
//
// Providers register via [Register] in their init() function and are created
// with [New]. Use [List] to discover available providers.
//
//	import _ "github.com/lookatitude/beluga-ai/voice/tts/providers/elevenlabs"
//
//	engine, err := tts.New("elevenlabs", tts.Config{Voice: "rachel"})
//	audio, err := engine.Synthesize(ctx, "Hello, world!")
//
//	// Streaming:
//	for chunk, err := range engine.SynthesizeStream(ctx, textStream) {
//	    if err != nil { break }
//	    transport.Send(chunk)
//	}
//
// # Frame Processor Integration
//
// Use [AsFrameProcessor] to wrap a TTS engine as a voice.FrameProcessor for
// integration with the cascading pipeline:
//
//	processor := tts.AsFrameProcessor(engine, 24000)
//
// # Configuration
//
// The [Config] struct supports voice, model, sample rate, format, speed, pitch,
// and provider-specific extras. Use functional options like [WithVoice],
// [WithModel], [WithSampleRate], [WithFormat], [WithSpeed], and [WithPitch]
// to configure individual operations.
//
// # Hooks
//
// The [Hooks] struct provides callbacks: BeforeSynthesize, OnAudioChunk, and
// OnError. Use [ComposeHooks] to merge multiple hooks.
//
// # Available Providers
//
//   - elevenlabs — ElevenLabs (voice/tts/providers/elevenlabs)
//   - cartesia — Cartesia Sonic (voice/tts/providers/cartesia)
//   - playht — PlayHT (voice/tts/providers/playht)
//   - lmnt — LMNT (voice/tts/providers/lmnt)
//   - fish — Fish Audio (voice/tts/providers/fish)
//   - groq — Groq TTS (voice/tts/providers/groq)
//   - smallest — Smallest.ai (voice/tts/providers/smallest)
package tts
