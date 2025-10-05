# Schema Package Standards Compliance - Quick Start Guide

**Generated**: October 5, 2025  
**For**: Framework developers and contributors working with the schema package

## Overview

This guide provides a quick introduction to using the enhanced schema package after standards compliance improvements. The schema package now includes comprehensive benchmarking, organized mock infrastructure, health checks, enhanced testing patterns, and complete OTEL tracing.

## Quick Setup

### 1. Install Required Tools

```bash
# Install mockery for mock generation
go install github.com/vektra/mockery/v2@latest

# Verify installation
mockery --version
```

### 2. Generate Mocks

```bash
# Navigate to schema package
cd pkg/schema

# Generate mocks (will create internal/mock/ directory)
go generate ./...

# Verify mocks were created
ls internal/mock/
```

### 3. Run New Test Suite

```bash
# Run all tests including new advanced tests
go test ./pkg/schema/... -v

# Run benchmarks 
go test ./pkg/schema/... -bench=. -benchmem

# Run race condition tests
go test ./pkg/schema/... -race

# Run integration tests
go test ./tests/integration/... -v
```

## Using New Features

### 1. Benchmark Testing

The package now includes comprehensive benchmarks for all performance-critical operations:

```bash
# Run specific benchmark categories
go test ./pkg/schema/ -bench=BenchmarkMessage -benchmem
go test ./pkg/schema/ -bench=BenchmarkFactory -benchmem  
go test ./pkg/schema/ -bench=BenchmarkValidation -benchmem
go test ./pkg/schema/ -bench=BenchmarkConcurrent -benchmem

# Compare benchmarks (save baseline first)
go test ./pkg/schema/ -bench=. -benchmem > baseline.txt
# After changes:
go test ./pkg/schema/ -bench=. -benchmem > current.txt
# Compare with benchcmp tool
```

Expected performance targets:
- Message creation: < 1ms
- Factory functions: < 100μs  
- Validation operations: < 500μs
- Memory allocations: minimized

### 2. Mock Infrastructure

Use organized mocks for comprehensive testing:

```go
package mypackage

import (
    "testing"
    "github.com/lookatitude/beluga-ai/pkg/schema/internal/mock"
    "github.com/stretchr/testify/assert"
)

func TestWithMocks(t *testing.T) {
    // Use generated mock
    mockMessage := &mock.MockMessage{}
    mockMessage.On("GetType").Return("human")
    mockMessage.On("GetContent").Return("test message")
    
    // Test your code with the mock
    result := myFunction(mockMessage)
    
    // Verify expectations
    mockMessage.AssertExpectations(t)
    assert.Equal(t, expected, result)
}

// Use advanced mock with options
func TestAdvancedMocking(t *testing.T) {
    mockMessage := schema.NewAdvancedMockMessage(
        "human",
        "test content",
        schema.WithMockToolCalls([]schema.ToolCall{}),
        schema.WithMockAdditionalArgs(map[string]interface{}{
            "metadata": "test",
        }),
    )
    
    // Mock is configured and ready to use
    assert.Equal(t, "test content", mockMessage.GetContent())
}
```

### 3. Health Checks

Monitor schema package health:

```go
package main

import (
    "context"
    "log"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

func checkSchemaHealth() {
    ctx := context.Background()
    
    // Check validation health
    if err := schema.CheckValidationHealth(ctx); err != nil {
        log.Printf("Validation health check failed: %v", err)
    }
    
    // Check configuration health
    if err := schema.CheckConfigurationHealth(ctx); err != nil {
        log.Printf("Configuration health check failed: %v", err)
    }
    
    // Check metrics health
    if err := schema.CheckMetricsHealth(ctx); err != nil {
        log.Printf("Metrics health check failed: %v", err)
    }
}
```

### 4. OTEL Tracing

Use enhanced tracing capabilities:

```go
package main

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)

func createMessageWithTracing() {
    tracer := otel.Tracer("my-service")
    ctx, span := tracer.Start(context.Background(), "create-message")
    defer span.End()
    
    // Factory functions now include automatic tracing
    message := schema.NewHumanMessageWithContext(ctx, "Hello, world!")
    
    // Spans include relevant attributes automatically
    // - message.type = "human"
    // - message.length = 13
    // - operation.success = true
    
    // Use message...
}
```

### 5. Enhanced Testing Patterns

Write comprehensive table-driven tests:

```go
func TestMessageValidation(t *testing.T) {
    tests := []struct {
        name        string
        input       string
        messageType schema.MessageType
        wantError   bool
        errorCode   string
    }{
        {
            name:        "valid_human_message",
            input:       "Hello, world!",
            messageType: schema.RoleHuman,
            wantError:   false,
        },
        {
            name:        "empty_message",
            input:       "",
            messageType: schema.RoleHuman,
            wantError:   true,
            errorCode:   "ErrCodeInvalidMessage",
        },
        {
            name:        "message_too_long",
            input:       strings.Repeat("a", 10001), // Exceeds limit
            messageType: schema.RoleHuman,
            wantError:   true,
            errorCode:   "ErrCodeMessageTooLong",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            message := schema.NewChatMessage(tt.messageType, tt.input)
            err := schema.ValidateMessage(message)
            
            if tt.wantError {
                assert.Error(t, err)
                if tt.errorCode != "" {
                    assert.True(t, schema.IsSchemaError(err, tt.errorCode))
                }
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### 6. Integration Testing

Run cross-package integration tests:

```bash
# Run integration tests
go test ./tests/integration/... -v

# Run specific integration test categories
go test ./tests/integration/... -v -run TestMessageFlow
go test ./tests/integration/... -v -run TestConfiguration
```

Example integration test:

```go
func TestSchemaLLMIntegration(t *testing.T) {
    // Test that schema messages work correctly with LLM package
    ctx := context.Background()
    
    // Create message using schema
    message := schema.NewHumanMessage("Test prompt")
    
    // Use with LLM package (integration test)
    // This validates that schema contracts work across packages
    response, err := llm.ProcessMessage(ctx, message)
    
    assert.NoError(t, err)
    assert.NotNil(t, response)
    
    // Validate response is also valid schema message
    assert.Implements(t, (*schema.Message)(nil), response)
}
```

## Performance Monitoring

### Monitor Benchmark Results

```bash
# Create performance monitoring script
cat > scripts/perf-monitor.sh << 'EOF'
#!/bin/bash
echo "Running schema package benchmarks..."
go test ./pkg/schema/ -bench=. -benchmem -count=5 > perf-results.txt
echo "Results saved to perf-results.txt"

# Check for performance regressions
if [ -f "baseline-perf.txt" ]; then
    echo "Comparing with baseline..."
    # Use benchcmp or similar tool to compare results
fi
EOF

chmod +x scripts/perf-monitor.sh
./scripts/perf-monitor.sh
```

### Monitor Health Checks

```go
// Add to your monitoring/health check endpoints
func healthHandler(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    health := map[string]interface{}{
        "timestamp": time.Now(),
        "schema": map[string]interface{}{
            "validation": checkHealth(ctx, schema.CheckValidationHealth),
            "config":     checkHealth(ctx, schema.CheckConfigurationHealth),
            "metrics":    checkHealth(ctx, schema.CheckMetricsHealth),
        },
    }
    
    json.NewEncoder(w).Encode(health)
}
```

## Troubleshooting

### Common Issues

1. **Mock generation fails**:
   ```bash
   # Check mockery installation
   which mockery
   
   # Regenerate with verbose output
   mockery --all --output internal/mock --outpkg mock --verbose
   ```

2. **Benchmark performance regression**:
   ```bash
   # Check for memory leaks
   go test ./pkg/schema/ -bench=BenchmarkMessage -benchmem -memprofile=mem.prof
   go tool pprof mem.prof
   
   # Check CPU usage
   go test ./pkg/schema/ -bench=BenchmarkMessage -cpuprofile=cpu.prof
   go tool pprof cpu.prof
   ```

3. **Integration test failures**:
   ```bash
   # Run with verbose output
   go test ./tests/integration/... -v -timeout 30s
   
   # Check for race conditions
   go test ./tests/integration/... -race
   ```

4. **Health check failures**:
   ```bash
   # Check logs for specific health check errors
   # Verify OTEL configuration
   # Ensure all dependencies are available
   ```

### Getting Help

- Check the enhanced README.md for detailed documentation
- Review the migration guide for upgrading existing code
- Look at examples in the test files for usage patterns
- Consult the constitutional requirements document

## Next Steps

1. **Integrate into CI/CD**: Add benchmark monitoring to your continuous integration
2. **Add Custom Benchmarks**: Create benchmarks for your specific use cases
3. **Extend Mocks**: Add custom mock behaviors for complex test scenarios
4. **Monitor Health**: Set up alerting for health check failures
5. **Contribute**: Submit improvements and additional test cases

This enhanced schema package now provides enterprise-grade testing, observability, and maintainability while preserving all existing functionality and extensibility patterns.
