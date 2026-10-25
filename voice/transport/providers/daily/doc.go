// Package daily registers the "daily" transport name for Daily.co rooms in the
// Beluga AI voice pipeline.
//
// # Status: not implemented
//
// This provider is a placeholder. It does not connect to Daily:
// [New] (and transport.New("daily", ...)) always returns an error wrapping
// [transport.ErrNotImplemented], which in turn wraps errors.ErrUnsupported.
// Earlier releases returned a transport that silently discarded audio and
// never produced frames; that fake success was removed.
//
// Use the built-in "websocket" transport (voice/transport) until a real
// Daily integration is available.
//
//	import _ "github.com/lookatitude/beluga-ai/v2/voice/transport/providers/daily"
//
//	_, err := transport.New("daily", transport.Config{URL: "..."})
//	if errors.Is(err, transport.ErrNotImplemented) {
//	    // fall back to another transport
//	}
package daily
