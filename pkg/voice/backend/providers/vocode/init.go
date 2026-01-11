package vocode

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

func init() {
	// Auto-register Vocode provider with the global registry (T223)
	provider := NewVocodeProvider()
	backend.GetRegistry().Register("vocode", func(ctx context.Context, config *vbiface.Config) (vbiface.VoiceBackend, error) {
		return provider.CreateBackend(ctx, config)
	})
}
