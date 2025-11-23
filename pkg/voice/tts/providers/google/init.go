package google

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/tts"
)

func init() {
	// Register Google provider with the global registry
	tts.GetRegistry().Register("google", NewGoogleProvider)
}
