package webrtc

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/transport"
)

func init() {
	// Register WebRTC provider with the global registry
	transport.GetRegistry().Register("webrtc", NewWebRTCTransport)
}
