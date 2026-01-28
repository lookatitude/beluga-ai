# Package Standardization Plan: pkg/retrievers and pkg/memory

## Summary

Refactor `pkg/retrievers` and `pkg/memory` to follow the standard package structure convention, aligning them with the patterns established in `pkg/embeddings` and `pkg/chatmodels`.

## Scope

| Package | Current Status | Work Required |
|---------|----------------|---------------|
| pkg/retrievers | NON-COMPLIANT | Add registry, reorganize providers |
| pkg/memory | MOSTLY COMPLIANT | Reorganize providers from internal/ to providers/ |

---

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-01-27-1800-package-standardization/` with:
- `plan.md` - This plan
- `shape.md` - Shaping notes and decisions
- `standards.md` - Applied standards (registry-shape, op-err-code, required-files)
- `references.md` - Reference implementations (pkg/embeddings, pkg/tools)

---

## Task 2: Standardize pkg/retrievers

### 2.1 Create Registry Infrastructure

**Create `pkg/retrievers/iface/registry.go`:**
- Define `RetrieverFactory` function type
- Define `Registry` interface with Register, Create, ListProviders, IsRegistered methods

**Create `pkg/retrievers/registry.go`:**
- Implement `ProviderRegistry` struct with sync.Once singleton pattern
- Add global convenience functions: `Register()`, `Create()`, `ListProviders()`
- Ensure interface compliance: `var _ iface.Registry = (*ProviderRegistry)(nil)`

### 2.2 Reorganize Providers

**Move `vectorstore.go` to `providers/vectorstore/`:**
- Create `providers/vectorstore/vectorstore.go` - Implementation
- Create `providers/vectorstore/config.go` - Provider config
- Create `providers/vectorstore/init.go` - Auto-registration via init()

**Move `multiquery.go` to `providers/multiquery/`:**
- Create `providers/multiquery/multiquery.go` - Implementation
- Create `providers/multiquery/config.go` - Provider config
- Create `providers/multiquery/init.go` - Auto-registration via init()

**Update `providers/mock/`:**
- Create `providers/mock/init.go` - Auto-registration

### 2.3 Update Package Files

**Update `errors.go`:**
- Add `ErrCodeProviderNotFound` constant

**Update `retrievers.go`:**
- Add `NewProvider(ctx, name, config)` convenience function

**Delete old files:**
- Delete `vectorstore.go` (moved to providers/)
- Delete `multiquery.go` (moved to providers/)

---

## Task 3: Standardize pkg/memory

### 3.1 Create Internal Base Package

**Create `internal/base/` for shared code:**
- `internal/base/history.go` - BaseChatMessageHistory (from providers/base_history.go)
- `internal/base/composite.go` - CompositeChatMessageHistory
- `internal/base/helpers.go` - Shared helper functions

### 3.2 Move Providers from internal/ to providers/

**Move buffer provider:**
- `internal/buffer/*.go` → `providers/buffer/`
- Create `providers/buffer/init.go`

**Move summary provider:**
- `internal/summary/*.go` → `providers/summary/`
- Create `providers/summary/init.go`

**Move vectorstore provider:**
- `internal/vectorstore/*.go` → `providers/vectorstore/`
- Create `providers/vectorstore/init.go`

**Move window provider:**
- `internal/window/*.go` → `providers/window/`
- Create `providers/window/init.go`

**Move redis provider:**
- `internal/redis/*.go` → `providers/redis/`
- Create `providers/redis/init.go`

### 3.3 Clean Up

**Delete old directories:**
- `internal/buffer/`
- `internal/summary/`
- `internal/vectorstore/`
- `internal/window/`
- `internal/redis/`

**Delete old files in providers/:**
- `providers/base_history.go` (moved to internal/base/)
- `providers/base_history_mock.go`
- `providers/base_history_test.go`

### 3.4 Update Imports

Update all files that import from old paths:
- `memory.go`
- `registry.go`
- `test_utils.go`
- `advanced_test.go`
- `memory_test.go`
- `memory_integration_test.go`

---

## Task 4: Verification

### Run Tests
```bash
make test-unit  # All unit tests
go test -v ./pkg/retrievers/...
go test -v ./pkg/memory/...
```

### Run Linter
```bash
make lint
```

### Verify Provider Registration
```go
// Test that providers auto-register
import (
    _ "github.com/lookatitude/beluga-ai/pkg/retrievers/providers/vectorstore"
    _ "github.com/lookatitude/beluga-ai/pkg/retrievers/providers/multiquery"
)

providers := retrievers.ListProviders()
// Should include: "vectorstore", "multiquery", "mock"
```

---

## Critical Files

### pkg/retrievers
- `pkg/retrievers/retrievers.go` - Main API
- `pkg/retrievers/vectorstore.go` - To be moved
- `pkg/retrievers/multiquery.go` - To be moved
- `pkg/retrievers/errors.go` - Add error code
- `pkg/retrievers/iface/interfaces.go` - Existing interfaces

### pkg/memory
- `pkg/memory/memory.go` - Main API, needs import updates
- `pkg/memory/registry.go` - Existing registry
- `pkg/memory/internal/buffer/buffer.go` - To be moved
- `pkg/memory/internal/summary/summary.go` - To be moved
- `pkg/memory/providers/base_history.go` - To be moved to internal/base/

### Reference Implementations
- `pkg/embeddings/registry.go` - Registry pattern reference
- `pkg/embeddings/iface/registry.go` - Interface reference
- `pkg/tools/registry.go` - Alternative registry reference
- `pkg/vectorstores/providers/inmemory/init.go` - init.go pattern reference

---

## Standards Applied

- **global/required-files** - Required files for multi-provider packages
- **backend/registry-shape** - Registry singleton with sync.Once
- **backend/op-err-code** - Error struct with Op/Err/Code fields
- **testing/test-utils** - Consolidated test utilities

---

## Backward Compatibility

- **Clean break** - No type aliases at old paths (v2.0.0-beta allows breaking changes)
- Keep existing public API unchanged (GetRegistry, convenience functions)
- Users must update imports to new provider paths
