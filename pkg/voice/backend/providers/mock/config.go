// Package mock provides the mock voice backend provider for testing.
//
// Deprecated: This package has been moved to pkg/voicebackend/providers/mock.
// Please update your imports to use github.com/lookatitude/beluga-ai/pkg/voicebackend/providers/mock.
// This package will be removed in v2.0.
package mock

import (
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// MockConfig extends the base Config with mock-specific fields.
type MockConfig struct {
	*vbiface.Config
	// Mock-specific configuration can be added here if needed
}

// NewMockConfig creates a new mock configuration.
func NewMockConfig(baseConfig *vbiface.Config) *MockConfig {
	return &MockConfig{
		Config: baseConfig,
	}
}
