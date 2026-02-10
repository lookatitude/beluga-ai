// Package voice provides the voice and multimodal pipeline for the Beluga AI
// framework. It implements a frame-based processing model inspired by Pipecat
// where atomic Frames (audio chunks, text fragments, images, control signals)
// flow through linked FrameProcessors via Go channels.
//
// # Frame Types
//
// The fundamental data unit is the [Frame], which carries typed data:
//
//   - [FrameAudio] — raw audio data (PCM, Opus, etc.)
//   - [FrameText] — text fragments (transcripts, LLM output)
//   - [FrameControl] — control signals (start, stop, interrupt, end-of-utterance)
//   - [FrameImage] — image/video frames for multimodal pipelines
//
// Convenience constructors are provided: [NewAudioFrame], [NewTextFrame],
// [NewControlFrame], and [NewImageFrame].
//
// # FrameProcessor Interface
//
// The core abstraction is the [FrameProcessor] interface. Each processor reads
// frames from an input channel, processes them, and writes results to an output
// channel. Processors run as goroutines and must close the output channel when
// done.
//
//	type FrameProcessor interface {
//	    Process(ctx context.Context, in <-chan Frame, out chan<- Frame) error
//	}
//
// Use [FrameProcessorFunc] to adapt plain functions as FrameProcessors. Use
// [Chain] to connect multiple processors in series.
//
// # Pipeline Modes
//
// Three composable pipeline modes are supported:
//
//   - Cascading: STT → LLM → TTS (each a FrameProcessor goroutine)
//   - S2S: Native audio-in/audio-out (OpenAI Realtime, Gemini Live)
//   - Hybrid: S2S default, fallback to cascade for complex tool use
//
// # Cascading Pipeline
//
// The [VoicePipeline] implements the cascading mode:
//
//	pipe := voice.NewPipeline(
//	    voice.WithTransport(transport),
//	    voice.WithVAD(vad),
//	    voice.WithSTT(stt),
//	    voice.WithLLM(model),
//	    voice.WithTTS(tts),
//	)
//	err := pipe.Run(ctx)
//
// # Hybrid Pipeline
//
// The [HybridPipeline] combines S2S and cascade modes, switching based on a
// configurable [SwitchPolicy]:
//
//	hybrid := voice.NewHybridPipeline(
//	    voice.WithS2S(s2sEngine),
//	    voice.WithCascade(cascadePipeline),
//	    voice.WithSwitchPolicy(voice.OnToolOverload),
//	)
//	err := hybrid.Run(ctx)
//
// # Voice Activity Detection
//
// The [VAD] interface detects speech in audio data. A built-in [EnergyVAD]
// uses RMS energy thresholds, and providers in voice/vad/providers/ offer
// Silero and WebRTC-based detection. The VAD registry follows the standard
// [RegisterVAD]/[NewVAD]/[ListVAD] pattern.
//
// # Session Management
//
// The [VoiceSession] tracks conversational state (idle, listening, speaking)
// and [Turn] history. It is safe for concurrent use.
//
// # Hooks
//
// The [Hooks] struct provides optional callbacks for pipeline events:
// OnSpeechStart, OnSpeechEnd, OnTranscript, OnResponse, and OnError.
// Use [ComposeHooks] to merge multiple hooks.
//
// # Latency Budget
//
// Target end-to-end latency: transport <50ms, VAD <1ms, STT <200ms,
// LLM TTFT <300ms, TTS TTFB <200ms, return <50ms = <800ms E2E.
package voice
