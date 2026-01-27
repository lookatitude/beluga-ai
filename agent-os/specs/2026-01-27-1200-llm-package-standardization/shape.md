# Package Structure Shape

## Standard Package Structure (Reference: pkg/tools)

```
pkg/{package_name}/
├── iface/                    # Public interfaces and types (REQUIRED)
│   ├── interfaces.go         # Interface definitions only
│   └── types.go              # Shared types (no implementation)
├── internal/                 # Private implementation details (if needed)
├── providers/                # Provider implementations
│   └── {provider}/
│       ├── init.go           # Registry registration
│       └── {provider}.go     # Implementation
├── config.go                 # Configuration structs with validation
├── metrics.go                # OTEL metrics implementation
├── errors.go                 # Custom errors (Op/Err/Code pattern)
├── {package_name}.go         # Main API and factory functions
├── registry.go               # Full registry implementation
├── test_utils.go             # Test helpers and mock factories
├── advanced_test.go          # Comprehensive test suite
└── README.md                 # Package documentation
```

## Error Pattern (Op/Err/Code)

```go
type ErrorCode string

const (
    ErrorCodeInvalidInput ErrorCode = "invalid_input"
    // ...
)

type PackageError struct {
    Op      string    // Operation that failed
    Err     error     // Underlying error
    Code    ErrorCode // Error classification
    Message string    // Human-readable message
}
```

## Registry Pattern (sync.Once singleton)

```go
var (
    globalRegistry *Registry
    registryOnce   sync.Once
)

func GetRegistry() *Registry {
    registryOnce.Do(func() {
        globalRegistry = &Registry{
            providers: make(map[string]ProviderFactory),
        }
    })
    return globalRegistry
}
```

## Current vs Target State

### pkg/embeddings

**Current:**
```
pkg/embeddings/
├── iface/
│   ├── errors.go          # DUPLICATE - delete
│   ├── registry.go        # Interface only - keep
│   └── iface_test.go      # Move to root
├── internal/registry/
│   └── registry.go        # Implementation - move to root
├── providers/             # Keep as-is
├── testutils/
│   └── helpers.go         # Merge into test_utils.go
├── embeddings_mock.go     # Merge into test_utils.go
├── advanced_mock.go       # Merge into test_utils.go
├── factory.go             # Delete - use registry.go
├── errors.go              # Keep - add missing codes
├── registry.go            # Thin wrapper - replace with full impl
└── test_utils.go          # Keep - consolidation target
```

**Target:**
```
pkg/embeddings/
├── iface/
│   ├── interfaces.go      # Embedder interface
│   └── registry.go        # Registry interface only
├── providers/             # Unchanged
├── config.go              # Configuration
├── errors.go              # Consolidated errors
├── registry.go            # Full implementation
├── embeddings.go          # Main API
├── test_utils.go          # All mocks consolidated
├── advanced_test.go       # Tests
└── errors_test.go         # Error tests (moved from iface)
```

### pkg/chatmodels

**Current:**
```
pkg/chatmodels/
├── iface/
│   └── registry.go        # Has full implementation
├── internal/              # Empty directory
├── providers/
├── advanced_mock.go       # Merge into test_utils.go
├── registry.go            # Thin wrapper
└── test_utils.go
```

**Target:**
```
pkg/chatmodels/
├── iface/
│   └── registry.go        # Interface + types only
├── providers/
├── registry.go            # Full implementation
├── test_utils.go          # Consolidated mocks
└── ...
```
