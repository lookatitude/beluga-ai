// Package vocode provides the Vocode voice backend provider.
//
// Deprecated: This package has been moved to pkg/voicebackend/providers/vocode.
// Please update your imports to use github.com/lookatitude/beluga-ai/pkg/voicebackend/providers/vocode.
// This package will be removed in v2.0.
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
