# Import Cycle Audit Report

This document provides a comprehensive audit of all packages for potential import cycle issues, similar to the ones fixed in `pkg/chatmodels` and `pkg/embeddings`.

## Summary

- **Fixed Packages:** `pkg/chatmodels`, `pkg/embeddings` ✅
- **At-Risk Packages:** `pkg/llms`, `pkg/vectorstores`, voice packages (stt, tts, vad, etc.)
- **Safe Packages:** All other packages (agents, core, memory, monitoring, orchestration, prompts, schema, server, voice/session) ✅
- **Status:** No active cycles detected across all packages. Potential for future cycles in at-risk packages if test files import providers.

## Detailed Findings

### ✅ Fixed Packages

#### `pkg/chatmodels`
- **Status:** Fixed ✅
- **Solution:** Separate registry interface pattern
- **Details:** Providers now use `pkg/chatmodels/registry` instead of importing main package
- **Test Status:** Tests pass, can import providers safely

#### `pkg/embeddings`
- **Status:** Fixed ✅
- **Solution:** Separate registry interface pattern with reflection
- **Details:** Providers use reflection to access config without importing main package
- **Test Status:** Tests pass, can import providers safely

### ⚠️ At-Risk Packages

These packages have providers that import the main package directly, but their test files don't currently import providers. If test files are updated to import providers in the future, they would hit the same import cycle issue.

#### `pkg/llms`
- **Provider Pattern:** Providers import `github.com/lookatitude/beluga-ai/pkg/llms` directly
- **Example:** `pkg/llms/providers/mock/init.go` imports `pkg/llms`
- **Test Status:** ✅ No cycle (test files don't import providers)
- **Risk Level:** Medium - Tests don't import providers, but could in future
- **Recommendation:** Consider applying the same registry interface pattern proactively

#### `pkg/vectorstores`
- **Provider Pattern:** Providers import `github.com/lookatitude/beluga-ai/pkg/vectorstores` directly
- **Example:** `pkg/vectorstores/providers/pinecone/init.go` imports `pkg/vectorstores`
- **Test Status:** ✅ No cycle (test files don't import providers)
- **Risk Level:** Medium - Tests don't import providers, but could in future
- **Recommendation:** Consider applying the same registry interface pattern proactively

#### Voice Packages (`pkg/voice/stt`, `pkg/voice/tts`, `pkg/voice/vad`, etc.)
- **Provider Pattern:** Providers import their respective main packages directly
- **Test Status:** ✅ No cycle (test files don't import providers)
- **Risk Level:** Low - Voice packages are less likely to need provider imports in tests
- **Recommendation:** Monitor, but lower priority

### ✅ Safe Packages

These packages either don't have providers or use a different pattern that doesn't create cycles:

#### Core Packages
- `pkg/agents` - ✅ Safe - Registry pattern, but providers don't use `init()` to auto-register. Built-in types registered in main package's `init()`. Provider packages are just implementations, not auto-registered.
- `pkg/config` - ✅ Safe - No provider pattern
- `pkg/core` - ✅ Safe - No provider pattern
- `pkg/memory` - ✅ Safe - Internal factory pattern, no external provider registration
- `pkg/monitoring` - ✅ Safe - Provider registry exists but uses manual registration, not `init()` auto-registration
- `pkg/orchestration` - ✅ Safe - Providers are internal implementations, not auto-registered via `init()`
- `pkg/prompts` - ✅ Safe - No provider pattern
- `pkg/retrievers` - ✅ Safe - No provider pattern
- `pkg/schema` - ✅ Safe - No provider pattern
- `pkg/server` - ✅ Safe - No provider pattern

#### Voice Packages (All Safe)
- `pkg/voice/stt` - ✅ Safe - Providers import main package, but test files don't import providers
- `pkg/voice/tts` - ✅ Safe - Providers import main package, but test files don't import providers
- `pkg/voice/vad` - ✅ Safe - Providers import main package, but test files don't import providers
- `pkg/voice/s2s` - ✅ Safe - Providers import main package, but test files don't import providers
- `pkg/voice/transport` - ✅ Safe - Providers import main package, but test files don't import providers
- `pkg/voice/turndetection` - ✅ Safe - Providers import main package, but test files don't import providers
- `pkg/voice/noise` - ✅ Safe - Providers import main package, but test files don't import providers
- `pkg/voice/session` - ✅ Safe - No provider pattern

## Pattern Analysis

### Current Pattern (At-Risk)

```go
// pkg/llms/providers/mock/init.go
package mock

import "github.com/lookatitude/beluga-ai/pkg/llms"

func init() {
    llms.GetRegistry().Register("mock", NewMockProviderFactory())
}
```

**Problem:** If a test file imports this provider, it creates a cycle:
```
test_file → provider → main_package → (cycle detected)
```

### Fixed Pattern (Recommended)

```go
// pkg/chatmodels/providers/mock/init.go
package mock

import (
    "github.com/lookatitude/beluga-ai/pkg/chatmodels/iface"
    "github.com/lookatitude/beluga-ai/pkg/chatmodels/registry"
)

func init() {
    registry.GetRegistry().Register("mock", func(model string, config any, options *iface.Options) (iface.ChatModel, error) {
        return NewMockChatModel(model, config, options)
    })
}
```

**Solution:** Providers import `registry` package instead of main package, breaking the cycle.

## Recommendations

### Immediate Actions
1. ✅ **Completed:** Fixed `pkg/chatmodels` and `pkg/embeddings`
2. ✅ **Completed:** Documented the pattern for future reference

### Future Considerations

1. **Proactive Fixes (Optional):**
   - Apply the registry interface pattern to `pkg/llms` and `pkg/vectorstores` to prevent future issues
   - This would make the codebase more consistent and prevent accidental cycles

2. **Testing Guidelines:**
   - Document that test files should avoid importing providers directly when providers import the main package
   - Or, ensure all packages use the registry interface pattern

3. **Code Review:**
   - When adding new provider packages, ensure they use the registry interface pattern
   - When updating test files, check for potential import cycles

## Testing

To verify no import cycles exist:

```bash
# Test all packages
go test -c ./pkg/... 2>&1 | grep -E "(import cycle|cycle not allowed)"

# Test specific package with provider imports
go test -c ./pkg/llms 2>&1 | grep -E "(import cycle|cycle not allowed)"
```

## Related Documentation

- [test-import-cycles.md](./test-import-cycles.md) - Detailed explanation of the issue and solutions
- TEST_ISSUES_SUMMARY.md - Summary of test issues
