// Package assemblyai provides the AssemblyAI STT provider for the Beluga AI
// voice pipeline. It uses the AssemblyAI Transcription API for batch
// transcription and WebSocket API for real-time streaming.
//
// # Registration
//
// This package registers itself as "assemblyai" with the stt registry. Import
// it with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/stt/providers/assemblyai"
//
// # Usage
//
//	engine, err := stt.New("assemblyai", stt.Config{
//	    Language: "en",
//	    Extra: map[string]any{"api_key": "..."},
//	})
//	text, err := engine.Transcribe(ctx, audioBytes)
//
// Batch transcription uploads audio, creates a transcript, and polls for
// completion. Streaming uses the real-time WebSocket endpoint for low-latency
// partial and final transcripts with word-level timing.
//
// # Configuration
//
// Required configuration in Config.Extra:
//
//   - api_key — AssemblyAI API key (required)
//   - base_url — Custom REST API base URL (optional)
//   - ws_url — Custom WebSocket URL (optional)
//
// # Exported Types
//
//   - [Engine] — implements stt.STT using AssemblyAI
//   - [New] — constructor accepting stt.Config
package assemblyai
