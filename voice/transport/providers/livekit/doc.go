// Package livekit provides the LiveKit transport provider for the Beluga AI
// voice pipeline. It implements the [transport.AudioTransport] interface for
// bidirectional audio I/O through LiveKit rooms.
//
// LiveKit is treated as a TRANSPORT, not a framework dependency. LiveKit
// provides WebRTC transport while Beluga handles all STT/LLM/TTS processing
// through the frame-based pipeline.
//
// # Registration
//
// This package registers itself as "livekit" with the transport registry.
// Import it with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/transport/providers/livekit"
//
// # Usage
//
//	t, err := transport.New("livekit", transport.Config{
//	    URL:   "wss://myapp.livekit.cloud",
//	    Token: "...",
//	    Extra: map[string]any{"room": "my-room"},
//	})
//	frames, err := t.Recv(ctx)
//
// # Configuration
//
// Required fields in [transport.Config]:
//
//   - URL — LiveKit server URL (required)
//   - Token — LiveKit authentication token (required)
//
// Optional Extra fields:
//
//   - room — LiveKit room name
//
// Default sample rate is 16000 Hz, default channel count is 1 (mono).
//
// # Exported Types
//
//   - [Transport] — implements transport.AudioTransport for LiveKit
//   - [New] — constructor accepting transport.Config
package livekit
