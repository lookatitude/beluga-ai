package webrtc

import (
	"github.com/lookatitude/beluga-ai/pkg/vad"
)

func init() {
	// Register WebRTC provider with the global registry
	vad.GetRegistry().Register("webrtc", NewWebRTCProvider)
}
