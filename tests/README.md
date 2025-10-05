# Integration Testing Framework

This directory contains comprehensive integration tests for the Beluga AI Framework, testing cross-package interactions and end-to-end workflows.

## Directory Structure

```
tests/
├── integration/            # Cross-package integration tests
│   ├── end_to_end/         # Complete workflow tests
│   ├── package_pairs/      # Two-package integration tests
│   ├── provider_compat/    # Provider interoperability tests  
│   ├── config_validation/  # Cross-package configuration tests
│   ├── observability/      # Monitoring and tracing tests
│   └── utils/             # Shared integration test utilities
├── fixtures/              # Test data and configurations
├── benchmarks/           # Performance benchmarks across packages
└── README.md            # This file
```

## Test Categories

### End-to-End Tests (`end_to_end/`)
Complete workflow tests that exercise multiple packages together:
- **RAG Pipeline**: LLMs + Memory + Vectorstores + Embeddings + Retrievers
- **Multi-Agent Workflows**: Agents + Orchestration + Memory + LLMs
- **Full Observability**: All packages + Monitoring + Server
- **Configuration Management**: Config + All provider packages
- **Schema Compatibility**: Schema + All packages using messages/documents

### Package Pair Tests (`package_pairs/`)
Two-package integration tests covering all important interactions:
- **LLMs Integration** (8 test suites)
- **Memory Integration** (5 test suites)  
- **Vectorstores Integration** (4 test suites)
- **Agents Integration** (3 test suites)
- **Embeddings Integration** (3 test suites)

### Provider Compatibility Tests (`provider_compat/`)
Tests ensuring different providers work consistently:
- Provider interface compliance
- Cross-provider switching
- Provider-specific configurations
- Performance comparisons

### Configuration Validation Tests (`config_validation/`)
Tests for configuration consistency and validation:
- Cross-package configuration validation
- Configuration inheritance and overrides
- Environment variable handling
- Validation error consistency

### Observability Tests (`observability/`)
Tests for monitoring, metrics, and tracing:
- Cross-package metrics collection
- Distributed tracing across components
- Health check consistency
- Performance monitoring

## Running Integration Tests

### Environment Setup

Set required environment variables for real provider testing:
```bash
export OPENAI_API_KEY="your-openai-api-key"
export ANTHROPIC_API_KEY="your-anthropic-api-key" 
# ... other provider API keys
```

### Running All Integration Tests
```bash
# Run all integration tests
go test ./tests/integration/... -v

# Run only fast tests (mocks)
go test ./tests/integration/... -v -short

# Run with race detection
go test ./tests/integration/... -v -race

# Run specific test suite
go test ./tests/integration/end_to_end/rag_pipeline_test.go -v
```

### Running Specific Categories
```bash
# End-to-end tests
go test ./tests/integration/end_to_end/... -v

# Package pair tests
go test ./tests/integration/package_pairs/... -v

# Provider compatibility tests
go test ./tests/integration/provider_compat/... -v

# Observability tests
go test ./tests/integration/observability/... -v
```

## Test Fixtures

The `fixtures/` directory contains:
- Sample configurations for different deployment scenarios
- Test datasets for embeddings and vector stores
- Mock provider configurations
- Performance baseline data

## Performance Benchmarks

The `benchmarks/` directory contains:
- Cross-package performance tests
- Resource usage benchmarks
- Scalability tests
- Memory usage analysis

## Writing Integration Tests

### Test Naming Convention
- File names: `<primary_package>_<secondary_package>_test.go` for pair tests
- Test function names: `TestIntegration<Package1><Package2><Scenario>`
- Benchmark function names: `BenchmarkIntegration<Package1><Package2>`

### Test Structure Template
```go
func TestIntegrationLLMsMemory(t *testing.T) {
    helper := utils.NewIntegrationTestHelper()
    
    // Setup components
    llm := helper.CreateMockLLM("test-llm")
    memory := helper.CreateMockMemory("test-memory")
    
    // Test integration scenarios
    err := helper.TestConversationFlow(llm, memory, 5)
    assert.NoError(t, err)
    
    // Verify cross-package interactions
    helper.AssertCrossPackageMetrics(t, "llms", "memory")
}
```

### Best Practices

1. **Use Mocks by Default**: Use real providers only when specifically testing provider integration
2. **Test Error Propagation**: Verify errors are properly propagated between packages
3. **Test Configuration**: Verify configurations work across package boundaries
4. **Test Observability**: Ensure metrics and tracing work end-to-end
5. **Test Resource Cleanup**: Verify proper cleanup of resources
6. **Test Concurrent Usage**: Verify thread safety across package interactions

## Contributing

When adding new packages or providers:

1. Add corresponding integration tests to `package_pairs/`
2. Update end-to-end tests to include new components
3. Add provider compatibility tests if adding new providers
4. Update configuration validation tests for new config options
5. Add observability tests for new metrics/traces

## Troubleshooting

### Common Issues

**Test Timeouts**: Increase timeout for slow operations or use mocks
**API Rate Limits**: Use mock providers for CI/CD pipelines
**Environment Dependencies**: Ensure all required services are available
**Flaky Tests**: Check for race conditions and add proper synchronization

### Debug Mode

Set `BELUGA_DEBUG=true` to enable detailed logging during integration tests.

### Test Data

Test fixtures are designed to be:
- **Deterministic**: Same inputs produce same outputs
- **Isolated**: Tests don't interfere with each other  
- **Realistic**: Representative of real-world usage
- **Lightweight**: Fast execution for CI/CD pipelines
