package grok

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/s2s"
	s2siface "github.com/lookatitude/beluga-ai/pkg/voice/s2s/iface"
)

func init() {
	// Register Grok provider with the global registry
	// Wrap the constructor to match the expected signature
	s2s.GetRegistry().Register("grok", func(config *s2s.Config) (s2siface.S2SProvider, error) {
		return NewGrokVoiceProvider(config)
	})
}
