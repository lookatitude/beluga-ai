// Package groq provides the Groq STT provider for the Beluga AI voice pipeline.
// It uses the Groq Whisper endpoint (OpenAI-compatible API) for transcription.
//
// # Registration
//
// This package registers itself as "groq" with the stt registry. Import it
// with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/groq"
//
// # Usage
//
//	engine, err := stt.New("groq", stt.Config{
//	    Model: "whisper-large-v3",
//	    Extra: map[string]any{"api_key": "gsk-..."},
//	})
//	text, err := engine.Transcribe(ctx, audioBytes)
//
// Groq Whisper does not support native streaming. TranscribeStream buffers
// all audio chunks and transcribes them in a single batch request.
//
// # Configuration
//
// Required configuration in Config.Extra:
//
//   - api_key — Groq API key (required)
//   - base_url — Custom API base URL (optional)
//
// The default model is "whisper-large-v3".
//
// # Exported Types
//
//   - [Engine] — implements stt.STT using Groq Whisper
//   - [New] — constructor accepting stt.Config
package groq
