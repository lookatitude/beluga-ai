package deepgram

import (
	"github.com/lookatitude/beluga-ai/pkg/stt"
)

func init() {
	// Register Deepgram provider with the global registry
	stt.GetRegistry().Register("deepgram", NewDeepgramProvider)
}
