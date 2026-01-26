# Two-Tier Factory Pattern

## Purpose

Wrapper packages use a two-tier factory pattern that delegates provider creation to sub-package registries. This enables unified creation while maintaining sub-package independence.

## Pattern Structure

```
Tier 1: Facade Factory (wrapper package)
    └── Tier 2: Sub-Package Registries
            ├── STT Registry
            ├── TTS Registry
            └── VAD Registry
```

## Implementation

### Tier 1: Facade Factory

```go
// pkg/voice/factory.go
package voice

import (
    "context"
    "fmt"
    "sync"

    "github.com/lookatitude/beluga-ai/pkg/voice/stt"
    "github.com/lookatitude/beluga-ai/pkg/voice/tts"
    "github.com/lookatitude/beluga-ai/pkg/voice/vad"
)

type VoiceFactory struct {
    sttRegistry *stt.Registry
    ttsRegistry *tts.Registry
    vadRegistry *vad.Registry
    mu          sync.RWMutex
}

var (
    globalFactory *VoiceFactory
    factoryOnce   sync.Once
)

func GetFactory() *VoiceFactory {
    factoryOnce.Do(func() {
        globalFactory = &VoiceFactory{
            sttRegistry: stt.GetRegistry(),
            ttsRegistry: tts.GetRegistry(),
            vadRegistry: vad.GetRegistry(),
        }
    })
    return globalFactory
}

func (f *VoiceFactory) CreateVoiceAgent(ctx context.Context, cfg *Config) (VoiceAgent, error) {
    f.mu.RLock()
    defer f.mu.RUnlock()

    // Delegate to sub-package registries
    sttProvider, err := f.sttRegistry.GetProvider(cfg.STT.Provider, &cfg.STT)
    if err != nil {
        return nil, fmt.Errorf("create stt provider '%s': %w", cfg.STT.Provider, err)
    }

    ttsProvider, err := f.ttsRegistry.GetProvider(cfg.TTS.Provider, &cfg.TTS)
    if err != nil {
        return nil, fmt.Errorf("create tts provider '%s': %w", cfg.TTS.Provider, err)
    }

    vadProvider, err := f.vadRegistry.GetProvider(cfg.VAD.Provider, &cfg.VAD)
    if err != nil {
        return nil, fmt.Errorf("create vad provider '%s': %w", cfg.VAD.Provider, err)
    }

    return &voiceAgent{
        stt:    sttProvider,
        tts:    ttsProvider,
        vad:    vadProvider,
        config: cfg,
    }, nil
}
```

### Tier 2: Sub-Package Registry

```go
// pkg/voice/stt/registry.go
package stt

import (
    "fmt"
    "sync"

    "github.com/lookatitude/beluga-ai/pkg/voice/stt/iface"
)

type Registry struct {
    factories map[string]func(*Config) (iface.Transcriber, error)
    mu        sync.RWMutex
}

var (
    globalRegistry *Registry
    registryOnce   sync.Once
)

func GetRegistry() *Registry {
    registryOnce.Do(func() {
        globalRegistry = &Registry{
            factories: make(map[string]func(*Config) (iface.Transcriber, error)),
        }
    })
    return globalRegistry
}

func (r *Registry) Register(name string, factory func(*Config) (iface.Transcriber, error)) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.factories[name] = factory
}

func (r *Registry) GetProvider(name string, cfg *Config) (iface.Transcriber, error) {
    r.mu.RLock()
    factory, exists := r.factories[name]
    r.mu.RUnlock()

    if !exists {
        return nil, fmt.Errorf("stt provider '%s' not found", name)
    }

    return factory(cfg)
}

func (r *Registry) ListProviders() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()

    names := make([]string, 0, len(r.factories))
    for name := range r.factories {
        names = append(names, name)
    }
    return names
}
```

### Provider Auto-Registration

```go
// pkg/voice/stt/providers/deepgram/init.go
package deepgram

import "github.com/lookatitude/beluga-ai/pkg/voice/stt"

func init() {
    stt.GetRegistry().Register("deepgram", NewDeepgramTranscriber)
}

func NewDeepgramTranscriber(cfg *stt.Config) (stt.iface.Transcriber, error) {
    // Provider implementation
    return &DeepgramTranscriber{
        apiKey:     cfg.APIKey,
        model:      cfg.Model,
        sampleRate: cfg.SampleRate,
    }, nil
}
```

## Convenience Functions

```go
// pkg/voice/voice.go
package voice

// Convenience function using global factory
func NewVoiceAgent(ctx context.Context, cfg *Config) (VoiceAgent, error) {
    return GetFactory().CreateVoiceAgent(ctx, cfg)
}

// With functional options
func NewVoiceAgentWithOptions(ctx context.Context, opts ...VoiceOption) (VoiceAgent, error) {
    cfg := DefaultConfig()
    for _, opt := range opts {
        opt(cfg)
    }
    return GetFactory().CreateVoiceAgent(ctx, cfg)
}
```

## Error Handling

```go
func (f *VoiceFactory) CreateVoiceAgent(ctx context.Context, cfg *Config) (VoiceAgent, error) {
    // Validate config first
    if err := cfg.Validate(); err != nil {
        return nil, &Error{
            Op:   "CreateVoiceAgent",
            Code: ErrInvalidConfig,
            Err:  err,
        }
    }

    // Create with detailed error context
    sttProvider, err := f.sttRegistry.GetProvider(cfg.STT.Provider, &cfg.STT)
    if err != nil {
        return nil, &Error{
            Op:   "CreateVoiceAgent",
            Code: ErrProviderNotFound,
            Err:  fmt.Errorf("stt provider '%s': %w", cfg.STT.Provider, err),
        }
    }

    // ... continue with other providers
}
```

## Testing

```go
func TestTwoTierFactory(t *testing.T) {
    // Register mock providers
    stt.GetRegistry().Register("mock", stt.NewMockTranscriber)
    tts.GetRegistry().Register("mock", tts.NewMockSpeaker)
    vad.GetRegistry().Register("mock", vad.NewMockDetector)

    // Create via factory
    cfg := &voice.Config{
        STT: stt.Config{Provider: "mock"},
        TTS: tts.Config{Provider: "mock"},
        VAD: vad.Config{Provider: "mock"},
    }

    agent, err := voice.NewVoiceAgent(ctx, cfg)
    require.NoError(t, err)
    require.NotNil(t, agent)
}
```

## Related Standards

- [registry-shape.md](./registry-shape.md) - Standard registry pattern
- [factory-signature.md](./factory-signature.md) - Factory function signatures
- [../global/wrapper-package-pattern.md](../global/wrapper-package-pattern.md) - Wrapper patterns
