# Standards Applied: Package Flattening

## global/naming
- **Rule**: Use plural forms for multi-provider packages, lowercase only
- **Application**: New packages follow this (stt, tts, vad already plural-like abbreviations)

## global/subpackage-structure
- **Rule**: Sub-packages should be independently importable
- **Application**: Promoting voice/* to pkg/* makes each fully independent
- **Before**: `import "pkg/voice/stt"` implicitly tied to voice parent
- **After**: `import "pkg/stt"` standalone package

## backend/registry-shape
- **Rule**: Registry should be at package root as `registry.go`, not in subdirectory
- **Application**: Moving `pkg/embeddings/registry/` to `pkg/embeddings/registry.go`
- **Pattern**:
```go
// pkg/embeddings/registry.go
var globalRegistry = make(map[string]ProviderFactory)

func Register(name string, factory ProviderFactory) { ... }
func New(ctx context.Context, name string, cfg Config) (Embedder, error) { ... }
```

## global/internal-vs-providers
- **Rule**: Use `internal/` only for implementation details that shouldn't be exposed
- **Application**:
  - `voicebackend/internal/` kept - LiveKit SDK wrapper complexity
  - `voicesession/internal/` kept - ~40 orchestration files
  - `voiceutils/audio/` promoted from internal - utilities useful to consumers

## global/wrapper-package-pattern
- **Rule**: Wrapper packages re-export subpackage functionality for convenience
- **Application**: `pkg/voice/deprecated.go` provides shims:
```go
// Deprecated: Use github.com/lookatitude/beluga-ai/pkg/stt instead.
type STTConfig = stt.Config
var NewSTTProvider = stt.NewProvider
```

## Package Independence Principle
Each promoted package should:
1. Have its own `iface/` directory if it defines interfaces
2. Not import parent package (no `pkg/voice` imports)
3. Import only `voiceutils` for shared types
4. Have complete registry pattern if multi-provider
