// Package gladia provides the Gladia STT provider for the Beluga AI voice
// pipeline. It uses the Gladia API for batch transcription and WebSocket
// streaming for real-time transcription.
//
// # Registration
//
// This package registers itself as "gladia" with the stt registry. Import it
// with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/gladia"
//
// # Usage
//
//	engine, err := stt.New("gladia", stt.Config{
//	    Language: "en",
//	    Extra: map[string]any{"api_key": "..."},
//	})
//	text, err := engine.Transcribe(ctx, audioBytes)
//
// Batch transcription uploads audio via multipart form, creates a
// transcription job, and polls for the result. Streaming uses Gladia's live
// WebSocket endpoint.
//
// # Configuration
//
// Required configuration in Config.Extra:
//
//   - api_key — Gladia API key (required)
//   - base_url — Custom API base URL (optional)
//
// # Exported Types
//
//   - [Engine] — implements stt.STT using Gladia
//   - [New] — constructor accepting stt.Config
package gladia
