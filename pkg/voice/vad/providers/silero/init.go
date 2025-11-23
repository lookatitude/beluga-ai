package silero

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/vad"
)

func init() {
	// Register Silero provider with the global registry
	vad.GetRegistry().Register("silero", NewSileroProvider)
}
