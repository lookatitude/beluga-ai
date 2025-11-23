package energy

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
)

func init() {
	// Register Energy provider with the global registry
	vad.GetRegistry().Register("energy", NewEnergyProvider)
}
