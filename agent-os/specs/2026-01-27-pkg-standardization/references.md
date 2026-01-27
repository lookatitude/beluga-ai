# Package Standardization - Reference Implementations

## Registry Patterns

### Pattern 1: Root registry.go (Preferred)

**Example**: `pkg/llms/registry.go`

```go
// registry.go at package root
var (
    globalRegistry = NewProviderRegistry()
    globalLLMRegistry = NewLLMRegistry()
)

func RegisterGlobal(name string, creator CreatorFunc) {
    globalRegistry.Register(name, creator)
}

func GetProvider(ctx context.Context, name string, config *Config) (iface.ChatModel, error) {
    return globalRegistry.Create(ctx, name, config)
}

func ListProviders() []string {
    return globalRegistry.ListProviders()
}

func IsRegistered(name string) bool {
    return globalRegistry.IsRegistered(name)
}
```

**Used by**: llms, agents, memory, vectorstores

### Pattern 2: registry/ Subdirectory with Wrapper

**Example**: `pkg/chatmodels/`

```
pkg/chatmodels/
├── registry/
│   └── registry.go    # Internal registry implementation
└── registry.go        # Wrapper that exposes GetRegistry()
```

**pkg/chatmodels/registry.go** (wrapper):
```go
package chatmodels

import "github.com/lookatitude/beluga-ai/pkg/chatmodels/registry"

// GetRegistry returns the global registry
func GetRegistry() *registry.Registry {
    return registry.GetGlobalRegistry()
}
```

**Used by**: chatmodels, embeddings, multimodal

**Rationale**: Avoids import cycles when iface/ types reference registry types.

### Pattern 3: GetRegistry() in factory.go

**Example**: `pkg/embeddings/factory.go`

```go
// factory.go includes registry access
func GetRegistry() *registry.ProviderRegistry {
    return globalProviderRegistry
}
```

**Used by**: embeddings

## internal/ Directory Usage

### When to Use internal/

1. **Complex base implementations**
   - `pkg/agents/internal/base/` - Base agent types

2. **Shared utilities across providers**
   - `pkg/llms/internal/common/` - Common provider utilities

3. **Mock implementations for testing**
   - `pkg/chatmodels/internal/mock/` - Test mocks

### When NOT to Use internal/

- Package is simple and doesn't need hidden implementation details
- All types are meant for public API
- Empty directories (never create empty internal/)

## Test Utils Patterns

### Pattern 1: test_utils.go at Root

```go
// pkg/llms/test_utils.go
package llms

type MockLLM struct {
    mock.Mock
}

func NewMockLLM() *MockLLM {
    return &MockLLM{}
}
```

### Pattern 2: internal/mock/ for Complex Mocks

```go
// pkg/chatmodels/internal/mock/chatmodel.go
package mock

type ChatModel struct {
    mock.Mock
}
```

## Package Structure Examples

### Standard Multi-Provider Package

```
pkg/llms/
├── iface/
│   ├── llm.go
│   └── errors.go
├── internal/
│   └── common/
│       └── retry.go
├── providers/
│   ├── openai/
│   ├── anthropic/
│   └── ollama/
├── config.go
├── errors.go
├── factory.go
├── llms.go
├── metrics.go
├── registry.go
├── test_utils.go
├── advanced_test.go
└── README.md
```

### Wrapper Package (voice)

```
pkg/voice/
├── iface/
│   └── voice.go
├── stt/
│   ├── iface/
│   ├── providers/
│   └── registry.go
├── tts/
│   ├── iface/
│   ├── providers/
│   └── registry.go
├── vad/
│   ├── iface/
│   ├── providers/
│   └── registry.go
├── config.go
├── errors.go
├── metrics.go
├── registry.go           # Facade delegating to sub-package registries
├── voice.go
├── test_utils.go
├── advanced_test.go
└── README.md
```

## File Locations

### Reference Files in Codebase

| Pattern | File |
|---------|------|
| Root registry.go | `/pkg/llms/registry.go` |
| Registry wrapper | `/pkg/chatmodels/registry.go` |
| GetRegistry in factory | `/pkg/embeddings/factory.go` |
| test_utils.go | `/pkg/llms/test_utils.go` |
| internal/mock/ | `/pkg/chatmodels/internal/mock/` |
| Wrapper facade | `/pkg/voice/voice.go` |
| Sub-package registry | `/pkg/voice/stt/registry.go` |
