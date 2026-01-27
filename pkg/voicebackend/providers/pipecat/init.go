package pipecat

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/voicebackend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voicebackend/iface"
)

func init() {
	// Auto-register Pipecat provider with the global registry (T191)
	provider := NewPipecatProvider()
	voicebackend.GetRegistry().Register("pipecat", func(ctx context.Context, config *vbiface.Config) (vbiface.VoiceBackend, error) {
		return provider.CreateBackend(ctx, config)
	})
}
