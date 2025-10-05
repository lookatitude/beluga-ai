# Quickstart: Core Package Constitutional Compliance Enhancement

## Quick Start Guide

This guide demonstrates how to use the enhanced core package with full constitutional compliance while maintaining all existing functionality.

### 1. Basic Dependency Injection Setup

```go
package main

import (
    "context"
    "log"
    
    "github.com/lookatitude/beluga-ai/pkg/core"
    "github.com/lookatitude/beluga-ai/pkg/core/iface"
    "go.opentelemetry.io/otel"
)

func main() {
    // Create container with OTEL observability
    meter := otel.Meter("beluga.core.quickstart")
    tracer := otel.Tracer("beluga.core.quickstart")
    
    container := core.NewContainerWithOptions(
        core.WithLogger(core.NewStructuredLogger()),
        core.WithTracerProvider(otel.GetTracerProvider()),
    )
    
    // Register a service factory
    container.Register(func() MyAIService {
        return &myAIServiceImpl{}
    })
    
    // Resolve and use the service
    var service MyAIService
    if err := container.Resolve(&service); err != nil {
        log.Fatal("Failed to resolve service:", err)
    }
    
    // Service is ready for AI workflows
    result, err := service.ProcessData(context.Background(), inputData)
}
```

### 2. Creating Runnable Components

```go
// Implement Runnable interface for AI components
type MyAIComponent struct {
    name string
    config *core.Config
}

// Constitutional compliance: all methods support context and options
func (c *MyAIComponent) Invoke(ctx context.Context, input any, options ...iface.Option) (any, error) {
    // Apply functional options
    config := make(map[string]any)
    for _, opt := range options {
        opt.Apply(&config)
    }
    
    // Process with constitutional error handling
    result, err := c.processInput(ctx, input, config)
    if err != nil {
        return nil, core.NewValidationError("AI component processing failed", err)
    }
    
    return result, nil
}

func (c *MyAIComponent) Batch(ctx context.Context, inputs []any, options ...iface.Option) ([]any, error) {
    results := make([]any, len(inputs))
    for i, input := range inputs {
        result, err := c.Invoke(ctx, input, options...)
        if err != nil {
            return nil, err
        }
        results[i] = result
    }
    return results, nil
}

func (c *MyAIComponent) Stream(ctx context.Context, input any, options ...iface.Option) (<-chan any, error) {
    ch := make(chan any, 1)
    go func() {
        defer close(ch)
        result, err := c.Invoke(ctx, input, options...)
        if err != nil {
            // In real implementation, would send error through channel
            return
        }
        ch <- result
    }()
    return ch, nil
}
```

### 3. Advanced Testing with Constitutional Utilities

```go
func TestMyAIComponent(t *testing.T) {
    // Use constitutional testing utilities
    helper := core.NewIntegrationTestHelper()
    
    // Create mock container
    mockContainer := core.NewAdvancedMockContainer("test-container",
        core.WithMockError(false, nil),
        core.WithMockDelay(0),
    )
    
    // Test DI resolution
    container := helper.CreateMockContainer("test")
    var component MyAIComponent
    err := container.Resolve(&component)
    assert.NoError(t, err)
    
    // Test Runnable interface compliance
    runner := core.NewRunnableScenarioRunner(&component)
    err = runner.RunExecutionScenario(context.Background(), []string{"test1", "test2"})
    assert.NoError(t, err)
    
    // Performance testing
    core.RunLoadTest(t, &component, 100, 10)
}
```

### 4. Configuration Management

```go
// Create configuration with functional options
config := core.NewConfig(
    core.WithTimeout(30 * time.Second),
    core.WithMaxRetries(3),
    core.WithObservability(true, true, true), // metrics, tracing, logging
)

// Use configuration in components
component := NewMyComponent(config)
```

### 5. Health Monitoring Integration

```go
// Check component health
if err := container.CheckHealth(ctx); err != nil {
    log.Printf("Core container unhealthy: %v", err)
    // Handle degraded state
}

// Health check with custom components
healthChecker := &MyHealthChecker{}
if err := healthChecker.CheckHealth(ctx); err != nil {
    log.Printf("Custom component unhealthy: %v", err)
}
```

### 6. Observability Integration

```go
// Metrics collection
metrics, err := core.NewMetrics(meter, tracer)
if err != nil {
    log.Fatal("Failed to create metrics:", err)
}

// Record operations with OTEL
metrics.RecordRunnableInvoke(ctx, "MyAIComponent", duration, err)

// Tracing integration  
tracedComponent := core.NewTracedRunnable(
    component,
    tracer,
    metrics,
    "ai_component",
    "my_component",
)

// Component now has automatic tracing
result, err := tracedComponent.Invoke(ctx, input)
```

## Validation Steps

### 1. Verify Constitutional Compliance
```bash
# Run comprehensive tests
go test ./pkg/core/... -v

# Run advanced test suites  
go test ./pkg/core/advanced_test.go -v

# Run benchmarks
go test ./pkg/core/... -bench=. -benchmem

# Run concurrency tests
go test ./pkg/core/... -race
```

### 2. Verify Integration with Other Packages
```bash
# Test LLMs integration
go test ./tests/integration/package_pairs/llms_core_test.go -v

# Test agents integration  
go test ./tests/integration/package_pairs/agents_core_test.go -v

# Test memory integration
go test ./tests/integration/package_pairs/memory_core_test.go -v
```

### 3. Verify Performance Requirements
```bash
# DI resolution performance
go test ./pkg/core/ -bench=BenchmarkContainer -benchmem

# Runnable performance  
go test ./pkg/core/ -bench=BenchmarkRunnable -benchmem

# Metrics performance
go test ./pkg/core/ -bench=BenchmarkMetrics -benchmem
```

### 4. Verify Health Monitoring
```bash
# Health check functionality
go test ./pkg/core/ -run=TestHealthCheck

# Integration with monitoring systems
go test ./tests/integration/observability/ -v
```

## Success Criteria

- [x] All existing functionality preserved (DI, Runnable, utilities)
- [ ] Constitutional package structure implemented (iface/, config.go, test_utils.go)
- [ ] Advanced testing infrastructure complete
- [ ] Zero breaking changes to existing APIs  
- [ ] All performance benchmarks pass
- [ ] 100% test coverage achieved
- [ ] Integration tests with dependent packages pass
- [ ] OTEL observability fully integrated
- [ ] Health monitoring operational

## Rollback Plan

If constitutional compliance introduces issues:
1. Revert interface moves while keeping iface/ exports
2. Remove config.go if causing import conflicts
3. Keep advanced testing utilities as they provide value
4. Maintain OTEL metrics integration (already working)

The enhancement is designed to be additive and non-breaking, minimizing rollback risk.
