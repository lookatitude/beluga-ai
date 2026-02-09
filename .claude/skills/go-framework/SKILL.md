---
name: go-framework
description: Go framework design patterns and idiomatic structure for Beluga AI v2. Use when designing package structure, creating new packages, or implementing framework-level patterns like registries, factories, and lifecycle management.
---

# Go Framework Design Patterns for Beluga AI v2

## Package Structure Pattern

Every extensible package in Beluga follows this structure:
```
<package>/
├── <interface>.go     # Extension contract (Go interface)
├── registry.go        # Register(), New(), List()
├── hooks.go           # Lifecycle hook types
├── middleware.go       # Middleware type + Apply()
└── providers/          # Built-in implementations
    ├── <provider_a>/
    └── <provider_b>/
```

## Registry + Factory Pattern

```go
// Type-safe factory
type Factory func(cfg ProviderConfig) (Interface, error)

var (
    mu       sync.RWMutex
    registry = make(map[string]Factory)
)

func Register(name string, f Factory) {
    mu.Lock()
    defer mu.Unlock()
    if _, exists := registry[name]; exists {
        panic(fmt.Sprintf("provider %q already registered", name))
    }
    registry[name] = f
}

func New(name string, cfg ProviderConfig) (Interface, error) {
    mu.RLock()
    f, ok := registry[name]
    mu.RUnlock()
    if !ok {
        return nil, fmt.Errorf("unknown provider %q (registered: %v)", name, List())
    }
    return f(cfg)
}

func List() []string {
    mu.RLock()
    defer mu.RUnlock()
    names := make([]string, 0, len(registry))
    for name := range registry {
        names = append(names, name)
    }
    sort.Strings(names)
    return names
}
```

## Functional Options Pattern

```go
type options struct {
    maxRetries int
    timeout    time.Duration
    logger     *slog.Logger
}

type Option func(*options)

func WithMaxRetries(n int) Option {
    return func(o *options) { o.maxRetries = n }
}

func WithTimeout(d time.Duration) Option {
    return func(o *options) { o.timeout = d }
}

func defaultOptions() options {
    return options{
        maxRetries: 3,
        timeout:    30 * time.Second,
    }
}

func New(opts ...Option) *Thing {
    o := defaultOptions()
    for _, opt := range opts {
        opt(&o)
    }
    return &Thing{opts: o}
}
```

## Lifecycle Pattern

```go
type Lifecycle interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() HealthStatus
}

type App struct {
    mu         sync.Mutex
    components []Lifecycle
}

func (a *App) Register(components ...Lifecycle) {
    a.mu.Lock()
    defer a.mu.Unlock()
    a.components = append(a.components, components...)
}

func (a *App) Shutdown(ctx context.Context) error {
    a.mu.Lock()
    defer a.mu.Unlock()
    var errs []error
    // Stop in reverse order
    for i := len(a.components) - 1; i >= 0; i-- {
        if err := a.components[i].Stop(ctx); err != nil {
            errs = append(errs, err)
        }
    }
    return errors.Join(errs...)
}
```

## Interface Compliance Check

Always add compile-time interface checks:
```go
var _ ChatModel = (*OpenAIModel)(nil)
var _ Tool = (*FuncTool[any])(nil)
var _ Memory = (*CompositeMemory)(nil)
```

## Provider Registration via init()

```go
package openai

import "github.com/lookatitude/beluga-ai/llm"

func init() {
    llm.Register("openai", func(cfg llm.ProviderConfig) (llm.ChatModel, error) {
        return New(cfg)
    })
}
```

Users import with blank identifier:
```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
```

## Error Handling Pattern

```go
func (m *Model) Generate(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error) {
    resp, err := m.client.Call(ctx, msgs)
    if err != nil {
        var apiErr *provider.APIError
        if errors.As(err, &apiErr) {
            return nil, &core.Error{
                Op:      "llm.generate",
                Code:    mapErrorCode(apiErr.StatusCode),
                Message: apiErr.Message,
                Err:     err,
            }
        }
        return nil, &core.Error{
            Op:   "llm.generate",
            Code: core.ErrProviderDown,
            Err:  err,
        }
    }
    return resp, nil
}
```
