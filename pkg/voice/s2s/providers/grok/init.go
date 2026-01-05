package grok

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
)

func init() {
	// Register Grok provider with the global registry
	s2s.GetRegistry().Register("grok", NewGrokVoiceProvider)
}
