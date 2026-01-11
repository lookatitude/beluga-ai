package mock

import (
	"context"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/voice/backend"
	vbiface "github.com/lookatitude/beluga-ai/pkg/voice/backend/iface"
)

// MockProvider implements the BackendProvider interface for testing.
type MockProvider struct{}

// NewMockProvider creates a new mock provider.
func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

// GetName returns the provider name.
func (p *MockProvider) GetName() string {
	return "mock"
}

// GetCapabilities returns the capabilities of the mock provider.
func (p *MockProvider) GetCapabilities(ctx context.Context) (*vbiface.ProviderCapabilities, error) {
	return &vbiface.ProviderCapabilities{
		S2SSupport:            true,
		MultiUserSupport:     true,
		SessionPersistence:   true,
		CustomAuth:           true,
		CustomRateLimiting:   true,
		MaxConcurrentSessions: 0, // unlimited
		MinLatency:           100 * time.Millisecond,
		SupportedCodecs:      []string{"opus", "pcm"},
	}, nil
}

// CreateBackend creates a new mock backend instance.
func (p *MockProvider) CreateBackend(ctx context.Context, config *vbiface.Config) (vbiface.VoiceBackend, error) {
	mockConfig := NewMockConfig(config)
	return NewMockBackend(mockConfig)
}

// ValidateConfig validates the mock provider configuration.
func (p *MockProvider) ValidateConfig(ctx context.Context, config *vbiface.Config) error {
	// Mock provider accepts any valid base config
	return backend.ValidateConfig(config)
}

// GetConfigSchema returns the configuration schema for the mock provider.
func (p *MockProvider) GetConfigSchema() *vbiface.ConfigSchema {
	return &vbiface.ConfigSchema{
		Fields: []vbiface.ConfigField{
			{
				Name:        "provider",
				Type:        "string",
				Required:    true,
				Description: "Provider name (must be 'mock')",
			},
		},
	}
}
