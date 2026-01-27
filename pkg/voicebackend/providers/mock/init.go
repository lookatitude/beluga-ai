package mock

import (
	"github.com/lookatitude/beluga-ai/pkg/voicebackend"
)

func init() {
	// Register mock provider with the global registry
	provider := NewMockProvider()
	voicebackend.GetRegistry().Register("mock", provider.CreateBackend)
}
