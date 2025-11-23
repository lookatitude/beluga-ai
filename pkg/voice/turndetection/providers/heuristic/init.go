package heuristic

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/turndetection"
)

func init() {
	// Register Heuristic provider with the global registry
	turndetection.GetRegistry().Register("heuristic", NewHeuristicProvider)
}
