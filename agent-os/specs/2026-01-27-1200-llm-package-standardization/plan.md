# LLM/Model Package Standardization Plan

## Summary

Refactor `pkg/llms`, `pkg/chatmodels`, and `pkg/embeddings` to follow the standard package structure convention, using `pkg/tools` as the reference implementation.

## Current State

| Package | Compliance | Key Issues |
|---------|------------|------------|
| pkg/llms | Excellent | No changes needed |
| pkg/chatmodels | Good | Registry impl in iface/, scattered mock file |
| pkg/embeddings | Most deviations | Duplicate errors.go, fragmented registry, scattered mocks |

## Tasks

### Task 1: Save Spec Documentation
Create spec documentation in `agent-os/specs/2026-01-27-1200-llm-package-standardization/`.

### Task 2: Standardize pkg/embeddings (Highest Priority)

**2.1 Consolidate Error Types**
- Keep `errors.go` at root with Op/Err/Code pattern
- Migrate unique codes from `iface/errors.go`
- Delete `iface/errors.go`

**2.2 Consolidate Registry**
- Move impl from `internal/registry/registry.go` to root `registry.go`
- Add `sync.Once` pattern (following pkg/tools)
- Delete `internal/registry/` and `factory.go`
- Keep only interface in `iface/registry.go`

**2.3 Consolidate Test Files**
- Merge `embeddings_mock.go`, `advanced_mock.go` into `test_utils.go`
- Merge `testutils/helpers.go` into `test_utils.go`
- Move `iface/iface_test.go` to root
- Delete redundant files/directories

**2.4 Update Provider Registrations**
- Update all 6 provider `init.go` files to use root registry

### Task 3: Standardize pkg/chatmodels (Medium Priority)

**3.1 Move Registry Implementation**
- Move impl from `iface/registry.go` to root `registry.go`
- Keep only `Registry` interface + `ProviderFactory` type in `iface/registry.go`
- Add `sync.Once` pattern

**3.2 Consolidate Test Files**
- Merge `advanced_mock.go` into `test_utils.go`
- Delete `advanced_mock.go`

### Task 4: pkg/llms (No Changes)
Package already follows standard structure. No changes required.

### Task 5: Verification
- Run `make test-unit` for all three packages
- Run `make lint` to check for style issues
- Verify provider registration works correctly

## Timeline

1. Spec documentation - immediate
2. pkg/embeddings standardization - primary work
3. pkg/chatmodels standardization - secondary work
4. Verification - final step

## Backward Compatibility

- Type aliases at old locations where types are moved
- No public API changes (`GetRegistry()` remains entry point)
- Deprecated comments on old locations
