// Package iface defines interfaces for the multimodal registry.
package iface

import (
	"context"

	"github.com/lookatitude/beluga-ai/pkg/multimodal/iface"
)

// MultimodalModelFactory is a function type for creating multimodal model instances.
type MultimodalModelFactory func(ctx context.Context, config any) (iface.MultimodalModel, error)

// Registry defines the interface for multimodal provider registration and retrieval.
type Registry interface {
	// Register registers a new multimodal provider with the registry.
	Register(name string, creator MultimodalModelFactory)

	// Create creates a new multimodal model instance using the registered provider.
	Create(ctx context.Context, name string, config any) (iface.MultimodalModel, error)

	// ListProviders returns a list of all registered provider names.
	ListProviders() []string

	// IsRegistered checks if a provider is registered.
	IsRegistered(name string) bool
}
