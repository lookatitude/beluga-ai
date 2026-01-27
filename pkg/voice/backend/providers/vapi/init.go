// Package vapi provides the Vapi voice backend provider.
//
// Deprecated: This package has been moved to pkg/voicebackend/providers/vapi.
// Please update your imports to use github.com/lookatitude/beluga-ai/pkg/voicebackend/providers/vapi.
// This package will be removed in v2.0.
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
