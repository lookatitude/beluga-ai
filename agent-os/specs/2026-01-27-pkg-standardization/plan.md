# Full Package Standardization Plan

## Summary

Standardize the pkg/ organization across all 20 packages to achieve full consistency in:
- Naming conventions (keep plural, update docs)
- Registry pattern (registry.go at root, not registry/ subdirectory)
- Internal directory structure (optional, with clear guidelines)
- Documentation alignment (fix contradictions)

## Decisions Made

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Package naming | Keep plural (`agents`, `llms`) | Less breaking changes, matches existing code |
| Registry location | registry.go at root | Simpler pattern, matches majority of packages |
| internal/ directory | Optional | Only when genuinely needed |

## Tasks

### Task 1: Save Spec Documentation

Create `agent-os/specs/2026-01-27-pkg-standardization/` with:
- `plan.md` — This plan
- `shape.md` — Shaping decisions
- `standards.md` — Relevant standards
- `references.md` — Reference implementations

### Task 2: Update Documentation - Naming Conventions

Fix contradictions between docs and actual code:

**CLAUDE.md** (line 163):
```diff
- - **Packages**: lowercase, singular forms (`agent`, `llm`, not `agents`, `llms`)
+ - **Packages**: lowercase, plural forms (`agents`, `llms`, `embeddings`) for multi-provider packages
```

**CLAUDE.md** (line 43):
```diff
- (19 packages)
+ (20 packages)
```

**docs/package_design_patterns.md** (lines 5, 94, 290-292):
- Update package counts to 20
- Change "singular forms" to "plural forms"

**agent-os/standards/index.yml** (line 39):
```diff
- description: Package and folder naming — lowercase, singular
+ description: Package and folder naming — lowercase, plural forms preferred (agents, llms, embeddings)
```

### Task 3: Consolidate Registry Patterns

The embeddings package already has `GetRegistry()` in `factory.go`. No code changes needed for registry consolidation.

**Current state (already correct):**
- `chatmodels/registry.go` - wrapper exists
- `embeddings/factory.go` - has GetRegistry()
- `multimodal/registry.go` - wrapper exists

**Document the pattern:**
- Packages with import cycle issues use `registry/` subdirectory internally
- Root `registry.go` or `factory.go` provides `GetRegistry()` wrapper
- This pattern is acceptable and already consistent

### Task 4: Clarify internal/ Directory Guidelines

Update `docs/package_design_patterns.md` to clarify:

```markdown
### internal/ Directory (Optional)

The `internal/` directory is **optional** and should only be used when:
1. Complex base implementations that providers extend (e.g., `agents/internal/base`)
2. Shared utilities used by multiple providers (e.g., `llms/internal/common`)
3. Implementation details that should not be part of the public API

**Do NOT create empty internal/ directories.**

### Mock Location Standard

- **Package-only mocks**: Keep in `test_utils.go` at package root
- **Cross-package mocks**: May use `internal/mock/` if used by multiple packages
```

### Task 5: Consolidate Duplicate Mocks (Optional)

Low priority - packages with mocks in both `internal/mock/` and `test_utils.go`:
- `agents` - has both
- `embeddings` - has internal/mock/
- `chatmodels` - has internal/mock/
- `retrievers` - has internal/mock/

**Recommendation**: Leave as-is unless causing issues. The current patterns work.

### Task 6: Verify and Test

1. Run `make lint` - ensure no lint errors
2. Run `make test` - ensure all tests pass
3. Run `make build` - ensure build succeeds

## Files to Modify

### High Priority (Documentation)
| File | Change |
|------|--------|
| `CLAUDE.md` | Fix naming convention (line 163), package count (line 43) |
| `docs/package_design_patterns.md` | Fix naming convention, package counts, clarify internal/ |
| `agent-os/standards/index.yml` | Update naming standard description |

### No Code Changes Required
The codebase is already well-structured. The main gaps are documentation inconsistencies.

## Verification

After implementation:

```bash
# Verify no breaking changes
make lint
make test
make build

# Verify documentation consistency
grep -r "singular form" docs/ CLAUDE.md  # Should find 0 matches
grep -r "19 packages" docs/ CLAUDE.md    # Should find 0 matches
```

## Critical Files Reference

### Documentation to Update
- `/home/miguelp/Projects/lookatitude/beluga-ai/CLAUDE.md`
- `/home/miguelp/Projects/lookatitude/beluga-ai/docs/package_design_patterns.md`
- `/home/miguelp/Projects/lookatitude/beluga-ai/agent-os/standards/index.yml`

### Reference Implementations (for patterns)
- `/home/miguelp/Projects/lookatitude/beluga-ai/pkg/chatmodels/registry.go` - Registry wrapper pattern
- `/home/miguelp/Projects/lookatitude/beluga-ai/pkg/embeddings/factory.go` - GetRegistry() in factory
- `/home/miguelp/Projects/lookatitude/beluga-ai/pkg/llms/registry.go` - Standard root registry.go

## Summary

This is primarily a **documentation alignment** task. The codebase structure is already consistent - the documentation just needs to be updated to match reality:

1. **Naming**: Docs say "singular" but code uses "plural" → Update docs to say "plural"
2. **Package count**: Docs say "19" but actual is "20" → Update docs
3. **internal/**: Docs show it as required but it's optional → Clarify it's optional
4. **Registry pattern**: Both `registry.go` at root and `registry/` subdirectory are valid → Document both patterns

No breaking changes. No code refactoring. Just documentation fixes.
