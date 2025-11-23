package webrtc

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/noise"
)

func init() {
	// Register WebRTC provider with the global registry
	noise.GetRegistry().Register("webrtc", NewWebRTCNoiseProvider)
}
