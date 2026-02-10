// Package transport provides the audio transport interface and registry for the
// Beluga AI voice pipeline. Transports handle bidirectional audio I/O between
// clients and the voice pipeline, abstracting the underlying protocol
// (WebSocket, LiveKit, Daily, etc.).
//
// LiveKit is treated as a transport, not a framework dependency. LiveKit's
// server provides WebRTC transport, while Beluga handles all STT/LLM/TTS
// processing through the frame-based pipeline.
//
// # Core Interface
//
// The [AudioTransport] interface provides bidirectional audio I/O:
//
//	type AudioTransport interface {
//	    Recv(ctx context.Context) (<-chan voice.Frame, error)
//	    Send(ctx context.Context, frame voice.Frame) error
//	    AudioOut() io.Writer
//	    Close() error
//	}
//
// # Registry Pattern
//
// Providers register via [Register] in their init() function and are created
// with [New]. Use [List] to discover available providers.
//
//	import _ "github.com/lookatitude/beluga-ai/voice/transport"
//
//	t, err := transport.New("websocket", transport.Config{URL: "ws://..."})
//	frames, err := t.Recv(ctx)
//	for frame := range frames {
//	    // process incoming audio frame
//	}
//
// # Pipeline Integration
//
// Use [AsVoiceTransport] to adapt an AudioTransport to the voice.Transport
// interface expected by the [voice.VoicePipeline].
//
// # Built-in Transport
//
// The package includes a [WebSocketTransport] implementation registered as
// "websocket". Configure it with [NewWebSocketTransport] and options
// [WithWSSampleRate] and [WithWSChannels].
//
// # Configuration
//
// The [Config] struct supports URL, authentication token, sample rate,
// channel count, and provider-specific extras. Use functional options
// [WithURL], [WithToken], [WithSampleRate], and [WithChannels].
//
// # Available Providers
//
//   - websocket — Built-in WebSocket transport (voice/transport)
//   - livekit — LiveKit WebRTC rooms (voice/transport/providers/livekit)
//   - daily — Daily.co rooms (voice/transport/providers/daily)
//   - pipecat — Pipecat server (voice/transport/providers/pipecat)
package transport
