# Test Issues Summary

This document provides a quick reference for known test issues and their status.

## Current Test Status

### ✅ Passing Tests
- Most package tests pass successfully
- Integration tests pass
- `pkg/config/iface` - Fixed (was failing due to incorrect test expectation)

### ❌ Known Failures

#### 1. Import Cycle Issues

**Packages Affected:**
- `pkg/chatmodels` - All tests fail
- `pkg/embeddings` - All tests fail

**Error:**
```
import cycle not allowed in test
```

**Root Cause:**
Provider packages import the main package to register themselves, creating a cycle when test files try to import providers.

**Status:** Documented, requires architectural refactoring

**Documentation:** See [test-import-cycles.md](./test-import-cycles.md)

**Workaround:** None currently. Tests will fail until registry pattern is refactored.

#### 2. S2S Provider Tests Making Real API Calls

**Packages Affected:**
- `pkg/voice/s2s/providers/amazon_nova`
- `pkg/voice/s2s/providers/gemini`
- `pkg/voice/s2s/providers/grok`
- `pkg/voice/s2s/providers/openai_realtime`

**Error:**
```
s2s Process: [Provider] API request failed (code: invalid_request)
```

**Root Cause:**
Tests are making actual API calls instead of using mocked HTTP clients.

**Status:** Needs refactoring to use mocks or mark as integration tests

**Recommended Fix:**
1. Use HTTP client mocking (e.g., `httptest` or `gock`)
2. Or mark tests with `//go:build integration` tag
3. Or skip tests when API keys are not available

## Linter Issues

**Total Issues:** 5,839 (mostly style/naming)

**Common Issues:**
- `tagliatelle`: YAML/JSON tag naming conventions (303 issues)
- `thelper`: Test helper functions should call `t.Helper()` (89 issues)
- `unused`: Unused code (79 issues)
- `wrapcheck`: Error wrapping (31 issues)
- `revive`: Various style issues (1,677 issues)
- Others: formatting, naming, etc.

**Status:** Non-blocking, but should be addressed incrementally

**Note:** These are mostly style issues and don't prevent code from running.

## Quick Reference

### Running Tests

```bash
# Run all tests (will show failures)
make test

# Run only unit tests
make test-unit

# Run only integration tests
make test-integration

# Run tests for specific package
go test ./pkg/chatmodels
```

### Running Linter

```bash
# Run linter
make lint

# Auto-fix lint issues
make lint-fix
```

### Skipping Failing Tests

To skip tests that require provider registration:

```bash
# Skip chatmodels tests
go test ./pkg/... -skip pkg/chatmodels

# Skip embeddings tests
go test ./pkg/... -skip pkg/embeddings
```

## Next Steps

1. **High Priority:** Refactor registry pattern to eliminate import cycles
   - See [test-import-cycles.md](./test-import-cycles.md) for proposed solutions
   - Recommended: Solution 1 (Separate Registry Interface)

2. **Medium Priority:** Fix S2S provider tests
   - Add HTTP client mocking
   - Or mark as integration tests with build tags

3. **Low Priority:** Address linter issues incrementally
   - Focus on critical issues first (unused code, error handling)
   - Style issues can be addressed over time

## Related Documentation

- [Test Import Cycles](./test-import-cycles.md) - Detailed analysis and solutions
- [Package Design Patterns](../package_design_patterns.md) - Framework patterns
- [Architecture Overview](./architecture.md) - Overall architecture
