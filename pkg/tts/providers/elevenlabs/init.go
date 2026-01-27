package elevenlabs

import (
	"github.com/lookatitude/beluga-ai/pkg/tts"
)

func init() {
	// Register ElevenLabs provider with the global registry
	tts.GetRegistry().Register("elevenlabs", NewElevenLabsProvider)
}
