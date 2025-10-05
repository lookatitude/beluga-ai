# Container Interface Contract

## Interface Definition

```go
type Container interface {
    // Register registers a factory function for a type
    // Factory function MUST return concrete type or interface
    Register(factoryFunc interface{}) error

    // Resolve resolves a dependency by type using reflection
    // Target MUST be pointer to the desired type
    Resolve(target interface{}) error

    // MustResolve resolves a dependency or panics
    // Should be used only when dependency is guaranteed to exist
    MustResolve(target interface{})

    // Has checks if a type is registered in the container
    // Returns true if factory or singleton exists for the type
    Has(typ reflect.Type) bool

    // Clear removes all registered dependencies
    // Used for testing and container reset scenarios
    Clear()

    // Singleton registers a singleton instance for a type
    // Instance will be returned for all future resolutions
    Singleton(instance interface{})

    // HealthChecker provides health check functionality
    HealthChecker
}
```

## Contract Requirements

### Register Method
- **Input**: `factoryFunc` (factory function interface{})
- **Output**: `error` (registration failure details)
- **Behavior**: Type-safe factory registration with validation
- **Validation**: Factory MUST be function, MUST return at least one value

### Resolve Method
- **Input**: `target` (pointer to desired type)
- **Output**: `error` (resolution failure details)
- **Behavior**: Recursive dependency resolution with circular detection
- **Thread Safety**: MUST support concurrent resolution

### MustResolve Method  
- **Input**: `target` (pointer to desired type)
- **Output**: None (panics on failure)
- **Behavior**: Convenience method for guaranteed dependencies
- **Usage**: Only when dependency existence is certain

### Has Method
- **Input**: `typ` (reflect.Type)
- **Output**: `bool` (registration status)
- **Behavior**: Check factory or singleton existence
- **Performance**: MUST be fast lookup operation

### Clear Method
- **Input**: None
- **Output**: None  
- **Behavior**: Remove all registrations for fresh container state
- **Usage**: Testing and reset scenarios

### Singleton Method
- **Input**: `instance` (any interface{})
- **Output**: None
- **Behavior**: Register pre-created instance for type
- **Lifecycle**: Instance persists until container clear

## Implementation Requirements

### Dependency Resolution
- MUST detect circular dependencies and return meaningful errors
- MUST support recursive resolution of complex dependency graphs
- MUST cache resolved instances for performance
- MUST handle factory function errors gracefully

### Type Safety
- MUST use reflection safely with proper error handling
- MUST validate factory function signatures
- MUST ensure type compatibility during resolution
- MUST provide clear error messages for type mismatches

### Observability Integration
- MUST integrate with OTEL tracing for dependency resolution
- MUST provide structured logging for debugging
- MUST collect metrics on resolution performance
- MUST support health checking for container status

### Performance Requirements
- Registration MUST complete in <1ms
- Resolution MUST complete in <1ms for cached instances
- Has checks MUST complete in <100µs
- MUST scale to 1000+ registered types

## Error Scenarios

### Registration Errors
- Invalid factory function (not a function)
- Factory function with no return values
- Factory function type conflicts

### Resolution Errors  
- No factory registered for requested type
- Circular dependency detected
- Factory function execution failure
- Target parameter type mismatch

### Health Check Errors
- Container internal state corruption
- Critical dependency unavailable
- Memory exhaustion or resource limits

## Testing Contract

### Required Test Coverage
- Factory registration with various function signatures
- Dependency resolution with complex graphs
- Circular dependency detection
- Concurrent registration and resolution
- Health check scenarios
- Error condition testing
- Performance benchmarking

### Mock Implementation Requirements
- AdvancedMockContainer with configurable behavior
- Simulation of registration and resolution failures
- Performance simulation with delays
- Health status simulation
