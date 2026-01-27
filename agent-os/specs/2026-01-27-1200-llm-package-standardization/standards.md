# Standards Applied

## Framework Standards

### global/required-files.md
Multi-provider packages must have:
- `registry.go` - Full implementation at root level
- `errors.go` - Errors at root with Op/Err/Code pattern
- `test_utils.go` - All test utilities consolidated
- `iface/` - Interfaces only, no implementation

### global/subpackage-structure.md
- `iface/` contains only interfaces and shared types
- No implementation code in `iface/`
- Providers should import from `iface/` to avoid cycles

### backend/registry-shape.md
Registry implementation must use:
- `sync.Once` for thread-safe singleton initialization
- `sync.RWMutex` for concurrent access protection
- Global convenience functions (`Register`, `Create`, `GetRegistry`)

### backend/iface-errors.md
Error type location:
- Error struct and codes at package root (`errors.go`)
- Only error interface (if any) in `iface/`
- Helper functions (`Is*Error`, `As*Error`) at root

## Error Handling Pattern

```go
// errors.go at package root
type ErrorCode string

const (
    ErrCodeInvalidConfig   ErrorCode = "invalid_config"
    ErrCodeProviderNotFound ErrorCode = "provider_not_found"
    // ...
)

var (
    ErrInvalidConfig   = errors.New("invalid configuration")
    ErrProviderNotFound = errors.New("provider not found")
    // ...
)

type PackageError struct {
    Op      string
    Code    ErrorCode
    Message string
    Err     error
}

func (e *PackageError) Error() string { ... }
func (e *PackageError) Unwrap() error { return e.Err }

func NewPackageError(op string, code ErrorCode, message string, err error) *PackageError { ... }
func IsPackageError(err error) bool { ... }
func AsPackageError(err error) (*PackageError, bool) { ... }
```

## Registry Pattern

```go
// registry.go at package root
type ProviderFactory func(ctx context.Context, config any) (iface.Interface, error)

type Registry struct {
    providers map[string]ProviderFactory
    mu        sync.RWMutex
}

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

func (r *Registry) Register(name string, factory ProviderFactory) { ... }
func (r *Registry) Create(ctx context.Context, name string, config any) (iface.Interface, error) { ... }
func (r *Registry) ListProviders() []string { ... }
func (r *Registry) IsRegistered(name string) bool { ... }

// Global convenience functions
func Register(name string, factory ProviderFactory) { GetRegistry().Register(name, factory) }
func Create(ctx context.Context, name string, config any) (iface.Interface, error) { ... }
func ListProviders() []string { return GetRegistry().ListProviders() }

// Ensure interface compliance
var _ iface.Registry = (*Registry)(nil)
```

## Test Utilities Pattern

```go
// test_utils.go at package root
// All mocks and test helpers consolidated in one file

type MockProvider struct {
    mock.Mock
    // fields
}

func NewMockProvider(options ...MockOption) *MockProvider { ... }

type MockOption func(*MockProvider)

func WithMockError(err error) MockOption { ... }
func WithMockResponse(response any) MockOption { ... }

// Test data creation helpers
func CreateTestConfig() Config { ... }
func CreateTestData(count int) []TestData { ... }

// Assertion helpers
func AssertResponse(t *testing.T, response any) { ... }
func AssertError(t *testing.T, err error, expectedCode ErrorCode) { ... }
```
