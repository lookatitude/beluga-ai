// Package stt provides the speech-to-text (STT) interface and provider registry
// for the Beluga AI voice pipeline. Providers implement the [STT] interface and
// register themselves via init() for discovery.
//
// # Core Interface
//
// The [STT] interface supports both batch and streaming transcription:
//
//	type STT interface {
//	    Transcribe(ctx context.Context, audio []byte, opts ...Option) (string, error)
//	    TranscribeStream(ctx context.Context, audioStream iter.Seq2[[]byte, error], opts ...Option) iter.Seq2[TranscriptEvent, error]
//	}
//
// # Transcript Events
//
// Streaming transcription produces [TranscriptEvent] values containing the
// transcribed text, finality flag, confidence score, timestamp, detected
// language, and optional word-level timing via [Word].
//
// # Registry Pattern
//
// Providers register via [Register] in their init() function and are created
// with [New]. Use [List] to discover available providers.
//
//	import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
//
//	engine, err := stt.New("deepgram", stt.Config{Language: "en"})
//	text, err := engine.Transcribe(ctx, audioBytes)
//
//	// Streaming:
//	for event, err := range engine.TranscribeStream(ctx, audioStream) {
//	    if err != nil { break }
//	    fmt.Printf("[%v] %s (final=%v)\n", event.Timestamp, event.Text, event.IsFinal)
//	}
//
// # Frame Processor Integration
//
// Use [AsFrameProcessor] to wrap an STT engine as a voice.FrameProcessor
// for integration with the cascading pipeline.
//
//	processor := stt.AsFrameProcessor(engine)
//
// # Configuration
//
// The [Config] struct supports language, model, punctuation, diarization,
// sample rate, encoding, and provider-specific extras. Use functional options
// like [WithLanguage], [WithModel], and [WithPunctuation] to configure
// individual operations.
//
// # Hooks
//
// The [Hooks] struct provides callbacks: OnTranscript (each event),
// OnUtterance (finalized text), and OnError. Use [ComposeHooks] to merge.
//
// # Available Providers
//
//   - deepgram — Deepgram Nova-2 (voice/stt/providers/deepgram)
//   - assemblyai — AssemblyAI (voice/stt/providers/assemblyai)
//   - whisper — OpenAI Whisper (voice/stt/providers/whisper)
//   - groq — Groq Whisper (voice/stt/providers/groq)
//   - elevenlabs — ElevenLabs Scribe (voice/stt/providers/elevenlabs)
//   - gladia — Gladia (voice/stt/providers/gladia)
package stt
