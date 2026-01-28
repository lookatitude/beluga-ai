// Package iface defines the registry interface for template engine providers.
// This contains factory types and registry interfaces used by providers
// to register themselves without importing the main prompts package.
package iface

import (
	"context"
)

// TemplateEngineFactory defines the function signature for creating template engines.
// This type is used by providers to register themselves with the registry.
// The config parameter is `any` to avoid import cycles - providers should
// assert it to prompts.Config when implementing.
type TemplateEngineFactory func(ctx context.Context, config any) (TemplateEngine, error)

// TemplateRegistry defines the interface for template engine provider registration.
// Implementations of this interface manage provider registration and creation.
type TemplateRegistry interface {
	// Register registers a provider factory function with the given name.
	Register(name string, factory TemplateEngineFactory)

	// Create creates a new template engine instance using the registered provider factory.
	Create(ctx context.Context, name string, config any) (TemplateEngine, error)

	// ListProviders returns a list of all registered provider names.
	ListProviders() []string

	// IsRegistered checks if a provider is registered.
	IsRegistered(name string) bool
}
