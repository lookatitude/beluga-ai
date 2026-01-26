# Wrapper/Aggregation Package Pattern

## Purpose

Wrapper packages serve as high-level bridges that compose and orchestrate functionality from multiple sub-packages. They provide unified entry points for complex features while hiding implementation details.

## When to Use

Use wrapper packages when:
- Aggregating 2+ related sub-packages into a cohesive unit
- Providing a facade interface for complex multi-component functionality
- Coordinating cross-sub-package orchestration
- Implementing unified observability across sub-packages

## Pattern Definition

### Characteristics

1. **Facade Interface**: Expose a single entry point that delegates to sub-components
2. **Sub-Package Composition**: Use DI to compose sub-packages via functional options
3. **Cross-Package Orchestration**: Handle coordination between sub-packages
4. **Error Propagation**: Aggregate errors from sub-packages with context
5. **Span Aggregation**: Create parent spans that contain child spans from sub-packages

### Required Structure

```
pkg/{wrapper_package}/
├── iface/                    # Shared interfaces across sub-packages
├── {subpackage1}/           # Independent sub-package
│   ├── iface/               # Sub-package interfaces
│   ├── providers/           # Sub-package providers
│   ├── config.go            # Sub-package config
│   ├── metrics.go           # Sub-package metrics
│   ├── errors.go            # Sub-package errors
│   ├── registry.go          # Sub-package registry
│   └── test_utils.go        # Sub-package test utilities
├── {subpackage2}/           # Another independent sub-package
├── config.go                # Root config embedding sub-package configs
├── metrics.go               # Aggregated metrics from sub-packages
├── errors.go                # Error definitions
├── registry.go              # Facade registry delegating to sub-packages
├── {package_name}.go        # Facade API
├── test_utils.go            # Composite mocks for all sub-packages
└── README.md                # Documentation
```

## Implementation

### Facade Interface

```go
// voice.go - Facade API
package voice

type VoiceAgent interface {
    StartSession(ctx context.Context, cfg *SessionConfig) (Session, error)
    ProcessAudio(ctx context.Context, audio []byte) (*ProcessResult, error)
    Close() error
}

func NewVoiceAgent(opts ...VoiceOption) (VoiceAgent, error) {
    cfg := defaultConfig()
    for _, opt := range opts {
        opt(cfg)
    }

    // Initialize sub-packages from config
    sttProvider, err := stt.GetRegistry().GetProvider(cfg.STT.Provider, &cfg.STT)
    if err != nil {
        return nil, fmt.Errorf("stt provider: %w", err)
    }

    ttsProvider, err := tts.GetRegistry().GetProvider(cfg.TTS.Provider, &cfg.TTS)
    if err != nil {
        return nil, fmt.Errorf("tts provider: %w", err)
    }

    return &voiceAgent{
        stt:    sttProvider,
        tts:    ttsProvider,
        config: cfg,
    }, nil
}
```

### Facade Registry

```go
// registry.go - Facade registry delegating to sub-packages
package voice

type Registry struct {
    STT       *stt.Registry
    TTS       *tts.Registry
    VAD       *vad.Registry
    Transport *transport.Registry
}

var (
    globalRegistry *Registry
    registryOnce   sync.Once
)

func GetRegistry() *Registry {
    registryOnce.Do(func() {
        globalRegistry = &Registry{
            STT:       stt.GetRegistry(),
            TTS:       tts.GetRegistry(),
            VAD:       vad.GetRegistry(),
            Transport: transport.GetRegistry(),
        }
    })
    return globalRegistry
}

func (r *Registry) ListAllProviders() map[string][]string {
    return map[string][]string{
        "stt":       r.STT.ListProviders(),
        "tts":       r.TTS.ListProviders(),
        "vad":       r.VAD.ListProviders(),
        "transport": r.Transport.ListProviders(),
    }
}
```

## Examples in Framework

### voice Package
- Aggregates: `stt`, `tts`, `vad`, `session`, `transport`, `noise`, `turndetection`
- Facade: `VoiceAgent` interface
- Use case: End-to-end voice processing pipelines

### orchestration Package
- Aggregates: `chains`, `graphs`, `workflows`
- Facade: `WorkflowEngine` interface
- Use case: DAG-based workflow execution

## Anti-Patterns

### Avoid: Tight Coupling Between Sub-Packages

```go
// Bad: Sub-package directly imports sibling
package stt

import "github.com/lookatitude/beluga-ai/pkg/voice/tts"  // Don't do this

func (t *Transcriber) ProcessWithTTS() {
    tts.Synthesize(...)  // Tight coupling
}
```

### Avoid: Leaking Internal Sub-Package Types

```go
// Bad: Exposing internal sub-package types in facade
type VoiceResult struct {
    STTInternal *stt.internalResult  // Don't expose internal types
}

// Good: Use interfaces or wrapper types
type VoiceResult struct {
    Transcription TranscriptionResult  // Public wrapper type
}
```

## Related Standards

- [subpackage-structure.md](./subpackage-structure.md) - Sub-package independence requirements
- [config-propagation.md](./config-propagation.md) - Hierarchical config patterns
- [backend/span-aggregation.md](../backend/span-aggregation.md) - OTEL span aggregation
- [backend/two-tier-factory.md](../backend/two-tier-factory.md) - Two-tier factory pattern
