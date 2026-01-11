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
