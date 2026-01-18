# Quickstart: Comprehensive Test Coverage Improvement

**Feature**: 001-comprehensive-test-coverage  
**Date**: 2026-01-16

## Overview

This quickstart guide provides step-by-step instructions for implementing comprehensive test coverage improvements across all packages in the Beluga AI Framework.

## Prerequisites

- Go 1.24.1+ installed
- Access to Beluga AI Framework repository
- Understanding of existing testing patterns (see `pkg/llms/test_utils.go` for reference)
- Familiarity with Go testing and coverage tools

## Step 1: Analyze Current Coverage

### 1.1 Generate Coverage Report

```bash
# Generate coverage report for all packages
make test-coverage

# View HTML report
open coverage/coverage.html

# Check coverage percentage
go tool cover -func=coverage/coverage.out | tail -n1
```

### 1.2 Identify Coverage Gaps

```bash
# List packages with coverage below 100%
go tool cover -func=coverage/coverage.out | grep -E "pkg/[^/]+/[^/]+\.go" | awk '$3 < 100.0'
```

### 1.3 Identify Missing Mocks

```bash
# Find packages with external dependencies
grep -r "http\.Client\|net/http\|APIKey\|api_key" pkg/*/ --include="*.go" | grep -v test
```

## Step 2: Create/Update test_utils.go

### 2.1 Template Structure

For each package requiring mocks, create or update `test_utils.go`:

```go
package packagename

import (
    "context"
    "errors"
    "sync"
    "time"
    "github.com/stretchr/testify/mock"
)

// AdvancedMockInterface provides comprehensive mock implementation
type AdvancedMockInterface struct {
    mock.Mock
    mu sync.RWMutex
    callCount int
    shouldError bool
    errorToReturn error
    simulateDelay time.Duration
}

// NewAdvancedMockInterface creates a new advanced mock
func NewAdvancedMockInterface(opts ...MockOption) *AdvancedMockInterface {
    m := &AdvancedMockInterface{}
    for _, opt := range opts {
        opt(m)
    }
    return m
}

// MockOption configures mock behavior
type MockOption func(*AdvancedMockInterface)

func WithMockError(shouldError bool, err error) MockOption {
    return func(m *AdvancedMockInterface) {
        m.shouldError = shouldError
        m.errorToReturn = err
    }
}

func WithMockDelay(delay time.Duration) MockOption {
    return func(m *AdvancedMockInterface) {
        m.simulateDelay = delay
    }
}

// Implement interface methods with configurable behavior
// Support all common error types: network errors, API errors, timeouts, 
// rate limits, authentication failures, invalid requests, service unavailable
```

### 2.2 Error Scenario Support

Ensure mocks support all common error types:

```go
// Example error types to support
var (
    ErrNetworkTimeout = errors.New("network timeout")
    ErrConnectionFailed = errors.New("connection failed")
    ErrRateLimit = errors.New("rate limit exceeded")
    ErrAuthFailed = errors.New("authentication failed")
    ErrInvalidRequest = errors.New("invalid request")
    ErrServiceUnavailable = errors.New("service unavailable")
)
```

## Step 3: Create/Update advanced_test.go

### 3.1 Template Structure

For each package, create or update `advanced_test.go`:

```go
package packagename

import (
    "context"
    "testing"
    "time"
    "github.com/stretchr/testify/assert"
)

func TestAdvancedPackageName(t *testing.T) {
    tests := []struct {
        name string
        setup func() *Component
        testFn func(t *testing.T, comp *Component)
        expectedError bool
    }{
        {
            name: "successful operation",
            setup: func() *Component {
                return NewComponent()
            },
            testFn: func(t *testing.T, comp *Component) {
                ctx := context.Background()
                result, err := comp.DoOperation(ctx, "input")
                assert.NoError(t, err)
                assert.NotNil(t, result)
            },
            expectedError: false,
        },
        // Add comprehensive test cases covering all scenarios
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            comp := tt.setup()
            tt.testFn(t, comp)
        })
    }
}

func TestConcurrencyAdvanced(t *testing.T) {
    // Concurrency tests
}

func TestLoadTesting(t *testing.T) {
    // Load tests
}

func BenchmarkPackageOperations(b *testing.B) {
    // Benchmarks
}
```

### 3.2 Coverage Requirements

Ensure tests cover:
- All public API methods
- All error handling paths
- All edge cases
- All state transitions
- All configuration options

## Step 4: Document Exclusions

### 4.1 Create Exclusion Documentation

For untestable code paths, document in package README or test_utils.go:

```go
// EXCLUSIONS: Code paths excluded from 100% coverage requirement
//
// 1. pkg/example/example.go:handlePanic()
//    Reason: Panic handler cannot be tested without causing actual panic
//    Date: 2026-01-16
//    Reviewer: [name]
//
// 2. pkg/example/example.go:osSpecificCode()
//    Reason: OS-specific code requires platform-specific testing infrastructure
//    Date: 2026-01-16
//    Reviewer: [name]
```

### 4.2 Review Exclusions

Periodically review exclusions to determine if testing approaches can be improved.

## Step 5: Create Integration Tests

### 5.1 Identify Direct Dependencies

```bash
# Find direct package dependencies
go list -f '{{.ImportPath}} {{.Imports}}' ./pkg/... | grep -E "pkg/"
```

### 5.2 Create Integration Test Files

For each direct dependency pair, create integration test in `tests/integration/package_pairs/`:

```go
package package_pairs

import (
    "context"
    "testing"
    "github.com/lookatitude/beluga-ai/pkg/package1"
    "github.com/lookatitude/beluga-ai/pkg/package2"
    "github.com/stretchr/testify/assert"
)

func TestPackage1Package2Integration(t *testing.T) {
    // Test integration between package1 and package2
    ctx := context.Background()
    
    // Setup
    p1 := package1.New(...)
    p2 := package2.New(...)
    
    // Execute integration scenario
    result, err := p1.UsePackage2(ctx, p2)
    
    // Verify
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

## Step 6: Verify Coverage

### 6.1 Run Coverage Analysis

```bash
# Generate coverage report
make test-coverage

# Verify 100% coverage (excluding documented exclusions)
go tool cover -func=coverage/coverage.out
```

### 6.2 Verify Mocks

```bash
# Run tests without network access
# (Disconnect network or use network isolation)
make test-unit

# All tests should pass using mocks
```

### 6.3 Verify Integration Tests

```bash
# Run integration tests
make test-integration

# Verify 80%+ integration coverage
# (Coverage analysis for integration tests)
```

## Step 7: Generate Reports

### 7.1 HTML Report

```bash
# Generate HTML report
go tool cover -html=coverage/coverage.out -o coverage/coverage.html

# Open in browser
open coverage/coverage.html
```

### 7.2 Machine-Readable Report

```bash
# Parse coverage.out for JSON/XML
# (Use coverage.out format or custom parser)
go tool cover -func=coverage/coverage.out -o coverage/coverage.json
```

## Step 8: Maintain Patterns

### 8.1 Verify Pattern Compliance

- All packages have `test_utils.go` with AdvancedMock pattern
- All packages have `advanced_test.go` with comprehensive tests
- All mocks follow functional options pattern
- All tests follow table-driven pattern where applicable

### 8.2 Code Review Checklist

- [ ] test_utils.go follows AdvancedMock pattern
- [ ] advanced_test.go includes comprehensive test cases
- [ ] All error scenarios are covered
- [ ] Exclusions are documented with justification
- [ ] Integration tests cover direct dependencies
- [ ] Coverage reports are generated in both formats

## Common Patterns

### Mock with Error Simulation

```go
mock := NewAdvancedMockInterface(
    WithMockError(true, ErrRateLimit),
)
```

### Mock with Delay Simulation

```go
mock := NewAdvancedMockInterface(
    WithMockDelay(100 * time.Millisecond),
)
```

### Table-Driven Test

```go
tests := []struct {
    name string
    input string
    expected string
    expectError bool
}{
    // Test cases
}
```

## Troubleshooting

### Coverage Not Reaching 100%

1. Check for undocumented exclusions
2. Review error handling paths
3. Verify all public methods are tested
4. Check for unreachable code (may indicate dead code)

### Mocks Not Working

1. Verify mock implements correct interface
2. Check mock configuration options
3. Ensure tests use mocks instead of real implementations
4. Verify mock is thread-safe for concurrent tests

### Integration Tests Failing

1. Check direct dependency relationships
2. Verify package initialization
3. Review test setup/teardown
4. Check for resource leaks

## Next Steps

After completing quickstart:

1. Review generated coverage reports
2. Address any coverage gaps
3. Document any additional exclusions
4. Create integration tests for missing dependency pairs
5. Update package documentation with test examples

## References

- Framework Testing Standards: `.cursor/rules/beluga-test-standards.mdc`
- Framework Design Patterns: `.cursor/rules/beluga-design-patterns.mdc`
- Example AdvancedMock: `pkg/llms/test_utils.go`
- Example Advanced Tests: `pkg/llms/advanced_test.go`
