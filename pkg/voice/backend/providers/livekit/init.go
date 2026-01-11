package livekit

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
)

func init() {
	// Register LiveKit provider with the global registry
	provider := NewLiveKitProvider()
	backend.GetRegistry().Register("livekit", provider.CreateBackend)
}
