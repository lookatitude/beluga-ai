// Package mock provides the mock voice backend provider for testing.
//
// Deprecated: This package has been moved to pkg/voicebackend/providers/mock.
// Please update your imports to use github.com/lookatitude/beluga-ai/pkg/voicebackend/providers/mock.
// This package will be removed in v2.0.
package mock

import (
	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
)

func init() {
	// Register mock provider with the global registry
	provider := NewMockProvider()
	backend.GetRegistry().Register("mock", provider.CreateBackend)
}
