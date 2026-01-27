package rnnoise

import (
	"github.com/lookatitude/beluga-ai/pkg/vad"
)

func init() {
	// Register RNNoise provider with the global registry
	vad.GetRegistry().Register("rnnoise", NewRNNoiseProvider)
}
