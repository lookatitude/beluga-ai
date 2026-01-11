package mock

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
)

func init() {
	// Register mock provider with the global registry
	provider := NewMockProvider()
	backend.GetRegistry().Register("mock", provider.CreateBackend)
}
