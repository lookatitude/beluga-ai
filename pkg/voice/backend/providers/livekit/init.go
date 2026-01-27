// Package livekit provides the LiveKit voice backend provider.
//
// Deprecated: This package has been moved to pkg/voicebackend/providers/livekit.
// Please update your imports to use github.com/lookatitude/beluga-ai/pkg/voicebackend/providers/livekit.
// This package will be removed in v2.0.
package livekit

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
)

func init() {
	// Register LiveKit provider with the global registry
	provider := NewLiveKitProvider()
	backend.GetRegistry().Register("livekit", provider.CreateBackend)
}
