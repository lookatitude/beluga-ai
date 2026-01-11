// Package iface defines interfaces for multimodal provider operations.
package iface

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
)

// MultimodalProvider defines the interface for multimodal provider implementations.
// Providers are responsible for creating model instances and managing provider-specific configuration.
type MultimodalProvider interface {
	// CreateModel creates a new multimodal model instance with the given configuration.
	// Config is passed as any to avoid import cycles - providers should extract their specific config.
	CreateModel(ctx context.Context, config any) (MultimodalModel, error)

	// GetName returns the name of this provider (e.g., "openai", "google", "anthropic").
	GetName() string

	// GetCapabilities returns the capabilities of this provider for different modalities.
	GetCapabilities(ctx context.Context) (*types.ModalityCapabilities, error)

	// ValidateConfig validates the provider-specific configuration.
	// Config is passed as any to avoid import cycles - providers should extract their specific config.
	ValidateConfig(ctx context.Context, config any) error
}
