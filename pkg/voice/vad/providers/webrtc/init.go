package webrtc

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
)

func init() {
	// Register WebRTC provider with the global registry
	vad.GetRegistry().Register("webrtc", NewWebRTCProvider)
}
