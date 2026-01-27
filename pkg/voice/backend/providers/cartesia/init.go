// Package cartesia provides the Cartesia voice backend provider.
//
// Deprecated: This package has been moved to pkg/voicebackend/providers/cartesia.
// Please update your imports to use github.com/lookatitude/beluga-ai/pkg/voicebackend/providers/cartesia.
// This package will be removed in v2.0.
package cartesia

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

func init() {
	// Auto-register Cartesia provider with the global registry (T239)
	provider := NewCartesiaProvider()
	backend.GetRegistry().Register("cartesia", func(ctx context.Context, config *vbiface.Config) (vbiface.VoiceBackend, error) {
		return provider.CreateBackend(ctx, config)
	})
}
