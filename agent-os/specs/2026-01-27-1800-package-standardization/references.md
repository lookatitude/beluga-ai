# Reference Implementations

## pkg/embeddings - Registry Pattern Reference

### iface/registry.go
```go
// Package iface defines the registry interface for embedder providers.
package iface

import "context"

// EmbedderFactory defines the function signature for creating embedders.
type EmbedderFactory func(ctx context.Context, config any) (Embedder, error)

// Registry defines the interface for embedder provider registration.
type Registry interface {
    Register(name string, factory EmbedderFactory)
    Create(ctx context.Context, name string, config any) (Embedder, error)
    ListProviders() []string
    IsRegistered(name string) bool
}
```

### registry.go
```go
package embeddings

import (
    "context"
    "sync"
    "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
)

type ProviderFactory = iface.EmbedderFactory

type ProviderRegistry struct {
    providers map[string]ProviderFactory
    mu        sync.RWMutex
}

var (
    globalRegistry *ProviderRegistry
    registryOnce   sync.Once
)

func GetRegistry() *ProviderRegistry {
    registryOnce.Do(func() {
        globalRegistry = &ProviderRegistry{
            providers: make(map[string]ProviderFactory),
        }
    })
    return globalRegistry
}

func (r *ProviderRegistry) Register(name string, factory ProviderFactory) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.providers[name] = factory
}

func (r *ProviderRegistry) Create(ctx context.Context, name string, config any) (iface.Embedder, error) {
    r.mu.RLock()
    factory, exists := r.providers[name]
    r.mu.RUnlock()

    if !exists {
        return nil, NewProviderNotFoundError("create_embedder", name)
    }

    return factory(ctx, config)
}

func (r *ProviderRegistry) ListProviders() []string {
    r.mu.RLock()
    defer r.mu.RUnlock()

    names := make([]string, 0, len(r.providers))
    for name := range r.providers {
        names = append(names, name)
    }
    return names
}

func (r *ProviderRegistry) IsRegistered(name string) bool {
    r.mu.RLock()
    defer r.mu.RUnlock()
    _, exists := r.providers[name]
    return exists
}

// Global convenience functions
func Register(name string, factory ProviderFactory) {
    GetRegistry().Register(name, factory)
}

func Create(ctx context.Context, name string, config any) (iface.Embedder, error) {
    return GetRegistry().Create(ctx, name, config)
}

func ListProviders() []string {
    return GetRegistry().ListProviders()
}

var _ iface.Registry = (*ProviderRegistry)(nil)
```

## pkg/vectorstores/providers/inmemory - Provider init.go Reference

```go
package inmemory

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
    vectorstoresiface "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

func init() {
    vectorstores.GetRegistry().Register("inmemory", func(ctx context.Context, config vectorstoresiface.Config) (vectorstores.VectorStore, error) {
        localConfig := Config{
            Embedder:       config.Embedder,
            SearchK:        config.SearchK,
            ScoreThreshold: config.ScoreThreshold,
        }
        store, err := NewInMemoryVectorStoreFromConfig(ctx, localConfig)
        if err != nil {
            return nil, err
        }
        return &vectorStoreWrapper{store: store.(*InMemoryVectorStore)}, nil
    })
}
```

## pkg/chatmodels - Provider Structure Reference

```
pkg/chatmodels/
├── iface/
│   ├── interfaces.go    # ChatModel interface
│   └── registry.go      # Registry interface
├── providers/
│   ├── anthropic/
│   │   ├── anthropic.go
│   │   ├── config.go
│   │   └── init.go
│   ├── openai/
│   │   ├── openai.go
│   │   ├── config.go
│   │   └── init.go
│   └── ...
├── chatmodels.go        # Main API
├── config.go            # Package config
├── errors.go            # Error types
├── metrics.go           # OTEL metrics
├── registry.go          # Registry implementation
└── README.md
```

## pkg/tools - Alternative Registry Reference

The tools package uses a slightly different pattern with explicit tool types:

```go
// registry.go
type ToolFactory func(ctx context.Context, config any) (iface.Tool, error)

type ToolRegistry struct {
    factories map[string]ToolFactory
    mu        sync.RWMutex
}

// Global singleton
var globalToolRegistry = &ToolRegistry{
    factories: make(map[string]ToolFactory),
}

func GetRegistry() *ToolRegistry {
    return globalToolRegistry
}
```

This pattern is simpler (no sync.Once) but still thread-safe. The retrievers package will use the sync.Once pattern for consistency with embeddings.
