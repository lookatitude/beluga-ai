# Quickstart: Embeddings Package Corrections Validation

**Date**: October 5, 2025
**Purpose**: Quick validation guide to verify embeddings package corrections have been successfully implemented

## Prerequisites

- Go 1.21+ installed
- Access to OpenAI API key (optional, for full provider testing)
- Ollama server running locally (optional, for Ollama provider testing)
- Project checked out to `008-for-the-embeddings` branch

## Quick Validation Steps

### 1. Run Full Test Suite (5 minutes)
```bash
cd /home/swift/Projects/lookatitude/beluga-ai
go test ./pkg/embeddings/... -v -cover

# Expected: All tests pass, coverage >= 90%
# Verify: No failing tests in advanced_test.go
```

### 2. Check Ollama Dimension Handling (2 minutes)
```bash
cd /home/swift/Projects/lookatitude/beluga-ai
go test ./pkg/embeddings/providers/ollama/ -v -run TestOllamaEmbedder_GetDimension

# Expected: Test passes and shows dimension discovery logic
# Verify: GetDimension() attempts to query actual dimensions
```

### 3. Run Performance Benchmarks (3 minutes)
```bash
cd /home/swift/Projects/lookatitude/beluga-ai
go test ./pkg/embeddings/... -bench=. -benchmem

# Expected: All benchmarks complete successfully
# Verify: New load testing benchmarks are present and passing
```

### 4. Validate Framework Compliance (1 minute)
```bash
cd /home/swift/Projects/lookatitude/beluga-ai
# Check that all required files exist
ls -la pkg/embeddings/ | grep -E "(config\.go|metrics\.go|errors\.go|test_utils\.go|advanced_test\.go|README\.md)"

# Expected: All required files present
# Verify: Package structure matches framework standards
```

### 5. Test Global Registry (1 minute)
```bash
cd /home/swift/Projects/lookatitude/beluga-ai
go test ./pkg/embeddings/ -v -run TestProviderRegistry

# Expected: Registry tests pass for all providers
# Verify: Thread-safe registration and retrieval
```

## Correction Verification Checklist

### ✅ Ollama Dimension Handling
- [ ] GetDimension() returns actual dimensions instead of 0
- [ ] Dimension querying logic implemented
- [ ] Fallback to default dimensions when API unavailable

### ✅ Test Suite Reliability
- [ ] All advanced_test.go tests pass
- [ ] No failing rate limiting or mock behavior tests
- [ ] Test coverage maintained >= 90%

### ✅ Performance Testing Enhancement
- [ ] Load testing benchmarks added
- [ ] Concurrent user simulation implemented
- [ ] Sustained throughput testing available

### ✅ Documentation Enhancement
- [ ] README includes advanced configuration examples
- [ ] Troubleshooting section added
- [ ] Provider-specific setup guides present

### ✅ Integration Testing
- [ ] Cross-package integration tests added
- [ ] Vector store compatibility verified
- [ ] End-to-end embedding workflows tested

## Expected Outcomes

### Performance Baselines
- **Single embedding**: <100ms p95 latency
- **Batch processing**: 10-1000 documents supported
- **Memory usage**: <100MB per operation
- **Concurrent requests**: Thread-safe implementation
- **Load testing**: Sustains realistic user patterns

### Quality Metrics
- **Test coverage**: ≥ 90% across all components
- **Framework compliance**: 100% constitution adherence
- **Error handling**: Structured Op/Err/Code pattern
- **Observability**: Full OTEL integration
- **Documentation**: Comprehensive usage guides

### Provider Compatibility
- **OpenAI**: Full API integration with all models
- **Ollama**: Local model support with dimension discovery
- **Mock**: Deterministic testing with configurable dimensions

## Troubleshooting

### Common Issues

**Tests failing after corrections:**
```bash
# Run with verbose output to identify issues
go test ./pkg/embeddings/... -v 2>&1 | grep FAIL
```

**Performance benchmarks not running:**
```bash
# Ensure benchmarks are properly tagged
go test ./pkg/embeddings/... -list=. | grep Benchmark
```

**Ollama dimension discovery failing:**
```bash
# Check Ollama server connectivity
curl http://localhost:11434/api/tags
```

**Coverage below threshold:**
```bash
# Generate detailed coverage report
go test ./pkg/embeddings/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Getting Help

1. Check the [Embeddings Package README](pkg/embeddings/README.md)
2. Review [Framework Constitution](.specify/memory/constitution.md)
3. Run integration tests: `go test ./tests/integration/...`
4. Check GitHub issues for similar problems

## Success Criteria

**All corrections implemented successfully when:**
- ✅ Full test suite passes (0 failures)
- ✅ Test coverage ≥ 90%
- ✅ Ollama dimension discovery working
- ✅ Performance benchmarks complete successfully
- ✅ Framework compliance checklist fully checked
- ✅ Documentation examples functional
- ✅ Integration tests pass

**Time estimate**: 15-20 minutes for complete validation
