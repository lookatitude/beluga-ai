# Package Design Patterns Refactor Plan

## Summary

Refactor the Beluga AI Framework package design patterns to formalize wrapper/aggregation packages, sub-package structure, and enforce mandatory requirements across all packages. This includes updating documentation, standards, and implementing code changes.

## Scope

- **Type**: Architectural refactor
- **Impact**: All 21 packages in `pkg/`
- **Source**: `temp_update_pkg_design.md`
- **Date**: 2026-01-26

## Objectives

1. Formalize wrapper/aggregation package patterns (voice, orchestration)
2. Standardize sub-package architecture across all packages
3. Enforce mandatory requirements (interfaces, OTEL, config, extensibility)
4. Add missing registry patterns to non-compliant packages
5. Bring safety/ package to full compliance

## Tasks

### Task 1: Save Spec Documentation
Create spec documentation in `agent-os/specs/2026-01-26-1500-package-design-refactor/`

### Task 2: Update Core Documentation
- Update `docs/package_design_patterns.md` with wrapper/aggregation patterns
- Update `docs/architecture.md` with wrapper package architecture

### Task 3: Create New Standards Files
- `agent-os/standards/global/wrapper-package-pattern.md`
- `agent-os/standards/global/subpackage-structure.md`
- `agent-os/standards/global/config-propagation.md`
- `agent-os/standards/backend/two-tier-factory.md`
- `agent-os/standards/backend/span-aggregation.md`
- `agent-os/standards/testing/subpackage-mocks.md`
- `agent-os/standards/testing/wrapper-integration.md`

### Task 4: Update Existing Standards
- `agent-os/standards/global/required-files.md`
- `agent-os/standards/global/provider-subpackage-layout.md`
- `agent-os/standards/backend/registry-shape.md`
- `agent-os/standards/index.yml`

### Task 5: Refactor safety/ Package
Create full compliance structure:
- `pkg/safety/iface/safety.go`
- `pkg/safety/config.go`
- `pkg/safety/metrics.go`
- `pkg/safety/test_utils.go`
- `pkg/safety/advanced_test.go`
- `pkg/safety/README.md`

### Task 6: Add Missing Registry Patterns
- `pkg/vectorstores/registry.go`
- `pkg/prompts/registry.go`
- `pkg/retrievers/registry.go`
- `pkg/orchestration/registry.go`
- `pkg/server/registry.go`
- `pkg/voice/registry.go` (facade)

### Task 7: Add Provider Auto-Registration
Add `init.go` files for auto-registration in providers

### Task 8: Voice Sub-Package Standardization
Verify/complete each voice sub-package structure

### Task 9: OTEL Audit
Verify all packages follow standard metrics pattern

### Task 10: Cleanup
Remove `temp_update_pkg_design.md`

## Implementation Order

1. Task 1 — Save spec documentation (preserves context)
2. Tasks 2-4 — Documentation and standards (defines patterns)
3. Task 5 — Safety package (most non-compliant)
4. Tasks 6-7 — Registry patterns (enables consistency)
5. Task 8 — Voice sub-packages (wrapper exemplar)
6. Task 9 — OTEL audit (quality verification)
7. Task 10 — Cleanup

## Verification

After implementation:

1. **Linting**: `make lint` passes
2. **Tests**: `make test` passes
3. **Coverage**: `make test-coverage` >= 80%
4. **Security**: `make security` passes
5. **Registry verification**: All providers auto-register via init()
6. **OTEL verification**: Metrics export correctly

## Critical Files

### Documentation
- `docs/package_design_patterns.md`
- `docs/architecture.md`
- `agent-os/standards/index.yml`

### Reference Implementations
- `pkg/llms/registry.go` — Gold standard registry
- `pkg/voice/stt/registry.go` — Exemplar sub-package registry
- `pkg/llms/providers/openai/init.go` — Exemplar auto-registration

### Packages Needing Changes
- `pkg/safety/` — Full restructure
- `pkg/vectorstores/` — Add registry
- `pkg/prompts/` — Add registry
- `pkg/retrievers/` — Add registry + restructure
- `pkg/orchestration/` — Add registry
- `pkg/server/` — Add registry
- `pkg/voice/` — Add facade registry
