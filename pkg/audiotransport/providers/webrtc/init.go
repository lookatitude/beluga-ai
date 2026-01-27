package webrtc

import (
	"github.com/lookatitude/beluga-ai/pkg/audiotransport"
)

func init() {
	// Register WebRTC provider with the global registry
	audiotransport.GetRegistry().Register("webrtc", NewWebRTCTransport)
}
