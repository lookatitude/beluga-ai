# LLM Package Test Suite Improvements

This document outlines the comprehensive improvements made to the test suite for the `llms` package, designed to support both unit testing and integration testing scenarios.

## Overview

The improved test suite provides a robust foundation for testing LLM implementations with comprehensive coverage of:

- **Unit Tests**: Isolated testing of individual components
- **Integration Tests**: Testing with real LLM providers
- **Concurrency Tests**: Thread safety and race condition testing
- **Performance Benchmarks**: Measuring operation performance
- **Interface Compliance**: Ensuring providers implement required interfaces
- **Edge Case Testing**: Handling of error conditions and boundary cases

## New Test Files

### 1. `test_utils.go`
Advanced test utilities and comprehensive mock implementations.

**Key Features:**
- `AdvancedMockChatModel`: Configurable mock with realistic streaming and tool support
- `MockMetricsRecorder`: Mock metrics collection for testing observability
- `MockTracingHelper`: Mock tracing functionality
- `IntegrationTestHelper`: Utilities for integration testing
- Test data creation helpers (`CreateTestMessages`, `CreateTestConfig`)
- Assertion helpers (`AssertHealthCheck`, `AssertStreamingResponse`, `AssertErrorType`)

### 2. `advanced_test.go`
Comprehensive test scenarios demonstrating improved testing patterns.

**Key Features:**
- Table-driven tests for all major functionality
- Concurrency testing with `ConcurrentTestRunner`
- Load testing capabilities with `RunLoadTest`
- Integration workflow testing
- Edge case and error scenario testing
- Performance benchmarks

### 3. `provider_interface_test.go`
Interface compliance testing for LLM providers.

**Key Features:**
- `ProviderInterfaceTestSuite`: Comprehensive interface testing
- Reflection-based method signature validation
- Contract testing (idempotency, lifecycle)
- Thread safety testing
- Provider robustness testing

### 4. `integration_test_setup.go`
Integration testing infrastructure.

**Key Features:**
- `IntegrationTestConfig`: Environment-based configuration
- `IntegrationTestHelper`: Provider setup and rate limiting
- Pre-configured test suites for major providers
- Cross-provider comparison testing
- Safety limits and cost controls

## Test Categories

### Unit Tests
```go
// Basic functionality tests
func TestEnsureMessagesAdvanced(t *testing.T)

// Configuration validation
func TestConfigurationAdvanced(t *testing.T)

// Error handling
func TestErrorHandlingAdvanced(t *testing.T)

// Mock testing
func TestAdvancedMockChatModel(t *testing.T)
```

### Integration Tests
```go
// Real provider testing (requires API keys)
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration tests")
    }
    RunAllIntegrationTests(t)
}

// Provider-specific integration
func TestAnthropicIntegration(t *testing.T) {
    suite := AnthropicIntegrationTestSuite()
    RunIntegrationTestSuite(t, suite)
}
```

### Concurrency Tests
```go
// Thread safety testing
func TestConcurrencyAdvanced(t *testing.T)

// Load testing
func TestLoadTesting(t *testing.T)
```

### Interface Compliance Tests
```go
// Provider interface validation
func TestProviderInterfaceCompliance(t *testing.T, providerName string, provider iface.ChatModel, config *Config)

// Method signature validation
func TestMethodSignatures(t *testing.T)

// Contract testing
func TestProviderContract(t *testing.T)
```

### Performance Benchmarks
```go
// Performance measurement
func BenchmarkProviderInterface(b *testing.B)
func BenchmarkAdvancedMockOperations(b *testing.B)
```

## Mock Implementations

### AdvancedMockChatModel

A comprehensive mock implementation with:

```go
mock := NewAdvancedMockChatModel("test-model",
    WithResponses("Response 1", "Response 2"),
    WithProviderName("test-provider"),
    WithStreamingDelay(10*time.Millisecond),
    WithNetworkDelay(true),
    WithError(customError),
    WithToolResults(map[string]interface{}{
        "calculator": "42",
    }),
)
```

**Features:**
- Configurable responses with cycling
- Realistic streaming simulation
- Tool call simulation
- Network delay simulation
- Error injection capabilities
- Thread-safe operation counting
- Health check simulation

### Integration Test Helper

Provides safe integration testing:

```go
helper := NewIntegrationTestHelper()
provider, err := helper.SetupAnthropicProvider("claude-3-haiku-20240307")
helper.TestProviderIntegration(t, provider, "anthropic")
```

**Features:**
- Environment variable configuration
- Automatic rate limiting
- Cost control (request limits)
- Provider setup helpers
- Cross-provider comparison

## Test Patterns

### Table-Driven Tests

```go
tests := []struct {
    name        string
    description string
    input       any
    expected    []schema.Message
    wantErr     bool
    errContains string
}{
    {"string_input", "Convert simple string", "Hello", expectedMessages, false, ""},
    {"nil_input", "Handle nil input", nil, nil, true, "invalid input type"},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        result, err := EnsureMessages(tt.input)
        // Assertions...
    })
}
```

### Concurrent Testing

```go
ConcurrentTestRunner(t, func(t *testing.T) {
    response, err := provider.Generate(ctx, messages)
    assert.NoError(t, err)
    assert.NotNil(t, response)
}, 10)
```

### Load Testing

```go
scenario := LoadTestScenario{
    Name:         "High Frequency Generate",
    Duration:     2 * time.Second,
    Concurrency:  5,
    RequestRate:  10,
    TestFunc: func(ctx context.Context) error {
        _, err := provider.Generate(ctx, messages)
        return err
    },
}
RunLoadTest(t, scenario)
```

### Interface Compliance Testing

```go
suite := NewProviderInterfaceTestSuite("anthropic", provider, config)
suite.WithVerbose(true).Run(t)
```

## Running Tests

### Unit Tests
```bash
# Run all unit tests
go test ./pkg/llms/

# Run with verbose output
go test -v ./pkg/llms/

# Run specific test
go test -run TestEnsureMessagesAdvanced ./pkg/llms/
```

### Integration Tests
```bash
# Set API keys
export ANTHROPIC_API_KEY=your_key
export OPENAI_API_KEY=your_key

# Run integration tests
go test -tags=integration ./pkg/llms/

# Run specific provider integration
go test -tags=integration -run TestAnthropicIntegration ./pkg/llms/

# Run with verbose output
go test -tags=integration -v ./pkg/llms/
```

### Benchmarks
```bash
# Run benchmarks
go test -bench=. ./pkg/llms/

# Run specific benchmark
go test -bench=BenchmarkProviderInterface ./pkg/llms/

# Run benchmarks with memory allocation info
go test -bench=. -benchmem ./pkg/llms/
```

### Concurrency Tests
```bash
# Run concurrency tests (may be slow)
go test -run TestConcurrency ./pkg/llms/

# Skip short tests to include concurrency
go test -run TestConcurrency -short=false ./pkg/llms/
```

## Configuration

### Environment Variables for Integration Tests

```bash
# API Keys
ANTHROPIC_API_KEY=your_anthropic_key
OPENAI_API_KEY=your_openai_key

# Test Settings
INTEGRATION_TEST_TIMEOUT=30s
SKIP_EXPENSIVE_TESTS=false
INTEGRATION_TEST_VERBOSE=true

# Provider Configuration
AWS_REGION=us-east-1
OLLAMA_BASE_URL=http://localhost:11434
```

### Test Configuration

The test suite automatically detects available providers and runs appropriate tests:

- **Anthropic**: Requires `ANTHROPIC_API_KEY`
- **OpenAI**: Requires `OPENAI_API_KEY`
- **AWS Bedrock**: Uses `AWS_REGION` (credentials from standard AWS sources)
- **Ollama**: Uses `OLLAMA_BASE_URL`

## Test Coverage Areas

### 1. Core Functionality
- Message processing and conversion
- Configuration validation
- Provider initialization
- Basic generate operations

### 2. Advanced Features
- Streaming responses
- Tool calling and binding
- Batch processing
- Error handling and retry logic

### 3. Observability
- Metrics collection
- Tracing integration
- Health checks
- Structured logging

### 4. Performance
- Concurrent operations
- Load testing
- Memory usage
- Response times

### 5. Reliability
- Thread safety
- Race condition prevention
- Resource cleanup
- Timeout handling

### 6. Integration
- Real API interactions
- Rate limiting
- Cost tracking
- Error recovery

## Best Practices

### Writing New Tests

1. **Use Table-Driven Tests**: For multiple test cases with similar structure
2. **Mock External Dependencies**: Use provided mock implementations
3. **Test Concurrently**: Include concurrency tests where appropriate
4. **Handle Timeouts**: Use context with timeouts for all operations
5. **Clean Up Resources**: Ensure proper cleanup in test teardown
6. **Test Error Cases**: Include comprehensive error scenario testing

### Mock Usage Example

```go
func TestMyFunction(t *testing.T) {
    // Create configurable mock
    mock := NewAdvancedMockChatModel("test-model",
        WithResponses("Expected response"),
        WithStreamingDelay(5*time.Millisecond),
        WithError(customError), // For error testing
    )

    // Use in tests
    result, err := mock.Generate(context.Background(), messages)

    // Assert expectations
    assert.NoError(t, err)
    assert.Equal(t, "Expected response", result.GetContent())

    // Verify interactions
    assert.Equal(t, 1, mock.GetCallCount())
}
```

### Integration Test Example

```go
func TestMyIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    helper := NewIntegrationTestHelper()
    provider, err := helper.SetupAnthropicProvider("claude-3-haiku-20240307")
    require.NoError(t, err)

    // Test your functionality
    result, err := myFunction(provider)
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Test Suite
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run Unit Tests
        run: go test ./pkg/llms/ -v

      - name: Run Benchmarks
        run: go test ./pkg/llms/ -bench=. -benchmem

  integration-test:
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Run Integration Tests
        env:
          ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        run: go test -tags=integration ./pkg/llms/ -v
```

## Migration Guide

### From Old Test Structure

**Before:**
```go
func TestBasic(t *testing.T) {
    // Manual test setup
    // Limited mock capabilities
    // No integration testing support
}
```

**After:**
```go
func TestBasicAdvanced(t *testing.T) {
    // Use table-driven tests
    tests := []struct{ /* test cases */ }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Comprehensive testing with helpers
        })
    }
}

func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration tests")
    }

    // Full integration testing with real providers
    helper := NewIntegrationTestHelper()
    // ... integration test logic
}
```

## Future Enhancements

### Planned Improvements

1. **Property-Based Testing**: Use libraries like `gopter` for property-based tests
2. **Fuzz Testing**: Add fuzz tests for input validation
3. **Performance Regression Testing**: Automated performance regression detection
4. **Chaos Testing**: Simulate network failures and service degradation
5. **Multi-Region Testing**: Test across different geographic regions
6. **Load Pattern Simulation**: More sophisticated load testing scenarios

### Additional Provider Support

The framework is designed to easily add new providers:

1. Implement the provider interface
2. Add integration test suite
3. Update environment configuration
4. Add to CI/CD pipeline

This comprehensive test suite provides a solid foundation for ensuring the reliability, performance, and correctness of LLM implementations while supporting both development and production deployment scenarios.
