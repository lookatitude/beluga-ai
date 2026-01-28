# Applied Standards

## global/required-files

Multi-provider packages MUST include these files:

```
pkg/{package_name}/
├── iface/                    # Public interfaces and types (REQUIRED)
│   ├── interfaces.go         # Core interfaces
│   └── registry.go           # Registry interface and factory type
├── providers/                # Provider implementations
│   └── {provider_name}/
│       ├── {provider}.go     # Implementation
│       ├── config.go         # Provider configuration
│       └── init.go           # Auto-registration via init()
├── config.go                 # Package-level configuration
├── errors.go                 # Custom errors (Op/Err/Code pattern)
├── metrics.go                # OTEL metrics implementation
├── registry.go               # Global registry implementation
├── {package_name}.go         # Main API and factory functions
└── README.md                 # Package documentation
```

## backend/registry-shape

Registry implementation MUST follow this pattern:

```go
// In iface/registry.go
type Factory func(ctx context.Context, config any) (Interface, error)

type Registry interface {
    Register(name string, factory Factory)
    Create(ctx context.Context, name string, config any) (Interface, error)
    ListProviders() []string
    IsRegistered(name string) bool
}

// In registry.go
type ProviderRegistry struct {
    providers map[string]Factory
    mu        sync.RWMutex
}

var (
    globalRegistry *ProviderRegistry
    registryOnce   sync.Once
)

func GetRegistry() *ProviderRegistry {
    registryOnce.Do(func() {
        globalRegistry = &ProviderRegistry{
            providers: make(map[string]Factory),
        }
    })
    return globalRegistry
}

// Global convenience functions
func Register(name string, factory Factory) {
    GetRegistry().Register(name, factory)
}

func Create(ctx context.Context, name string, config any) (Interface, error) {
    return GetRegistry().Create(ctx, name, config)
}

func ListProviders() []string {
    return GetRegistry().ListProviders()
}

// Interface compliance
var _ iface.Registry = (*ProviderRegistry)(nil)
```

## backend/op-err-code

Error types MUST follow this pattern:

```go
// Error codes for programmatic error handling
const (
    ErrCodeInvalidConfig    = "invalid_config"
    ErrCodeProviderNotFound = "provider_not_found"
    // ... other codes
)

// Error struct with Op/Err/Code fields
type Error struct {
    Op      string // Operation that failed
    Err     error  // Underlying error
    Code    string // Error classification
    Message string // Human-readable message (optional)
}

func (e *Error) Error() string {
    if e.Message != "" {
        return fmt.Sprintf("%s %s: %s", packageName, e.Op, e.Message)
    }
    return fmt.Sprintf("%s %s: %v", packageName, e.Op, e.Err)
}

func (e *Error) Unwrap() error {
    return e.Err
}

// Constructor functions
func NewError(op string, err error, code string) *Error
func NewErrorWithMessage(op string, err error, code, message string) *Error
func NewProviderNotFoundError(op, provider string) *Error
```

## backend/provider-init

Provider auto-registration MUST use init() function:

```go
// In providers/{name}/init.go
package providername

import (
    "context"
    "github.com/lookatitude/beluga-ai/pkg/{package}"
    "{package}iface" "github.com/lookatitude/beluga-ai/pkg/{package}/iface"
)

func init() {
    {package}.Register("providername", func(ctx context.Context, config any) ({package}iface.Interface, error) {
        // Type assert config to expected type
        cfg, ok := config.(*Config)
        if !ok {
            return nil, fmt.Errorf("invalid config type: expected *Config, got %T", config)
        }
        return New(cfg)
    })
}
```

## testing/test-utils

Test utilities SHOULD be consolidated in test_utils.go:

```go
// In test_utils.go
package packagename

// MockInterface provides a mock implementation for testing
type MockInterface struct {
    // ... mock fields
}

// NewMockInterface creates a mock for testing
func NewMockInterface(options ...MockOption) *MockInterface

// TestHelper provides common test utilities
type TestHelper struct {
    // ... helper fields
}
```
