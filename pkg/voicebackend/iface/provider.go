package iface

import (
	"context"
)

// ConfigSchema represents the configuration schema for a provider.
type ConfigSchema struct {
	Fields []ConfigField `json:"fields"`
}

// ConfigField represents a single configuration field.
type ConfigField struct {
	Default     any    `json:"default,omitempty"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// BackendProvider defines the interface for voice backend provider implementations.
// Providers are responsible for creating backend instances and managing provider-specific configuration.
type BackendProvider interface {
	// GetName returns the name of this provider (e.g., "livekit", "pipecat").
	GetName() string

	// GetCapabilities returns the capabilities of this provider for different modalities.
	GetCapabilities(ctx context.Context) (*ProviderCapabilities, error)

	// CreateBackend creates a new voice backend instance with the given configuration.
	CreateBackend(ctx context.Context, config *Config) (VoiceBackend, error)

	// ValidateConfig validates the provider-specific configuration.
	ValidateConfig(ctx context.Context, config *Config) error

	// GetConfigSchema returns the configuration schema for this provider.
	GetConfigSchema() *ConfigSchema
}
