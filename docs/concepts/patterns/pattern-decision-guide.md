# Pattern Decision Guide

This guide helps developers choose the right design pattern for their use case in the Beluga AI Framework.

## When to Use Which Pattern

### Factory Pattern

**Use when:**
- Creating instances of different provider types
- Need to abstract creation logic
- Want to support multiple implementations

**Examples:**
- Creating LLM providers (OpenAI, Anthropic, etc.)
- Creating vector stores (InMemory, PgVector, etc.)
- Creating embedding providers

**Don't use when:**
- Simple object creation with no variants
- Single implementation with no alternatives

### Global Registry Pattern

**Use when:**
- Need to register providers at runtime
- Want to support plugin-style extensions
- Need dynamic provider discovery

**Examples:**
- Registering custom LLM providers
- Adding custom agent types
- Extending vector store implementations

**Don't use when:**
- Fixed set of providers known at compile time
- Simple factory pattern is sufficient

### OTEL Metrics Pattern

**Use when:**
- Need observability in production
- Want to track performance metrics
- Need distributed tracing

**Examples:**
- All public API methods
- Long-running operations
- Critical business logic

**Don't use when:**
- Internal helper functions
- Simple getters/setters
- Performance-critical paths (use selectively)

### Error Handling Pattern

**Use when:**
- Need programmatic error handling
- Want to preserve error context
- Need error categorization

**Examples:**
- All public API methods
- External service calls
- Configuration validation

**Don't use when:**
- Simple validation errors
- Internal errors that don't escape package

### Configuration Pattern

**Use when:**
- Need flexible configuration
- Want environment variable support
- Need configuration validation

**Examples:**
- Provider configuration
- Agent settings
- System-wide settings

**Don't use when:**
- Hard-coded values are acceptable
- Simple constants suffice

### Options Pattern

**Use when:**
- Need optional configuration
- Want fluent API
- Need to override defaults

**Examples:**
- Agent creation options
- LLM configuration
- Memory setup

**Don't use when:**
- All parameters are required
- Simple constructor is clearer

### Composition Pattern

**Use when:**
- Need to extend base functionality
- Want to share common behavior
- Need interface compliance

**Examples:**
- Custom agent types
- Extended tool implementations
- Specialized memory types

**Don't use when:**
- Simple standalone implementation
- No shared behavior needed

## Pattern Trade-offs

### Factory vs Direct Construction

**Factory:**
- ✅ More flexible
- ✅ Easier to test
- ✅ Supports multiple providers
- ❌ More complex
- ❌ Additional abstraction layer

**Direct Construction:**
- ✅ Simpler
- ✅ More direct
- ✅ Less overhead
- ❌ Less flexible
- ❌ Harder to test

### Global Registry vs Factory

**Global Registry:**
- ✅ Runtime registration
- ✅ Plugin support
- ✅ Dynamic discovery
- ❌ Global state
- ❌ More complex

**Factory:**
- ✅ Explicit creation
- ✅ No global state
- ✅ Simpler
- ❌ Less flexible
- ❌ Compile-time only

### OTEL Metrics vs Simple Logging

**OTEL Metrics:**
- ✅ Standardized
- ✅ Rich observability
- ✅ Distributed tracing
- ❌ More setup
- ❌ Additional dependencies

**Simple Logging:**
- ✅ Simple
- ✅ No dependencies
- ✅ Quick to implement
- ❌ Limited observability
- ❌ No metrics

## Pattern Anti-patterns to Avoid

### 1. Over-Engineering

**Anti-pattern:**
```go
// Unnecessary factory for simple case
factory := NewSimpleFactory()
obj := factory.Create() // Could just be: obj := NewObject()
```

**Better:**
```go
obj := NewObject()
```

### 2. Global State Abuse

**Anti-pattern:**
```go
// Too many global registries
globalRegistry1.Register(...)
globalRegistry2.Register(...)
globalRegistry3.Register(...)
```

**Better:**
```go
// Use dependency injection
type Service struct {
    registry1 Registry1
    registry2 Registry2
}
```

### 3. Error Swallowing

**Anti-pattern:**
```go
result, err := operation()
if err != nil {
    log.Printf("Error: %v", err) // Error lost
    return nil
}
```

**Better:**
```go
result, err := operation()
if err != nil {
    return nil, fmt.Errorf("operation failed: %w", err)
}
```

### 4. Configuration Complexity

**Anti-pattern:**
```go
// Too many configuration layers
config := NewConfig(
    WithOption1(NewOption1Config(WithSubOption(...))),
    WithOption2(NewOption2Config(WithSubOption(...))),
)
```

**Better:**
```go
config := NewConfig(
    WithOption1(value1),
    WithOption2(value2),
)
```

## Decision Tree

### Choosing a Creation Pattern

```
Need multiple implementations?
├─ Yes → Use Factory Pattern
│   └─ Need runtime registration?
│       ├─ Yes → Use Global Registry
│       └─ No → Use Factory
└─ No → Use Direct Construction
```

### Choosing an Observability Pattern

```
Need production observability?
├─ Yes → Use OTEL Metrics Pattern
│   └─ Need distributed tracing?
│       ├─ Yes → Use OTEL Tracing
│       └─ No → Use OTEL Metrics only
└─ No → Use Simple Logging
```

### Choosing an Error Pattern

```
Need programmatic error handling?
├─ Yes → Use Custom Error Types
│   └─ Need error codes?
│       ├─ Yes → Use Error Codes
│       └─ No → Use Wrapped Errors
└─ No → Use Standard Errors
```

## Best Practices

1. **Start Simple**: Begin with the simplest pattern that works
2. **Add Complexity Gradually**: Only add patterns when needed
3. **Be Consistent**: Use the same pattern across similar use cases
4. **Document Decisions**: Explain why a pattern was chosen
5. **Review Regularly**: Refactor if patterns become unnecessary

## Related Documentation

- [Pattern Examples](./pattern-examples.md) - Real-world examples
- [Cross-Package Patterns](./cross-package-patterns.md) - Pattern integration
- [Package Design Patterns](../../package_design_patterns.md) - Complete reference
