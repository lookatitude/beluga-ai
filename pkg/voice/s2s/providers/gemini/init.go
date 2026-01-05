package gemini

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
)

func init() {
	// Register Gemini provider with the global registry
	s2s.GetRegistry().Register("gemini", NewGeminiNativeProvider)
}
