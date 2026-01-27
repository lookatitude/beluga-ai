package webrtc

import (
	"github.com/lookatitude/beluga-ai/pkg/noisereduction"
)

func init() {
	// Register WebRTC provider with the global registry
	noisereduction.GetRegistry().Register("webrtc", NewWebRTCNoiseProvider)
}
