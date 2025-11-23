package spectral

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/noise"
)

func init() {
	// Register Spectral provider with the global registry
	noise.GetRegistry().Register("spectral", NewSpectralProvider)
}
