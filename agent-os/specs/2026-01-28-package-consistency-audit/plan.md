# Package Structure Consistency Audit & Fix Plan

## Summary

Comprehensive audit of all 29 packages in `pkg/` to ensure consistency with the package design patterns. This builds upon the 2026-01-26 refactor and addresses remaining inconsistencies.

## Audit Results

### Fully Compliant Packages (24 packages)
- agents, audiotransport, documentloaders, embeddings, llms, memory, messaging, monitoring, multimodal, retrievers, s2s, server, stt, textsplitters, tools, tts, turndetection, vad, vectorstores, voicebackend, chatmodels, config, core, schema, voicesession, voiceutils

### Issues Found (3 packages)

| Package | Issue | Fix Required |
|---------|-------|--------------|
| `noisereduction` | Main file named `noise.go` instead of `noisereduction.go` | Rename file |
| `orchestration` | Main file named `orchestrator.go` instead of `orchestration.go` | Rename file |
| `prompts` | Missing `registry.go` with `providers/` directory | Add registry |

### Documented Deviations (No Changes Needed)

| Package | Deviation | Rationale |
|---------|-----------|-----------|
| `config` | No registry.go | Uses factory functions - config is loaded, not created from providers |
| `core` | No core.go main file | Utility package with multiple entry points (di.go, runnable.go) |
| `schema` | No registry | Single-purpose data structure package |
| `voicesession` | No providers/registry | Single implementation |
| `voiceutils` | No main API file | Shared utility interfaces package |

---

## Implementation Tasks

### Task 1: Save Spec Documentation

Create `agent-os/specs/2026-01-28-package-consistency-audit/` with:
- `plan.md` - This plan
- `shape.md` - Audit findings and decisions
- `standards.md` - Applicable standards (required-files, subpackage-structure)
- `references.md` - Reference to prior refactor spec

### Task 2: Rename noisereduction/noise.go

**Files to modify:**
- `pkg/noisereduction/noise.go` → `pkg/noisereduction/noisereduction.go`

**Steps:**
1. `git mv pkg/noisereduction/noise.go pkg/noisereduction/noisereduction.go`
2. Verify build: `go build ./pkg/noisereduction/...`
3. Verify tests: `go test ./pkg/noisereduction/...`

### Task 3: Rename orchestration/orchestrator.go

**Files to modify:**
- `pkg/orchestration/orchestrator.go` → `pkg/orchestration/orchestration.go`

**Steps:**
1. `git mv pkg/orchestration/orchestrator.go pkg/orchestration/orchestration.go`
2. Verify build: `go build ./pkg/orchestration/...`
3. Verify tests: `go test ./pkg/orchestration/...`

### Task 4: Add registry.go to prompts package

**Files to create:**
- `pkg/prompts/registry.go`

**Reference implementation:** `pkg/embeddings/registry.go`

**Registry structure:**
- TemplateFactory type alias
- TemplateRegistry struct with thread-safe operations
- Global registry instance with sync.Once initialization
- Register, Create, ListProviders, IsRegistered methods
- Package-level convenience functions

**Update mock provider to register:**
- Add `init()` function to `pkg/prompts/providers/mock.go` to register the mock

### Task 5: Update Documentation

**Files to update:**
- `docs/package_design_patterns.md` - Add "Intentional Deviations" section

**Deviations to document:**
1. **config**: Uses factory functions, not provider registry
2. **core**: Utility package with multiple entry points
3. **schema**: Data structure definitions only
4. **voiceutils**: Shared interfaces and utilities
5. **voicesession**: Single implementation, no multi-provider

---

## Verification

### After Each Task

```bash
# Build verification
make build

# Unit tests
make test-unit

# Lint check
make lint
```

### Final Verification

```bash
# Full CI pipeline
make ci-local

# Specific package tests
go test -v ./pkg/noisereduction/...
go test -v ./pkg/orchestration/...
go test -v ./pkg/prompts/...
```
