// Package deepgram provides the Deepgram STT provider for the Beluga AI
// voice pipeline. It uses the Deepgram HTTP API for batch transcription and
// WebSocket API for real-time streaming.
//
// # Registration
//
// This package registers itself as "deepgram" with the stt registry. Import
// it with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/deepgram"
//
// # Usage
//
//	engine, err := stt.New("deepgram", stt.Config{Language: "en", Model: "nova-2"})
//	text, err := engine.Transcribe(ctx, audioBytes)
//
// Streaming transcription uses the Deepgram WebSocket API for low-latency
// partial and final transcripts with word-level timing.
//
// # Configuration
//
// Required configuration in Config.Extra:
//
//   - api_key — Deepgram API key (required)
//   - base_url — Custom REST API base URL (optional)
//   - ws_url — Custom WebSocket URL (optional)
//
// The default model is "nova-2". Language, punctuation, diarization, encoding,
// and sample rate are all supported through [stt.Config].
//
// # Exported Types
//
//   - [Engine] — implements stt.STT using Deepgram
//   - [New] — constructor accepting stt.Config
package deepgram
