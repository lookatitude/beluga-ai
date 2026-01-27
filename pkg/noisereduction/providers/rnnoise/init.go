package rnnoise

import (
	"github.com/lookatitude/beluga-ai/pkg/noisereduction"
)

func init() {
	// Register RNNoise provider with the global registry
	noisereduction.GetRegistry().Register("rnnoise", NewRNNoiseProvider)
}
