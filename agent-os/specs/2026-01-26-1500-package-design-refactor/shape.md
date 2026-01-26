# Package Design Patterns Refactor - Shape Notes

## Context

The Beluga AI Framework has 21 packages in `pkg/` with varying levels of compliance to the standardized package structure. While the framework is production-ready, several patterns need formalization:

1. **Wrapper/Aggregation Packages**: The `voice` and `orchestration` packages serve as wrappers around sub-packages, but this pattern isn't formally documented.

2. **Sub-Package Structure**: Some packages (llms, embeddings, voice) use sub-packages well, but the convention isn't standardized across all packages.

3. **Registry Patterns**: Not all multi-provider packages have proper registries.

4. **Safety Package**: Currently minimal, needs full structure compliance.

## Key Insights from Analysis

### Wrapper Package Pattern (voice as exemplar)

```
pkg/voice/
├── iface/                    # Shared interfaces
├── backend/                  # Backend orchestration
├── stt/                      # Speech-to-Text sub-package
├── tts/                      # Text-to-Speech sub-package
├── vad/                      # Voice Activity Detection
├── session/                  # Session management
├── transport/                # Audio transport
├── noise/                    # Noise cancellation
└── turndetection/            # Turn detection
```

**Pattern characteristics:**
- Facade interface at root level
- Sub-packages are independently importable
- Config propagation from parent to sub-packages
- Span aggregation for observability

### Registry Pattern (llms as exemplar)

```go
var (
    globalRegistry *Registry
    registryOnce   sync.Once
)

type Registry struct {
    providerFactories map[string]func(*Config) (iface.ChatModel, error)
    mu                sync.RWMutex
}

func GetRegistry() *Registry {
    registryOnce.Do(func() {
        globalRegistry = &Registry{
            providerFactories: make(map[string]func(*Config) (iface.ChatModel, error)),
        }
    })
    return globalRegistry
}
```

### Config Propagation Pattern

```go
type VoiceConfig struct {
    STT  stt.Config  `yaml:"stt"`
    TTS  tts.Config  `yaml:"tts"`
    VAD  vad.Config  `yaml:"vad"`
}

func NewVoiceAgent(cfg *VoiceConfig) (*VoiceAgent, error) {
    sttProvider, _ := stt.GetRegistry().GetProvider(cfg.STT.Provider, &cfg.STT)
    ttsProvider, _ := tts.GetRegistry().GetProvider(cfg.TTS.Provider, &cfg.TTS)
    // ...
}
```

## Non-Compliant Packages

### safety/ (Critical)
- No `iface/` directory
- No `config.go`
- No `metrics.go`
- No `registry.go`
- No `test_utils.go`
- No `advanced_test.go`

### Missing Registries
- vectorstores/
- prompts/
- retrievers/
- orchestration/
- server/

## Design Decisions

1. **Wrapper vs Regular Package**: A package is a "wrapper" if it:
   - Aggregates 2+ sub-packages
   - Provides facade interface
   - Handles cross-sub-package orchestration

2. **Sub-Package Independence**: Sub-packages MUST be independently importable and testable.

3. **Registry Requirement**: All multi-provider packages MUST have registries.

4. **OTEL Requirement**: All packages MUST have metrics.go with standard metrics pattern.

5. **Config Propagation**: Parent configs embed sub-package configs using YAML tags.

## Risk Assessment

- **Low Risk**: Documentation and standards updates
- **Medium Risk**: safety/ package refactor (minimal existing code)
- **Higher Risk**: Adding registries to existing packages (potential breaking changes)

## Success Criteria

1. All packages follow standard structure
2. All multi-provider packages have registries
3. All packages have OTEL metrics
4. All tests pass
5. Documentation is complete and accurate
