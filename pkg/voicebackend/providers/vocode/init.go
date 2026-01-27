package vocode

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/voicebackend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voicebackend/iface"
)

func init() {
	// Auto-register Vocode provider with the global registry (T223)
	provider := NewVocodeProvider()
	voicebackend.GetRegistry().Register("vocode", func(ctx context.Context, config *vbiface.Config) (vbiface.VoiceBackend, error) {
		return provider.CreateBackend(ctx, config)
	})
}
