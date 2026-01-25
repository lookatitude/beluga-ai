---
name: Create Test Suite
description: Comprehensive test suite creation following Beluga AI test standards
personas:
  - qa
---

# Create Test Suite

This skill guides you through creating a comprehensive test suite for a Beluga AI package, ensuring all test categories are covered.

## Prerequisites

- Package to test exists
- Understanding of package functionality
- Access to similar test examples in codebase

## Steps

### 1. Assess Testing Needs

Identify what needs testing:

```markdown
## Test Assessment: pkg/[package]

### Public API
- [ ] `NewComponent()` - Factory function
- [ ] `Component.Method1()` - [Description]
- [ ] `Component.Method2()` - [Description]

### Providers (if multi-provider)
- [ ] Provider A
- [ ] Provider B

### Configuration
- [ ] Validation
- [ ] Defaults
- [ ] Functional options

### Error Scenarios
- [ ] Invalid input
- [ ] Timeout
- [ ] Rate limit
- [ ] Configuration errors

### Edge Cases
- [ ] Empty input
- [ ] Large input
- [ ] Concurrent access
- [ ] Context cancellation
```

### 2. Create test_utils.go

```go
package mypackage

import (
    "sync"
    "sync/atomic"
    "time"

    "github.com/stretchr/testify/mock"
)

// AdvancedMockComponent provides configurable mock behavior
type AdvancedMockComponent struct {
    mock.Mock
    config      MockConfig
    healthState atomic.Bool
    callCount   atomic.Int64
}

type MockConfig struct {
    Error        error
    Delay        time.Duration
    ResponseFunc func(input string) string
}

type MockOption func(*AdvancedMockComponent)

func NewAdvancedMockComponent(opts ...MockOption) *AdvancedMockComponent {
    m := &AdvancedMockComponent{}
    m.healthState.Store(true)
    for _, opt := range opts {
        opt(m)
    }
    return m
}

func WithMockError(err error) MockOption {
    return func(m *AdvancedMockComponent) {
        m.config.Error = err
    }
}

func WithMockDelay(d time.Duration) MockOption {
    return func(m *AdvancedMockComponent) {
        m.config.Delay = d
    }
}

func WithMockResponse(fn func(string) string) MockOption {
    return func(m *AdvancedMockComponent) {
        m.config.ResponseFunc = fn
    }
}

// ConcurrentTestRunner helps test race conditions
type ConcurrentTestRunner struct {
    goroutines int
    errors     []error
    mu         sync.Mutex
    wg         sync.WaitGroup
}

func NewConcurrentTestRunner(n int) *ConcurrentTestRunner {
    return &ConcurrentTestRunner{goroutines: n}
}

func (r *ConcurrentTestRunner) Run(fn func() error) {
    r.wg.Add(r.goroutines)
    for i := 0; i < r.goroutines; i++ {
        go func() {
            defer r.wg.Done()
            if err := fn(); err != nil {
                r.mu.Lock()
                r.errors = append(r.errors, err)
                r.mu.Unlock()
            }
        }()
    }
}

func (r *ConcurrentTestRunner) Wait() error {
    r.wg.Wait()
    if len(r.errors) > 0 {
        return r.errors[0]
    }
    return nil
}

// LoadTestConfig for performance testing
type LoadTestConfig struct {
    Operations   int
    Concurrency  int
    Duration     time.Duration
}

type LoadTestResult struct {
    TotalOps    int
    SuccessOps  int
    FailedOps   int
    SuccessRate float64
    AvgLatency  time.Duration
    P50Latency  time.Duration
    P95Latency  time.Duration
    P99Latency  time.Duration
}

func RunLoadTest(config LoadTestConfig, fn func() error) LoadTestResult {
    // Implementation...
    return LoadTestResult{}
}
```

### 3. Create advanced_test.go

```go
package mypackage

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// ============================================
// Table-Driven Tests
// ============================================

func TestAdvancedComponent(t *testing.T) {
    tests := []struct {
        name          string
        input         Input
        config        Config
        expected      Output
        expectedError bool
        errorCode     ErrorCode
    }{
        {
            name:     "success - basic input",
            input:    Input{Value: "test"},
            config:   DefaultConfig(),
            expected: Output{Result: "processed"},
        },
        {
            name:     "success - complex input",
            input:    Input{Value: "complex", Options: []string{"a", "b"}},
            config:   DefaultConfig(),
            expected: Output{Result: "processed-complex"},
        },
        {
            name:          "error - empty input",
            input:         Input{},
            config:        DefaultConfig(),
            expectedError: true,
            errorCode:     ErrCodeInvalidInput,
        },
        {
            name:          "error - invalid config",
            input:         Input{Value: "test"},
            config:        Config{Timeout: 0},
            expectedError: true,
            errorCode:     ErrCodeInvalidConfig,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            component, err := NewComponent(tt.config)
            if tt.expectedError && err != nil {
                // Config validation error
                return
            }
            require.NoError(t, err)

            result, err := component.Process(context.Background(), tt.input)

            if tt.expectedError {
                require.Error(t, err)
                var e *Error
                require.ErrorAs(t, err, &e)
                assert.Equal(t, tt.errorCode, e.Code)
            } else {
                require.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}

// ============================================
// Concurrency Tests
// ============================================

func TestConcurrency(t *testing.T) {
    component, err := NewComponent(DefaultConfig())
    require.NoError(t, err)

    runner := NewConcurrentTestRunner(100)

    runner.Run(func() error {
        _, err := component.Process(context.Background(), Input{Value: "test"})
        return err
    })

    require.NoError(t, runner.Wait())
}

func TestConcurrentAccess(t *testing.T) {
    component, err := NewComponent(DefaultConfig())
    require.NoError(t, err)

    var wg sync.WaitGroup
    errors := make(chan error, 100)

    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            _, err := component.Process(context.Background(), Input{
                Value: fmt.Sprintf("test-%d", id),
            })
            if err != nil {
                errors <- err
            }
        }(i)
    }

    wg.Wait()
    close(errors)

    for err := range errors {
        t.Errorf("concurrent error: %v", err)
    }
}

// ============================================
// Load Tests
// ============================================

func TestLoad(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping load test in short mode")
    }

    component, err := NewComponent(DefaultConfig())
    require.NoError(t, err)

    result := RunLoadTest(LoadTestConfig{
        Operations:  1000,
        Concurrency: 10,
        Duration:    30 * time.Second,
    }, func() error {
        _, err := component.Process(context.Background(), Input{Value: "test"})
        return err
    })

    assert.True(t, result.SuccessRate > 0.99, "success rate should be > 99%%")
    assert.Less(t, result.P99Latency, 100*time.Millisecond, "P99 latency should be < 100ms")
}

// ============================================
// Error Scenario Tests
// ============================================

func TestErrorScenarios(t *testing.T) {
    tests := []struct {
        name     string
        setup    func() *Component
        input    Input
        expected ErrorCode
    }{
        {
            name: "timeout error",
            setup: func() *Component {
                c, _ := NewComponent(Config{Timeout: 1 * time.Nanosecond})
                return c
            },
            input:    Input{Value: "slow"},
            expected: ErrCodeTimeout,
        },
        {
            name: "context cancelled",
            setup: func() *Component {
                c, _ := NewComponent(DefaultConfig())
                return c
            },
            input:    Input{Value: "test"},
            expected: ErrCodeCancelled,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            component := tt.setup()

            ctx, cancel := context.WithCancel(context.Background())
            if tt.expected == ErrCodeCancelled {
                cancel()
            } else {
                defer cancel()
            }

            _, err := component.Process(ctx, tt.input)
            require.Error(t, err)

            var e *Error
            require.ErrorAs(t, err, &e)
            assert.Equal(t, tt.expected, e.Code)
        })
    }
}

// ============================================
// OTEL Validation Tests
// ============================================

func TestMetricsRecorded(t *testing.T) {
    // Setup test meter
    reader := metric.NewManualReader()
    provider := metric.NewMeterProvider(metric.WithReader(reader))
    meter := provider.Meter("test")

    metrics, err := NewMetrics(meter, nil)
    require.NoError(t, err)

    component, err := NewComponent(DefaultConfig(), WithMetrics(metrics))
    require.NoError(t, err)

    // Execute operation
    _, _ = component.Process(context.Background(), Input{Value: "test"})

    // Collect and verify metrics
    rm := &metricdata.ResourceMetrics{}
    require.NoError(t, reader.Collect(context.Background(), rm))

    // Verify operations_total was incremented
    // Verify duration_seconds was recorded
}

// ============================================
// Registry Tests (for multi-provider packages)
// ============================================

func TestProviderRegistry(t *testing.T) {
    // Test registration
    RegisterGlobal("test", func(ctx context.Context, config Config) (Interface, error) {
        return NewMockProvider(), nil
    })

    // Test creation
    provider, err := NewProvider(context.Background(), "test", DefaultConfig())
    require.NoError(t, err)
    require.NotNil(t, provider)

    // Test unknown provider
    _, err = NewProvider(context.Background(), "unknown", DefaultConfig())
    require.Error(t, err)
}

// ============================================
// Benchmarks
// ============================================

func BenchmarkComponentProcess(b *testing.B) {
    component, _ := NewComponent(DefaultConfig())
    input := Input{Value: "test"}
    ctx := context.Background()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = component.Process(ctx, input)
    }
}

func BenchmarkComponentConcurrent(b *testing.B) {
    component, _ := NewComponent(DefaultConfig())
    input := Input{Value: "test"}
    ctx := context.Background()

    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, _ = component.Process(ctx, input)
        }
    })
}
```

### 4. Run and Verify Tests

```bash
# Run with race detection
go test -race -v ./pkg/mypackage/...

# Run with coverage
go test -coverprofile=coverage.out ./pkg/mypackage/...
go tool cover -func=coverage.out

# Run benchmarks
go test -bench=. -benchmem ./pkg/mypackage/...
```

## Test Category Checklist

| Category | Required | Created |
|----------|----------|---------|
| Table-driven tests | Yes | [ ] |
| Concurrency tests | Yes | [ ] |
| Load tests | If applicable | [ ] |
| Error scenario tests | Yes | [ ] |
| OTEL validation tests | Yes | [ ] |
| Registry tests | If multi-provider | [ ] |
| Benchmarks | Yes | [ ] |

## Coverage Targets

| Level | Target |
|-------|--------|
| Critical paths | 100% |
| Overall package | ≥80% |

## Output

A comprehensive test suite with:
- `test_utils.go` - Mocks and helpers
- `advanced_test.go` - All test categories
- `{package}_test.go` - Basic unit tests
- Coverage meeting thresholds
- All tests passing with race detection
