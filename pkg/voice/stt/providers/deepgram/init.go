package deepgram

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

func init() {
	// Register Deepgram provider with the global registry
	stt.GetRegistry().Register("deepgram", NewDeepgramProvider)
}
