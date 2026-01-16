package twilio

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

func init() {
	// Auto-register Twilio provider with the global registry (T039)
	provider := NewTwilioProvider()
	backend.GetRegistry().Register("twilio", func(ctx context.Context, config *vbiface.Config) (vbiface.VoiceBackend, error) {
		return provider.CreateBackend(ctx, config)
	})
}
