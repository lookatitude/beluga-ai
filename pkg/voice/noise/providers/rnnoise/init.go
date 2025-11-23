package rnnoise

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/noise"
)

func init() {
	// Register RNNoise provider with the global registry
	noise.GetRegistry().Register("rnnoise", NewRNNoiseProvider)
}
