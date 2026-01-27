package livekit

import (
	"github.com/lookatitude/beluga-ai/pkg/voicebackend"
)

func init() {
	// Register LiveKit provider with the global registry
	provider := NewLiveKitProvider()
	voicebackend.GetRegistry().Register("livekit", provider.CreateBackend)
}
