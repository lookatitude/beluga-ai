// Package pipecat registers the "pipecat" transport name for Pipecat servers in the
// Beluga AI voice pipeline.
//
// # Status: not implemented
//
// This provider is a placeholder. It does not connect to Pipecat:
// [New] (and transport.New("pipecat", ...)) always returns an error wrapping
// [transport.ErrNotImplemented], which in turn wraps errors.ErrUnsupported.
// Earlier releases returned a transport that silently discarded audio and
// never produced frames; that fake success was removed.
//
// Use the built-in "websocket" transport (voice/transport) until a real
// Pipecat integration is available.
//
//	import _ "github.com/lookatitude/beluga-ai/v2/voice/transport/providers/pipecat"
//
//	_, err := transport.New("pipecat", transport.Config{URL: "..."})
//	if errors.Is(err, transport.ErrNotImplemented) {
//	    // fall back to another transport
//	}
package pipecat
