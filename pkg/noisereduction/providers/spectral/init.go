package spectral

import (
	"github.com/lookatitude/beluga-ai/pkg/noisereduction"
)

func init() {
	// Register Spectral provider with the global registry
	noisereduction.GetRegistry().Register("spectral", NewSpectralProvider)
}
