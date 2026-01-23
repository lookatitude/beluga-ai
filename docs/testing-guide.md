# Beluga AI Testing Guide

**Purpose**: Document testing patterns and best practices for the Beluga AI Framework  
**Last Updated**: 2026-01-16

## Overview

This guide documents the testing patterns, conventions, and best practices used throughout the Beluga AI Framework. All packages follow consistent testing patterns to ensure maintainability and reliability.

## Required Test Files

Every package in `pkg/` must include:

1. **`test_utils.go`** - Advanced mock implementations and test utilities
2. **`advanced_test.go`** - Comprehensive test suites with table-driven tests, concurrency tests, and benchmarks

### test_utils.go Structure

```go
// AdvancedMock{PackageName} provides a comprehensive mock implementation
type AdvancedMock{PackageName} struct {
    mock.Mock
    // Configuration fields
    // State fields
    // Error handling fields
}

// NewAdvancedMock{PackageName} creates a new advanced mock
func NewAdvancedMock{PackageName}(name string, opts ...MockOption) *AdvancedMock{PackageName} {
    // Implementation
}

// Mock options for configurable behavior
func WithMockError(shouldError bool, err error) MockOption { ... }
func WithMockDelay(delay time.Duration) MockOption { ... }
```

### advanced_test.go Structure

```go
// TestAdvanced{PackageName} provides comprehensive table-driven tests
func TestAdvanced{PackageName}(t *testing.T) {
    tests := []struct {
        name        string
        // Test parameters
        expectedErr bool
    }{
        // Test cases
    }
    // Test execution
}

// TestConcurrent{PackageName} tests concurrency
func TestConcurrent{PackageName}(t *testing.T) {
    // Concurrent test implementation
}

// Benchmark{PackageName}Operations benchmarks operations
func Benchmark{PackageName}Operations(b *testing.B) {
    // Benchmark implementation
}
```

## Testing Patterns

### 1. Table-Driven Tests

All test suites use table-driven tests for consistency and maintainability:

```go
tests := []struct {
    name        string
    input       InputType
    expected    ExpectedType
    expectedErr bool
}{
    {
        name:        "successful_operation",
        input:       validInput,
        expected:    expectedResult,
        expectedErr: false,
    },
    {
        name:        "error_case",
        input:       invalidInput,
        expectedErr: true,
    },
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        result, err := operation(tt.input)
        if tt.expectedErr {
            require.Error(t, err)
        } else {
            require.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        }
    })
}
```

### 2. Concurrency Testing

Test concurrent access and operations:

```go
func TestConcurrentOperations(t *testing.T) {
    const numGoroutines = 20
    var wg sync.WaitGroup
    wg.Add(numGoroutines)
    
    for i := 0; i < numGoroutines; i++ {
        go func() {
            defer wg.Done()
            // Test operation
        }()
    }
    
    wg.Wait()
    // Verify results
}
```

### 3. Error Handling Tests

Test all error types and error propagation:

```go
func TestErrorHandling(t *testing.T) {
    tests := []struct {
        name        string
        createError func() error
        validate    func(t *testing.T, err error)
    }{
        {
            name: "basic_error",
            createError: func() error {
                return NewPackageError("op", ErrCodeInvalidInput, nil)
            },
            validate: func(t *testing.T, err error) {
                assert.Error(t, err)
                assert.True(t, IsPackageError(err))
            },
        },
    }
    // Test execution
}
```

### 4. Mock Implementations

Use AdvancedMock pattern for all external dependencies:

```go
mock := NewAdvancedMockDependency("test-mock",
    WithMockError(false, nil),
    WithMockDelay(100*time.Millisecond),
)

// Use mock in tests
result, err := mock.Operation(ctx, input)
```

### 5. Integration Tests

Create integration tests for cross-package interactions:

```go
func TestIntegrationPackage1Package2(t *testing.T) {
    helper := utils.NewIntegrationTestHelper()
    defer helper.Cleanup(ctx)
    
    // Setup components
    comp1 := helper.CreateComponent1()
    comp2 := helper.CreateComponent2()
    
    // Test integration
    err := comp1.Use(comp2)
    require.NoError(t, err)
}
```

## Coverage Requirements

### Unit Test Coverage
- **Target**: 100% coverage (excluding documented exclusions)
- **Minimum**: 80% coverage
- **Critical paths**: MUST be 100% covered

### Integration Test Coverage
- **Target**: 80%+ coverage for cross-package interactions
- **Focus**: Direct package dependencies

## Exclusion Documentation

Document untestable code paths in `test_utils.go`:

```go
// EXCLUSION DOCUMENTATION
//
// The following code paths are excluded from test coverage:
//
// 1. Provider-specific implementations
//    - Reason: Require actual external service connections
//    - Coverage: Tested via integration tests
//    - Files: pkg/{package}/providers/*/*.go
//
// 2. Internal implementations
//    - Reason: Tested indirectly through public APIs
//    - Coverage: Tested via integration tests
//    - Files: pkg/{package}/internal/*.go
```

## Best Practices

### 1. Use Mocks for External Dependencies
- Always use AdvancedMock implementations for external services
- Configure mocks with functional options
- Test error scenarios with mock errors

### 2. Test Error Propagation
- Verify errors are properly wrapped
- Test error code propagation
- Test error context preservation

### 3. Test Configuration
- Test valid configurations
- Test invalid configurations
- Test configuration validation

### 4. Test Observability
- Verify metrics are recorded
- Verify traces are created
- Verify logs are structured

### 5. Test Resource Cleanup
- Verify resources are properly cleaned up
- Test cleanup on errors
- Test cleanup on context cancellation

### 6. Test Concurrent Access
- Test thread safety
- Test race conditions
- Test concurrent operations

## Running Tests

### Unit Tests
```bash
# Run all unit tests
go test ./pkg/... -v

# Run tests for specific package
go test ./pkg/agents/... -v

# Run with coverage
go test -cover ./pkg/...

# Run with race detection
go test -race ./pkg/...
```

### Integration Tests
```bash
# Run all integration tests
go test ./tests/integration/... -v

# Run specific integration test suite
go test ./tests/integration/package_pairs/... -v

# Run with short flag (skip slow tests)
go test ./tests/integration/... -short
```

### Benchmarks
```bash
# Run benchmarks
go test -bench=. ./pkg/...

# Run specific benchmark
go test -bench=BenchmarkPackageOperations ./pkg/...
```

## Validation

Run pattern validation:

```bash
./scripts/validate-test-patterns.sh
```

This validates:
- All packages have `test_utils.go`
- All packages have `advanced_test.go`
- AdvancedMock pattern is used
- Table-driven tests are present

## Troubleshooting

### Common Issues

1. **Test Timeouts**: Increase timeout or use mocks
2. **Race Conditions**: Add proper synchronization
3. **Flaky Tests**: Check for timing dependencies
4. **Coverage Gaps**: Review exclusion documentation

### Debug Mode

Set `BELUGA_DEBUG=true` for detailed logging during tests.

## References

- [Framework Quality Standards](../.agent/rules/framework-quality.mdc)
- [Framework Architecture](../.agent/rules/framework-architecture.mdc)
- [Integration Testing Framework](../tests/README.md)
