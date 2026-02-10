// Package pipecat provides the Pipecat transport provider for the Beluga AI
// voice pipeline. It implements the [transport.AudioTransport] interface for
// bidirectional audio I/O through a Pipecat server over WebSocket.
//
// # Registration
//
// This package registers itself as "pipecat" with the transport registry.
// Import it with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/transport/providers/pipecat"
//
// # Usage
//
//	t, err := transport.New("pipecat", transport.Config{
//	    URL: "ws://localhost:8765",
//	})
//	frames, err := t.Recv(ctx)
//
// # Configuration
//
// The transport requires a Pipecat server WebSocket URL. Default sample rate
// is 16000 Hz.
//
// # Exported Types
//
//   - [Transport] — implements transport.AudioTransport for Pipecat
//   - [New] — constructor accepting transport.Config
package pipecat
