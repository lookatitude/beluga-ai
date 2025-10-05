# Runnable Interface Contract

## Interface Definition

```go
type Runnable interface {
    // Invoke executes the runnable component with a single input and returns a single output.
    // Context MUST support cancellation and timeout
    // Options MUST support functional configuration pattern
    Invoke(ctx context.Context, input any, options ...Option) (any, error)

    // Batch executes the runnable component with multiple inputs concurrently or sequentially.
    // MUST handle partial failures gracefully
    // MUST respect context cancellation
    Batch(ctx context.Context, inputs []any, options ...Option) ([]any, error)

    // Stream executes the runnable component with streaming output.
    // Channel MUST be closed when complete
    // Errors MUST be sent through channel or returned immediately
    Stream(ctx context.Context, input any, options ...Option) (<-chan any, error)
}
```

## Contract Requirements

### Invoke Method
- **Input**: `ctx` (cancellation), `input` (any type), `options` (variadic functional options)
- **Output**: `any` (result), `error` (structured error with operation context)
- **Behavior**: Synchronous execution, MUST respect context cancellation
- **Error Handling**: MUST return FrameworkError with operation context

### Batch Method  
- **Input**: `ctx` (cancellation), `inputs` (slice of any), `options` (variadic functional options)
- **Output**: `[]any` (results matching input order), `error` (if complete failure)
- **Behavior**: Concurrent or sequential processing, partial failure handling
- **Error Handling**: Individual failures vs complete failure, detailed error reporting

### Stream Method
- **Input**: `ctx` (cancellation), `input` (any type), `options` (variadic functional options)  
- **Output**: `<-chan any` (streaming results), `error` (setup errors)
- **Behavior**: Asynchronous streaming, channel closed on completion
- **Error Handling**: Errors sent through channel with proper typing

## Implementation Requirements

### Thread Safety
- ALL implementations MUST be thread-safe
- Concurrent access to Invoke, Batch, Stream MUST be supported
- Internal state MUST be protected with appropriate synchronization

### Context Handling
- MUST respect context cancellation in all methods
- MUST propagate context to all underlying operations
- MUST handle context timeout gracefully

### Performance Requirements  
- Invoke method MUST complete in <100µs overhead
- Batch method MUST scale linearly with input size
- Stream method MUST have <10ms setup time

### Error Requirements
- MUST use structured error types (FrameworkError)
- MUST include operation context in errors
- MUST preserve error chains through Unwrap()
- MUST provide meaningful error messages

## Testing Contract

### Required Test Coverage
- Unit tests for each method with various input types
- Concurrency tests with multiple goroutines
- Context cancellation tests
- Error scenario tests
- Performance benchmark tests

### Mock Implementation Requirements
- AdvancedMockRunnable MUST support configurable behavior
- Mock options for error simulation, delay simulation
- Call count tracking for test verification
- Health check simulation
