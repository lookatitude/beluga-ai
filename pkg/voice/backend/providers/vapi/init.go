package vapi

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

func init() {
	// Auto-register Vapi provider with the global registry (T231)
	provider := NewVapiProvider()
	backend.GetRegistry().Register("vapi", func(ctx context.Context, config *vbiface.Config) (vbiface.VoiceBackend, error) {
		return provider.CreateBackend(ctx, config)
	})
}
