# Package Standardization - Shaping Decisions

## Context

During a comprehensive review of the pkg/ directory structure, several inconsistencies were discovered between documentation and actual code patterns. This document captures the shaping decisions made to resolve these inconsistencies.

## Key Decisions

### 1. Package Naming: Plural vs Singular

**Problem**: Documentation (CLAUDE.md, package_design_patterns.md) specified singular naming (`agent`, `llm`), but all actual packages use plural (`agents`, `llms`, `embeddings`).

**Decision**: Keep plural naming, update documentation.

**Rationale**:
- All 20 packages already use plural forms
- Changing to singular would require massive refactoring
- Plural forms are equally valid in Go (no strict convention)
- Examples: `agents`, `llms`, `embeddings`, `vectorstores`, `retrievers`

### 2. Registry Location: Root vs Subdirectory

**Problem**: Some packages use `registry.go` at root, others have `registry/` subdirectory.

**Decision**: Prefer `registry.go` at root; `registry/` subdirectory acceptable when needed for import cycle avoidance.

**Rationale**:
- Majority of packages use root `registry.go` pattern
- Some packages (chatmodels, embeddings, multimodal) need `registry/` subdirectory to avoid import cycles
- These packages provide `GetRegistry()` wrapper at root for consistent API
- Both patterns achieve the same goal

**Current State**:
| Package | Pattern | Notes |
|---------|---------|-------|
| llms | root registry.go | Standard pattern |
| agents | root registry.go | Standard pattern |
| chatmodels | registry/ + wrapper | Import cycle avoidance |
| embeddings | registry/ + GetRegistry() in factory.go | Import cycle avoidance |
| multimodal | registry/ + wrapper | Import cycle avoidance |
| memory | root registry.go | Standard pattern |
| vectorstores | root registry.go | Standard pattern |

### 3. internal/ Directory: Required vs Optional

**Problem**: Documentation showed `internal/` as part of standard package structure, implying it's required.

**Decision**: Make `internal/` explicitly optional.

**Rationale**:
- Not all packages need internal implementation details
- Creating empty `internal/` directories adds noise
- Should only be used for:
  - Complex base implementations
  - Shared utilities across providers
  - Implementation details not for public API

### 4. Mock Location: internal/mock/ vs test_utils.go

**Problem**: Some packages have mocks in both locations.

**Decision**: Leave current structure; document preferred patterns.

**Preferred Patterns**:
- `test_utils.go` at package root for package-only mocks
- `internal/mock/` when mocks are used by multiple sub-packages
- Both patterns are acceptable

## Package Count Discrepancy

**Problem**: Documentation stated "19 packages" but actual count is 20.

**Actual packages** (confirmed with `ls -d pkg/*/ | wc -l`):
1. agents
2. chatmodels
3. config
4. core
5. documentloaders
6. embeddings
7. llms
8. memory
9. messaging
10. monitoring
11. multimodal
12. orchestration
13. prompts
14. retrievers
15. safety
16. schema
17. server
18. textsplitters
19. vectorstores
20. voice

**Decision**: Update all documentation to say "20 packages".

## Non-Changes

The following were considered but NOT changed:

1. **No code refactoring** - The codebase structure is correct; only docs need updates
2. **No mock consolidation** - Current dual-location pattern works fine
3. **No registry refactoring** - Current patterns work and are consistent

## Outcome

This standardization effort is documentation-only. The code is already well-structured; the documentation simply needed to be aligned with reality.
