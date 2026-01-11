# Provider Registry API Contract

**Date**: 2026-01-11  
**Feature**: Voice Backends (001-voice-backends)  
**Version**: 1.0.0

## Overview

This document defines the API contract for the Voice Backends provider registry. The registry follows the same patterns as `pkg/llms/`, `pkg/embeddings/`, and `pkg/multimodal/` registries.

## Registry Interface

### BackendRegistry

Global registry for backend provider management.

```go
package backend

import (
    "context"
    "sync"
)

// BackendRegistry manages backend provider registration and retrieval.
type BackendRegistry struct {
    providers map[string]BackendProvider
    mu        sync.RWMutex
}

// Global registry instance
var (
    globalRegistry *BackendRegistry
    registryOnce   sync.Once
)

// GetRegistry returns the global registry instance.
// This follows the standard pattern used across all Beluga AI packages.
// It uses sync.Once to ensure thread-safe initialization.
//
// Example:
//
//	registry := backend.GetRegistry()
//	providers := registry.ListProviders()
//	fmt.Printf("Available providers: %v\n", providers)
func GetRegistry() *BackendRegistry {
    registryOnce.Do(func() {
        globalRegistry = &BackendRegistry{
            providers: make(map[string]BackendProvider),
        }
    })
    return globalRegistry
}

// Register registers a new backend provider with the registry.
// Per FR-001: Provides registry for voice backend providers.
// Thread-safe: Uses write lock for registration.
//
// Example:
//
//	registry := backend.GetRegistry()
//	registry.Register("livekit", livekitProvider)
func (r *BackendRegistry) Register(name string, provider BackendProvider) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.providers[name] = provider
}

// Create creates a new backend instance using the registered provider.
// Per FR-002: Creates backends from registered providers using configuration.
// Per FR-016: Validates configuration before instance creation.
// Returns error if provider not found or configuration invalid.
//
// Example:
//
//	registry := backend.GetRegistry()
//	backend, err := registry.Create(ctx, "livekit", config)
func (r *BackendRegistry) Create(ctx context.Context, name string, config *Config) (VoiceBackend, error) {
    r.mu.RLock()
    provider, exists := r.providers[name]
    r.mu.RUnlock()
    
    if !exists {
        return nil, NewBackendError("Create", ErrCodeProviderNotFound,
            fmt.Errorf("backend provider '%s' not found", name))
    }
    
    // Validate config
    if err := provider.ValidateConfig(ctx, config); err != nil {
        return nil, NewBackendError("Create", ErrCodeInvalidConfig, err)
    }
    
    return provider.CreateBackend(ctx, config)
}

// ListProviders returns a list of all registered provider names.
// Per User Story 4: Developers can query available providers.
// Thread-safe: Uses read lock for listing.
//
// Example:
//
//	registry := backend.GetRegistry()
//	providers := registry.ListProviders()
//	for _, name := range providers {
//	    fmt.Printf("Provider: %s\n", name)
//	}
func (r *BackendRegistry) ListProviders() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    names := make([]string, 0, len(r.providers))
    for name := range r.providers {
        names = append(names, name)
    }
    return names
}

// IsRegistered checks if a provider is registered.
// Thread-safe: Uses read lock for checking.
//
// Example:
//
//	registry := backend.GetRegistry()
//	if registry.IsRegistered("livekit") {
//	    // Provider is available
//	}
func (r *BackendRegistry) IsRegistered(name string) bool {
    r.mu.RLock()
    defer r.mu.RUnlock()
    _, exists := r.providers[name]
    return exists
}

// GetProvider returns a provider by name.
// Returns error if provider not found.
func (r *BackendRegistry) GetProvider(name string) (BackendProvider, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    provider, exists := r.providers[name]
    if !exists {
        return nil, NewBackendError("GetProvider", ErrCodeProviderNotFound,
            fmt.Errorf("backend provider '%s' not found", name))
    }
    return provider, nil
}
```

## Provider Registration Pattern

### Auto-Registration via init.go

All providers must auto-register using `init()` functions in `providers/*/init.go` files.

**Pattern** (matching `pkg/llms/providers/*/init.go`):

```go
// pkg/voice/backend/providers/livekit/init.go
package livekit

import "github.com/lookatitude/beluga-ai/pkg/voice/backend"

func init() {
    // Register LiveKit provider with the global registry
    backend.GetRegistry().Register("livekit", NewLiveKitProvider())
}
```

```go
// pkg/voice/backend/providers/pipecat/init.go
package pipecat

import "github.com/lookatitude/beluga-ai/pkg/voice/backend"

func init() {
    // Register Pipecat provider with the global registry
    backend.GetRegistry().Register("pipecat", NewPipecatProvider())
}
```

### Provider Factory Pattern

Providers implement the `BackendProvider` interface:

```go
// pkg/voice/backend/providers/livekit/provider.go
package livekit

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/voice/backend"
)

// LiveKitProvider implements BackendProvider for LiveKit.
type LiveKitProvider struct {
    name string
}

// NewLiveKitProvider creates a new LiveKit provider.
func NewLiveKitProvider() backend.BackendProvider {
    return &LiveKitProvider{
        name: "livekit",
    }
}

// GetName returns the provider name.
func (p *LiveKitProvider) GetName() string {
    return p.name
}

// GetCapabilities returns LiveKit capabilities.
func (p *LiveKitProvider) GetCapabilities(ctx context.Context) (*backend.ProviderCapabilities, error) {
    return &backend.ProviderCapabilities{
        S2SSupport:           true,
        MultiUserSupport:     true,
        SessionPersistence:   true,
        CustomAuth:           true,
        CustomRateLimiting:   true,
        MaxConcurrentSessions: 0, // Unlimited
        MinLatency:           100 * time.Millisecond,
        SupportedCodecs:      []string{"opus", "pcm"},
    }, nil
}

// CreateBackend creates a new LiveKit backend instance.
func (p *LiveKitProvider) CreateBackend(ctx context.Context, config *backend.Config) (backend.VoiceBackend, error) {
    // Implementation
}

// ValidateConfig validates LiveKit-specific configuration.
func (p *LiveKitProvider) ValidateConfig(ctx context.Context, config *backend.Config) error {
    // Implementation
}

// GetConfigSchema returns LiveKit configuration schema.
func (p *LiveKitProvider) GetConfigSchema() *backend.ConfigSchema {
    // Implementation
}
```

## Usage Examples

### Querying Available Providers

```go
registry := backend.GetRegistry()
providers := registry.ListProviders()
fmt.Printf("Available providers: %v\n", providers)
// Output: Available providers: [livekit pipecat vocode vapi cartesia]
```

### Checking Provider Registration

```go
registry := backend.GetRegistry()
if registry.IsRegistered("livekit") {
    fmt.Println("LiveKit provider is available")
}
```

### Creating Backend from Registry

```go
registry := backend.GetRegistry()
backend, err := registry.Create(ctx, "livekit", &backend.Config{
    Provider: "livekit",
    // ... config
})
if err != nil {
    log.Fatal(err)
}
```

### Provider Swapping (Zero Code Changes)

```go
// Application code (provider-agnostic)
func createBackend(providerName string, config *backend.Config) (backend.VoiceBackend, error) {
    registry := backend.GetRegistry()
    return registry.Create(ctx, providerName, config)
}

// Switch from LiveKit to Pipecat - only config change needed
backend1, _ := createBackend("livekit", livekitConfig)
backend2, _ := createBackend("pipecat", pipecatConfig) // Same code, different provider
```

## Thread Safety

All registry operations are thread-safe:
- `Register()`: Uses write lock (`mu.Lock()`)
- `Create()`: Uses read lock for lookup, then provider handles creation
- `ListProviders()`: Uses read lock (`mu.RLock()`)
- `IsRegistered()`: Uses read lock (`mu.RLock()`)
- `GetProvider()`: Uses read lock (`mu.RLock()`)

## Error Handling

Registry operations return `BackendError` with appropriate error codes:
- `ErrCodeProviderNotFound`: Provider not registered
- `ErrCodeInvalidConfig`: Configuration validation failed

## Compliance

This registry API contract complies with:
- ✅ FR-001: Registry for voice backend providers
- ✅ FR-002: Creating backends from registered providers
- ✅ FR-016: Configuration validation before creation
- ✅ User Story 4: Provider swapping without code changes
- ✅ Constitution III: Provider Registry Pattern (matches `pkg/llms/`, `pkg/embeddings/`, `pkg/multimodal/`)
