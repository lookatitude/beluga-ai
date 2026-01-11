// Package internal provides internal implementation details for the multimodal package.
package internal

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/multimodal/registry"
	"github.com/lookatitude/beluga-ai/pkg/multimodal/types"
)

// Router handles routing content blocks to appropriate providers based on capabilities.
type Router struct {
	registry *registry.ProviderRegistry
}

// NewRouter creates a new router.
func NewRouter(reg *registry.ProviderRegistry) *Router {
	return &Router{
		registry: reg,
	}
}

// Route determines which provider should handle each content block.
func (r *Router) Route(ctx context.Context, input *types.MultimodalInput) (map[string]string, error) {
	if input == nil {
		return nil, fmt.Errorf("Route: input cannot be nil")
	}

	if len(input.ContentBlocks) == 0 {
		return nil, fmt.Errorf("Route: input must have at least one content block")
	}

	// If routing config is provided, use it
	if input.Routing != nil {
		return r.routeWithConfig(ctx, input, input.Routing)
	}

	// Otherwise, use auto-routing
	return r.routeAuto(ctx, input)
}

// routeWithConfig routes content blocks according to the routing configuration.
func (r *Router) routeWithConfig(ctx context.Context, input *types.MultimodalInput, routingMap map[string]any) (map[string]string, error) {
	routing := make(map[string]string)
	strategy := getStringFromMap(routingMap, "strategy", "auto")

	switch strategy {
	case "manual":
		// Use explicitly specified providers
		fallbackToText := getBoolFromMap(routingMap, "fallback_to_text", true)
		for i, block := range input.ContentBlocks {
			provider := r.getProviderForModality(routingMap, block.Type)
			if provider == "" {
				if fallbackToText {
					// Fallback to text-only processing
					provider = r.findTextProvider(ctx)
				} else {
					return nil, fmt.Errorf("routeWithConfig: no provider specified for %s content and fallback disabled", block.Type)
				}
			}
			routing[fmt.Sprintf("%d", i)] = provider
		}
	case "fallback":
		// Try to find providers, fallback to text if not found
		fallbackToText := getBoolFromMap(routingMap, "fallback_to_text", true)
		for i, block := range input.ContentBlocks {
			provider := r.findProviderForModality(ctx, block.Type)
			if provider == "" {
				if fallbackToText {
					provider = r.findTextProvider(ctx)
				} else {
					return nil, fmt.Errorf("routeWithConfig: no provider found for %s content and fallback disabled", block.Type)
				}
			}
			routing[fmt.Sprintf("%d", i)] = provider
		}
	default: // "auto"
		return r.routeAuto(ctx, input)
	}

	return routing, nil
}

// getStringFromMap extracts a string value from a map.
func getStringFromMap(m map[string]any, key string, defaultValue string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultValue
}

// getBoolFromMap extracts a bool value from a map.
func getBoolFromMap(m map[string]any, key string, defaultValue bool) bool {
	if val, ok := m[key].(bool); ok {
		return val
	}
	return defaultValue
}

// routeAuto automatically routes content blocks to appropriate providers.
func (r *Router) routeAuto(ctx context.Context, input *types.MultimodalInput) (map[string]string, error) {
	routing := make(map[string]string)

	for i, block := range input.ContentBlocks {
		provider := r.findProviderForModality(ctx, block.Type)
		if provider == "" {
			// Fallback to text-only if no provider found
			provider = r.findTextProvider(ctx)
			if provider == "" {
				return nil, fmt.Errorf("routeAuto: no provider found for any modality")
			}
		}
		routing[fmt.Sprintf("%d", i)] = provider
	}

	return routing, nil
}

// getProviderForModality gets the provider for a modality from routing config.
func (r *Router) getProviderForModality(routingMap map[string]any, modality string) string {
	switch modality {
	case "text":
		return getStringFromMap(routingMap, "text_provider", "")
	case "image":
		return getStringFromMap(routingMap, "image_provider", "")
	case "audio":
		return getStringFromMap(routingMap, "audio_provider", "")
	case "video":
		return getStringFromMap(routingMap, "video_provider", "")
	default:
		return ""
	}
}

// findProviderForModality finds a provider that supports the given modality.
func (r *Router) findProviderForModality(ctx context.Context, modality string) string {
	providers := r.registry.ListProviders()

	for _, providerName := range providers {
		// Check if provider supports this modality
		// This is a simplified check - in production, you'd query the provider's capabilities
		if r.supportsModality(ctx, providerName, modality) {
			return providerName
		}
	}

	return ""
}

// supportsModality checks if a provider supports a specific modality.
func (r *Router) supportsModality(ctx context.Context, providerName, modality string) bool {
	// This is a simplified implementation
	// In production, you'd create a model instance and check its capabilities
	// For now, we'll use a basic heuristic based on provider names
	knownMultimodalProviders := []string{"openai", "google", "anthropic", "xai"}
	for _, p := range knownMultimodalProviders {
		if providerName == p {
			return true
		}
	}

	// Default: assume text-only providers support text
	return modality == "text"
}

// findTextProvider finds any provider that supports text (most providers do).
func (r *Router) findTextProvider(ctx context.Context) string {
	providers := r.registry.ListProviders()
	if len(providers) > 0 {
		return providers[0] // Return first available provider
	}
	return ""
}

// CheckCapability checks if a provider supports a specific modality.
func (r *Router) CheckCapability(ctx context.Context, providerName, modality string) (bool, error) {
	// Check if provider is registered
	if !r.registry.IsRegistered(providerName) {
		return false, fmt.Errorf("CheckCapability: provider '%s' not found", providerName)
	}

	// Create a temporary config to check capabilities
	// In production, you'd want a more efficient way to check capabilities
	// Use a map to avoid importing multimodal.Config
	config := map[string]any{
		"Provider": providerName,
		"Model":    "default", // Use default model
	}

	model, err := r.registry.Create(ctx, providerName, config)
	if err != nil {
		return false, fmt.Errorf("CheckCapability: %w", err)
	}

	return model.SupportsModality(ctx, modality)
}
