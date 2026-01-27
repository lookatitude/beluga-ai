// Deprecated: This package has been moved to pkg/voicebackend/providers/twilio.
// Please update your imports to use the new location. This package will be removed
// in a future release as part of the pkg/voice deprecation (Phase 1 of 3).
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
