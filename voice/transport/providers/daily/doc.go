// Package daily provides the Daily.co transport provider for the Beluga AI
// voice pipeline. It implements the [transport.AudioTransport] interface for
// bidirectional audio I/O through Daily.co rooms.
//
// # Registration
//
// This package registers itself as "daily" with the transport registry. Import
// it with a blank identifier to enable:
//
//	import _ "github.com/lookatitude/beluga-ai/voice/transport/providers/daily"
//
// # Usage
//
//	t, err := transport.New("daily", transport.Config{
//	    URL:   "https://myapp.daily.co/room",
//	    Token: "...",
//	})
//	frames, err := t.Recv(ctx)
//
// # Configuration
//
// The transport requires a Daily.co room URL. An optional authentication token
// and sample rate (default 16000 Hz) can be provided via [transport.Config].
//
// # Exported Types
//
//   - [Transport] — implements transport.AudioTransport for Daily.co
//   - [New] — constructor accepting transport.Config
package daily
