# Pattern Consistency Guide

This document defines the standard patterns that all packages in the Beluga AI Framework must follow to ensure consistency across the codebase.

## Table of Contents

1. [Overview](#overview)
2. [Registry Pattern](#registry-pattern)
3. [Metrics Initialization](#metrics-initialization)
4. [Configuration Pattern](#configuration-pattern)
5. [Error Handling Pattern](#error-handling-pattern)
6. [Migration Guide](#migration-guide)

---

## Overview

The Beluga AI Framework has evolved over time, leading to some inconsistencies in patterns across packages. This guide standardizes these patterns and provides migration paths.

### Current Inconsistencies

| Pattern | Current State | Standard |
|---------|--------------|----------|
| Registry | Mixed (init vs manual) | `init()` auto-registration |
| Metrics | Mixed (InitMetrics vs SetGlobalMetrics) | `InitMetrics()` + `GetMetrics()` |
| Config | Mixed (SetDefaults vs auto) | `SetDefaults()` + `Validate()` |

---

## Registry Pattern

### Standard Pattern: Auto-Registration via `init()`

All providers should auto-register using the `init()` function.

#### Standard Implementation

```go
// pkg/{package}/providers/{provider}/init.go
package {provider}

import "github.com/lookatitude/beluga-ai/pkg/{package}"

func init() {
    {package}.GetRegistry().Register("{provider_name}", New{Provider}Factory)
}
```

#### Package-Level Registry

```go
// pkg/{package}/registry.go
package {package}

import (
    "sync"
    "github.com/lookatitude/beluga-ai/pkg/{package}/iface"
)

var (
    globalRegistry *Registry
    registryOnce   sync.Once
)

type Registry struct {
    providers map[string]func(*Config) (iface.{Interface}, error)
    mu        sync.RWMutex
}

func GetRegistry() *Registry {
    registryOnce.Do(func() {
        globalRegistry = &Registry{
            providers: make(map[string]func(*Config) (iface.{Interface}, error)),
        }
    })
    return globalRegistry
}

func (r *Registry) Register(name string, factory func(*Config) (iface.{Interface}, error)) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.providers[name] = factory
}

func (r *Registry) GetProvider(name string, config *Config) (iface.{Interface}, error) {
    r.mu.RLock()
    factory, exists := r.providers[name]
    r.mu.RUnlock()
    
    if !exists {
        return nil, New{ErrorType}("GetProvider", ErrCodeUnsupportedProvider,
            fmt.Errorf("provider '%s' not registered", name))
    }
    
    return factory(config)
}
```

### Current State by Package

| Package | Current Pattern | Status | Action Required |
|---------|----------------|--------|-----------------|
| S2S | ✅ `init()` auto-registration | Correct | None |
| STT | ✅ `init()` auto-registration | Correct | None |
| TTS | ✅ `init()` auto-registration | Correct | None |
| VAD | ✅ `init()` auto-registration | Correct | None |
| Noise | ✅ `init()` auto-registration | Correct | None |
| LLMs | ❌ Manual factory registration | Needs fix | Add `init()` files |
| Vectorstores | ❌ Manual registration | Needs fix | Add `init()` files |
| Embeddings | ❌ Factory pattern | Needs fix | Add `init()` files |
| Memory | ❌ Internal factory only | Needs fix | Add global registry |

### Migration Steps

1. **Add Registry to Package** (if missing)
   - Create `registry.go` with `GetRegistry()` function
   - Implement `Register()` and `GetProvider()` methods

2. **Add `init.go` to Each Provider**
   ```go
   package {provider}
   
   import "github.com/lookatitude/beluga-ai/pkg/{package}"
   
   func init() {
       {package}.GetRegistry().Register("{provider_name}", New{Provider}Factory)
   }
   ```

3. **Update Provider Creation**
   ```go
   // Before
   factory := llms.NewFactory()
   factory.RegisterProviderFactory("openai", openai.NewOpenAIProviderFactory())
   provider, err := factory.CreateProvider("openai", config)
   
   // After
   provider, err := llms.NewProvider(ctx, "openai", config)
   ```

---

## Metrics Initialization

### Standard Pattern: `InitMetrics()` + `GetMetrics()`

All packages should use the same metrics initialization pattern.

#### Standard Implementation

```go
// pkg/{package}/metrics.go
package {package}

import (
    "sync"
    "go.opentelemetry.io/otel/metric"
)

var (
    globalMetrics *Metrics
    metricsOnce   sync.Once
)

// InitMetrics initializes the global metrics instance.
func InitMetrics(meter metric.Meter) {
    metricsOnce.Do(func() {
        globalMetrics = NewMetrics(meter)
    })
}

// GetMetrics returns the global metrics instance.
func GetMetrics() *Metrics {
    return globalMetrics
}
```

#### Usage in Providers

```go
// In provider implementation
type {Provider}Provider struct {
    metrics {Package}MetricsRecorder
    // ...
}

func New{Provider}Provider(config *Config) (*{Provider}Provider, error) {
    return &{Provider}Provider{
        metrics: {package}.GetMetrics(), // Get global metrics
        // ...
    }, nil
}
```

### Current State by Package

| Package | Current Pattern | Status | Action Required |
|---------|----------------|--------|-----------------|
| LLMs | ✅ `InitMetrics()` + `GetMetrics()` | Correct | None |
| S2S | ✅ `InitMetrics()` + `GetMetrics()` | Correct | None |
| Vectorstores | ❌ `SetGlobalMetrics()` | Needs fix | Change to `InitMetrics()` |
| Memory | ❌ `GetGlobalMetrics()` without init | Needs fix | Add `InitMetrics()` |

### Migration Steps

1. **Update Metrics Structure**
   ```go
   // Before
   var globalMetrics *Metrics
   
   func SetGlobalMetrics(metrics *Metrics) {
       globalMetrics = metrics
   }
   
   func GetGlobalMetrics() *Metrics {
       return globalMetrics
   }
   
   // After
   var (
       globalMetrics *Metrics
       metricsOnce   sync.Once
   )
   
   func InitMetrics(meter metric.Meter) {
       metricsOnce.Do(func() {
           globalMetrics = NewMetrics(meter)
       })
   }
   
   func GetMetrics() *Metrics {
       return globalMetrics
   }
   ```

2. **Update Initialization Code**
   ```go
   // Before
   metrics := vectorstores.NewMetricsCollector(meter)
   vectorstores.SetGlobalMetrics(metrics)
   
   // After
   vectorstores.InitMetrics(meter)
   ```

3. **Update Provider Usage**
   ```go
   // Before
   metrics := vectorstores.GetGlobalMetrics()
   
   // After
   metrics := vectorstores.GetMetrics()
   ```

---

## Configuration Pattern

### Standard Pattern: Struct Tags + `SetDefaults()` + `Validate()`

All configuration structs should follow this pattern.

#### Standard Implementation

```go
// pkg/{package}/config.go
package {package}

import (
    "time"
    "github.com/go-playground/validator/v10"
)

type Config struct {
    // Base fields
    Provider string `mapstructure:"provider" yaml:"provider" validate:"required"`
    APIKey   string `mapstructure:"api_key" yaml:"api_key" validate:"required"`
    BaseURL  string `mapstructure:"base_url" yaml:"base_url"`
    Timeout  time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
    
    // Provider-specific options
    ProviderSpecific map[string]any `mapstructure:"provider_specific" yaml:"provider_specific"`
}

// DefaultConfig returns a configuration with default values.
func DefaultConfig() *Config {
    return &Config{
        Provider:         "",
        APIKey:           "",
        BaseURL:          "",
        Timeout:          30 * time.Second,
        ProviderSpecific: make(map[string]any),
    }
}

// SetDefaults sets default values for unset fields.
func (c *Config) SetDefaults() {
    if c.Timeout == 0 {
        c.Timeout = 30 * time.Second
    }
    if c.ProviderSpecific == nil {
        c.ProviderSpecific = make(map[string]any)
    }
}

// Validate validates the configuration.
func (c *Config) Validate() error {
    validate := validator.New()
    return validate.Struct(c)
}
```

#### Functional Options Pattern

```go
// ConfigOption is a functional option for configuring providers.
type ConfigOption func(*Config)

// WithAPIKey sets the API key.
func WithAPIKey(apiKey string) ConfigOption {
    return func(c *Config) {
        c.APIKey = apiKey
    }
}

// WithTimeout sets the timeout.
func WithTimeout(timeout time.Duration) ConfigOption {
    return func(c *Config) {
        c.Timeout = timeout
    }
}

// NewConfig creates a new configuration with options.
func NewConfig(opts ...ConfigOption) *Config {
    config := DefaultConfig()
    for _, opt := range opts {
        opt(config)
    }
    config.SetDefaults()
    return config
}
```

### Current State by Package

| Package | Current Pattern | Status | Action Required |
|---------|----------------|--------|-----------------|
| LLMs | ✅ Struct tags + Validate + SetDefaults | Correct | None |
| S2S | ✅ Struct tags + Validate + SetDefaults | Correct | None |
| Vectorstores | ⚠️ Partial | Needs review | Ensure all fields have tags |
| Memory | ⚠️ Partial | Needs review | Ensure all fields have tags |

### Migration Steps

1. **Add Struct Tags**
   ```go
   // Before
   type Config struct {
       Provider string
       APIKey   string
       Timeout  time.Duration
   }
   
   // After
   type Config struct {
       Provider string `mapstructure:"provider" yaml:"provider" validate:"required"`
       APIKey   string `mapstructure:"api_key" yaml:"api_key" validate:"required"`
       Timeout  time.Duration `mapstructure:"timeout" yaml:"timeout" default:"30s"`
   }
   ```

2. **Add SetDefaults Method**
   ```go
   func (c *Config) SetDefaults() {
       if c.Timeout == 0 {
           c.Timeout = 30 * time.Second
       }
       // ... other defaults
   }
   ```

3. **Add Validate Method**
   ```go
   func (c *Config) Validate() error {
       validate := validator.New()
       return validate.Struct(c)
   }
   ```

4. **Update Creation Functions**
   ```go
   func NewConfig(opts ...ConfigOption) *Config {
       config := DefaultConfig()
       for _, opt := range opts {
           opt(config)
       }
       config.SetDefaults() // Call SetDefaults after applying options
       return config
   }
   ```

---

## Error Handling Pattern

### Standard Pattern: Custom Error Types with Op/Err/Code

All packages should use consistent error handling.

#### Standard Implementation

```go
// pkg/{package}/errors.go
package {package}

import "fmt"

// {Package}Error represents errors specific to {package} operations.
type {Package}Error struct {
    Op      string // Operation that failed
    Err     error  // Underlying error
    Code    string // Error code for programmatic handling
    Message string // Human-readable message
}

func (e *{Package}Error) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %s: %v", e.Op, e.Message, e.Err)
    }
    return fmt.Sprintf("%s: %s", e.Op, e.Message)
}

func (e *{Package}Error) Unwrap() error {
    return e.Err
}

// New{Package}Error creates a new error.
func New{Package}Error(op, code string, err error) *{Package}Error {
    return &{Package}Error{
        Op:   op,
        Err:  err,
        Code: code,
        Message: getErrorMessage(code),
    }
}

// New{Package}ErrorWithMessage creates a new error with custom message.
func New{Package}ErrorWithMessage(op, code, message string, err error) *{Package}Error {
    return &{Package}Error{
        Op:      op,
        Err:     err,
        Code:    code,
        Message: message,
    }
}

// Error codes
const (
    ErrCodeInvalidConfig        = "invalid_config"
    ErrCodeUnsupportedProvider  = "unsupported_provider"
    ErrCodeInvalidRequest       = "invalid_request"
    ErrCodeTimeout              = "timeout"
    ErrCodeRateLimit            = "rate_limit"
)

func getErrorMessage(code string) string {
    messages := map[string]string{
        ErrCodeInvalidConfig:       "Invalid configuration",
        ErrCodeUnsupportedProvider: "Unsupported provider",
        ErrCodeInvalidRequest:      "Invalid request",
        ErrCodeTimeout:             "Operation timed out",
        ErrCodeRateLimit:           "Rate limit exceeded",
    }
    return messages[code]
}
```

### Current State

All packages follow this pattern correctly. No migration needed.

---

## Migration Guide

### Step-by-Step Migration Process

1. **Identify Inconsistencies**
   - Review package against this guide
   - Document current patterns
   - Identify required changes

2. **Create Migration Branch**
   ```bash
   git checkout -b fix/pattern-consistency-{package}
   ```

3. **Update Registry Pattern** (if needed)
   - Add `registry.go` if missing
   - Add `init.go` to each provider
   - Update provider creation code

4. **Update Metrics Pattern** (if needed)
   - Change `SetGlobalMetrics()` to `InitMetrics()`
   - Update initialization code
   - Update provider usage

5. **Update Config Pattern** (if needed)
   - Add struct tags
   - Add `SetDefaults()` method
   - Add `Validate()` method
   - Update creation functions

6. **Update Tests**
   - Update test setup code
   - Ensure all tests pass
   - Add tests for new patterns

7. **Update Documentation**
   - Update package README
   - Update examples
   - Update migration guide

8. **Review and Merge**
   - Code review
   - Ensure backward compatibility
   - Merge to main

### Backward Compatibility

When migrating, ensure backward compatibility:

1. **Deprecation Warnings**
   ```go
   // SetGlobalMetrics is deprecated. Use InitMetrics instead.
   // Deprecated: Use InitMetrics(meter) instead.
   func SetGlobalMetrics(metrics *Metrics) {
       // ... implementation
   }
   ```

2. **Wrapper Functions**
   ```go
   // NewProvider is a convenience function that uses the registry.
   func NewProvider(ctx context.Context, providerName string, config *Config) (iface.Provider, error) {
       registry := GetRegistry()
       return registry.GetProvider(providerName, config)
   }
   ```

3. **Gradual Migration**
   - Support both patterns during transition
   - Document migration path
   - Remove old patterns in next major version

---

## Checklist

Before considering a package consistent:

- [ ] Registry uses `init()` auto-registration
- [ ] Metrics use `InitMetrics()` + `GetMetrics()` pattern
- [ ] Config has struct tags, `SetDefaults()`, and `Validate()`
- [ ] Errors use custom error types with Op/Err/Code
- [ ] All tests pass
- [ ] Documentation updated
- [ ] Examples updated
- [ ] Backward compatibility maintained (if applicable)

---

## Examples

### Complete Example: LLM Provider

```go
// pkg/llms/registry.go
package llms

var (
    globalRegistry *Registry
    registryOnce   sync.Once
)

func GetRegistry() *Registry {
    registryOnce.Do(func() {
        globalRegistry = &Registry{
            providers: make(map[string]func(*Config) (iface.ChatModel, error)),
        }
    })
    return globalRegistry
}

// pkg/llms/providers/openai/init.go
package openai

import "github.com/lookatitude/beluga-ai/pkg/llms"

func init() {
    llms.GetRegistry().Register("openai", NewOpenAIProviderFactory)
}

// pkg/llms/metrics.go
package llms

var (
    globalMetrics *Metrics
    metricsOnce   sync.Once
)

func InitMetrics(meter metric.Meter) {
    metricsOnce.Do(func() {
        globalMetrics = NewMetrics(meter)
    })
}

func GetMetrics() *Metrics {
    return globalMetrics
}
```

---

## Resources

- [Package Design Patterns](../package_design_patterns.md)
- [Provider Implementation Guide](../guides/implementing-providers.md)
- [Gap Analysis](../architecture/gaps-analysis.md)
