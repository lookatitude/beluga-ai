package google

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/stt"
)

func init() {
	// Register Google provider with the global registry
	stt.GetRegistry().Register("google", NewGoogleProvider)
}
