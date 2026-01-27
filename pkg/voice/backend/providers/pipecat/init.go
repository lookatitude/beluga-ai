// Package pipecat provides the Pipecat voice backend provider.
//
// Deprecated: This package has been moved to pkg/voicebackend/providers/pipecat.
// Please update your imports to use github.com/lookatitude/beluga-ai/pkg/voicebackend/providers/pipecat.
// This package will be removed in v2.0.
package pipecat

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

func init() {
	// Auto-register Pipecat provider with the global registry (T191)
	provider := NewPipecatProvider()
	backend.GetRegistry().Register("pipecat", func(ctx context.Context, config *vbiface.Config) (vbiface.VoiceBackend, error) {
		return provider.CreateBackend(ctx, config)
	})
}
