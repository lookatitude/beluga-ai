package cartesia

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/voicebackend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voicebackend/iface"
)

func init() {
	// Auto-register Cartesia provider with the global registry (T239)
	provider := NewCartesiaProvider()
	voicebackend.GetRegistry().Register("cartesia", func(ctx context.Context, config *vbiface.Config) (vbiface.VoiceBackend, error) {
		return provider.CreateBackend(ctx, config)
	})
}
