# Test Import Cycle Issue

## Problem

The `pkg/chatmodels` and `pkg/embeddings` test suites fail due to import cycles when attempting to register providers for testing.

### Root Cause

The provider registration pattern creates an import cycle:

```
pkg/chatmodels/chatmodels_test.go
  → imports pkg/chatmodels/providers/mock (or openai)
    → imports pkg/chatmodels (to register in init())
      → creates cycle back to test file
```

### Current Architecture

1. **Provider packages** (`pkg/chatmodels/providers/mock`, `pkg/chatmodels/providers/openai`, etc.) import the main package (`pkg/chatmodels`) to register themselves in `init()` functions.

2. **Test files** need providers to be registered to test the registry-based factory functions (`NewChatModel`, `NewEmbedder`, etc.).

3. **Import cycle**: Test file → Provider package → Main package → (cycle detected)

### Affected Packages

- `pkg/chatmodels` - All tests fail with "import cycle not allowed in test"
- `pkg/embeddings` - All tests fail with "import cycle not allowed in test"

## Proposed Solutions

### Solution 1: Separate Registry Interface (Recommended)

Create a separate registry interface package that providers can import without creating cycles.

**Structure:**
```
pkg/chatmodels/
├── registry/
│   ├── iface/
│   │   └── registry.go      # Registry interface only
│   └── registry.go          # Implementation
├── providers/
│   └── mock/
│       └── init.go          # Imports registry/iface, not main package
└── chatmodels.go            # Imports registry, uses implementation
```

**Benefits:**
- Clean separation of concerns
- No import cycles
- Providers don't need to import the full main package
- Easy to test

**Implementation Steps:**
1. Create `pkg/chatmodels/registry/iface/registry.go` with just the interface
2. Move registry implementation to `pkg/chatmodels/registry/registry.go`
3. Update provider `init.go` files to import `registry/iface` instead of main package
4. Update main package to use the registry implementation

### Solution 2: Test Helper Package

Create a separate test helper package that imports providers and can be imported by tests.

**Structure:**
```
pkg/chatmodels/
├── testhelpers/
│   └── register.go          # Imports all providers, registers them
└── chatmodels_test.go        # Imports testhelpers
```

**Limitation:** Still creates cycles because testhelpers → providers → main package

### Solution 3: Manual Registration in Tests

Manually register providers in test setup without importing provider packages.

**Implementation:**
```go
func init() {
    // Manually register providers for tests
    GetRegistry().Register("mock", func(model string, config *Config, options *iface.Options) (iface.ChatModel, error) {
        // Direct instantiation without going through provider package
        return mock.NewMockChatModel(model, config, options)
    })
}
```

**Limitation:** Requires exporting provider constructors, which may not be desired.

### Solution 4: Build Tags for Test Registration

Use build tags to create separate registration files for tests.

**Structure:**
```
pkg/chatmodels/
├── providers/
│   └── mock/
│       ├── init.go              # Normal registration
│       └── init_test.go         # Test-only registration (build tag)
└── chatmodels_test.go
```

**Limitation:** Still has cycles, just moves the problem.

## Recommended Approach: Solution 1

Solution 1 (Separate Registry Interface) is the cleanest and most maintainable approach. It follows Go best practices and eliminates the root cause of the cycle.

### Implementation Example

**`pkg/chatmodels/registry/iface/registry.go`:**
```go
package iface

import "github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"

// ChatModelFactory defines the function signature for creating chat models.
type ChatModelFactory func(model string, config *Config, options *iface.Options) (iface.ChatModel, error)

// Registry defines the interface for chat model provider registration.
type Registry interface {
    Register(name string, factory ChatModelFactory)
    CreateProvider(model string, config *Config, options *iface.Options) (iface.ChatModel, error)
    IsRegistered(name string) bool
    ListProviders() []string
}
```

**`pkg/chatmodels/providers/mock/init.go`:**
```go
package mock

import (
    "github.com/lookatitude/beluga-ai/pkg/chatmodels/registry/iface"
    chatmodelsiface "github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
)

var globalRegistry iface.Registry

func SetRegistry(registry iface.Registry) {
    globalRegistry = registry
}

func init() {
    if globalRegistry != nil {
        globalRegistry.Register("mock", func(model string, config *Config, options *chatmodelsiface.Options) (chatmodelsiface.ChatModel, error) {
            return NewMockChatModel(model, config, options)
        })
    }
}
```

**Note:** This approach requires a registration mechanism that doesn't rely on `init()` functions, or a way to set the registry before `init()` runs.

## Alternative: Lazy Registration

Instead of registering in `init()`, use a lazy registration pattern where providers register themselves on first access:

```go
// In provider package
func Register() {
    chatmodels.GetRegistry().Register("mock", NewMockChatModel)
}

// In test
func init() {
    mock.Register()
    openai.Register()
}
```

This breaks the cycle because registration happens explicitly, not automatically in `init()`.

## Current Workaround

For now, tests that require provider registration are skipped or fail. The functionality works correctly in production because:

1. Applications import provider packages directly (not in test context)
2. The import cycle only occurs in test context
3. Production code doesn't have this issue

## Related Issues

- Similar pattern exists in `pkg/embeddings`
- Other packages using registry pattern may face same issue
- Consider standardizing registry pattern across all packages

## References

- [Go Import Cycles](https://golang.org/ref/spec#Import_declarations)
- [Go Testing Best Practices](https://golang.org/doc/effective_go#testing)
- [Dependency Injection Patterns](https://github.com/golang/go/wiki/CodeReviewComments#interfaces)
