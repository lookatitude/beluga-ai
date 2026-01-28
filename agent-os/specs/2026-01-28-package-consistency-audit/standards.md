# Applicable Standards

## 1. Required Files Convention

From `docs/package_design_patterns.md`:

### Standard Package Layout

Every package **MUST** follow this standardized structure:

```
pkg/{package_name}/
├── iface/                    # Interfaces and types (REQUIRED)
├── internal/                 # Private implementation details (OPTIONAL)
├── providers/                # Provider implementations (for multi-provider packages)
├── config.go                 # Configuration structs and validation (REQUIRED)
├── metrics.go                # OTEL metrics implementation (REQUIRED)
├── errors.go                 # Custom error types with Op/Err/Code pattern (REQUIRED)
├── {package_name}.go         # Main interfaces and factory functions
├── factory.go OR registry.go # Global factory/registry for multi-provider packages
├── test_utils.go             # Advanced testing utilities and mocks (REQUIRED)
├── advanced_test.go          # Comprehensive test suites (REQUIRED)
└── README.md                 # Package documentation (REQUIRED)
```

### Key Requirements

1. **Main file naming**: `{package_name}.go` must match the package directory name
2. **Registry requirement**: Packages with `providers/` directory must have `registry.go`
3. **iface/ directory**: All public interfaces must be in `iface/` subdirectory

## 2. Registry Pattern

From `docs/package_design_patterns.md`:

### Global Registry Pattern (REQUIRED for multi-provider packages)

```go
type ProviderRegistry struct {
    mu       sync.RWMutex
    creators map[string]func(ctx context.Context, config Config) (Interface, error)
}

func NewProviderRegistry() *ProviderRegistry {
    return &ProviderRegistry{
        creators: make(map[string]func(ctx context.Context, config Config) (Interface, error)),
    }
}

func (r *ProviderRegistry) Register(name string, creator func(ctx context.Context, config Config) (Interface, error)) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.creators[name] = creator
}

func (r *ProviderRegistry) Create(ctx context.Context, name string, config Config) (Interface, error) {
    r.mu.RLock()
    creator, exists := r.creators[name]
    r.mu.RUnlock()

    if !exists {
        return nil, NewError("unknown_provider", fmt.Errorf("provider '%s' not found", name))
    }
    return creator(ctx, config)
}

// Global factory instance
var globalRegistry = NewProviderRegistry()

// Global convenience functions
func RegisterGlobal(name string, creator func(ctx context.Context, config Config) (Interface, error)) {
    globalRegistry.Register(name, creator)
}

func NewProvider(ctx context.Context, name string, config Config) (Interface, error) {
    return globalRegistry.Create(ctx, name, config)
}
```

## 3. Intentional Deviations

The following packages are permitted to deviate from the standard structure for documented reasons:

### config package
- **Deviation**: No registry.go
- **Rationale**: Configuration is loaded from files/environment, not created through provider factories
- **Pattern used**: Direct factory functions for loading config

### core package
- **Deviation**: No {package_name}.go main file
- **Rationale**: Utility package containing multiple independent entry points (di.go, runnable.go, errors.go)
- **Pattern used**: Multiple focused files for different utilities

### schema package
- **Deviation**: No registry.go
- **Rationale**: Pure data structure definitions package
- **Pattern used**: Direct struct exports, no factory pattern

### voicesession package
- **Deviation**: No registry.go
- **Rationale**: Single implementation, not a multi-provider package
- **Pattern used**: Direct constructor function

### voiceutils package
- **Deviation**: No main API file
- **Rationale**: Shared interfaces and utility types for other voice packages
- **Pattern used**: Interface definitions only, imported by other packages

### convenience package
- **Deviation**: Namespace package with no root-level code files
- **Rationale**: Aggregation package grouping related convenience sub-packages
- **Pattern used**: Sub-packages (agent, rag, voiceagent, mock, context, provider) each follow standard structure independently

## 4. Provider Registration Pattern

Providers must register themselves using `init()` functions:

```go
// providers/mock.go
func init() {
    prompts.Register("mock", func(ctx context.Context, config any) (iface.TemplateEngine, error) {
        return NewMockTemplateEngine(), nil
    })
}
```

This ensures providers are automatically available when their package is imported.
