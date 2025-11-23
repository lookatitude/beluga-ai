# Research: Fix Corrupted Mock Files

## Overview
Research findings for fixing corrupted mock files missing package declarations in the beluga-ai package.

## Research Questions

### 1. What package names should be used for each mock file?

**Decision**: Use the package name matching the directory structure:
- `pkg/core/di_mock.go` → `package core`
- `pkg/prompts/advanced_mock.go` → `package prompts`
- `pkg/memory/advanced_mock.go` → `package memory`
- `pkg/vectorstores/advanced_mock.go` → `package vectorstores`
- `pkg/vectorstores/iface/iface_mock.go` → `package vectorstores` (not `package iface`)

**Rationale**: 
- Verified by checking existing files in each directory
- `pkg/core/traced_runnable.go` uses `package core`
- `pkg/prompts/prompts.go` uses `package prompts`
- `pkg/memory/memory.go` uses `package memory`
- `pkg/vectorstores/vectorstores.go` uses `package vectorstores`
- `pkg/vectorstores/iface/options.go` uses `package vectorstores` (subdirectories in Go don't create separate packages unless explicitly declared)

**Alternatives considered**: None - Go package naming is deterministic based on directory structure.

### 2. What imports are needed for the mock files?

**Decision**: Each mock file needs:
- `package <package_name>` declaration
- Import for `github.com/stretchr/testify/mock` (for `mock.Mock` type)

**Rationale**: 
- Mock files use `mock.Mock` embedded struct
- Existing mock files in the codebase follow this pattern
- No other imports needed as mocks are self-contained

**Alternatives considered**: None - standard Go mock pattern.

### 3. Should we validate the fix with go build?

**Decision**: Yes, validate with:
- `go build ./pkg/...` to ensure all packages compile
- `go test ./pkg/...` to ensure tests still pass
- `go mod verify` to ensure module integrity

**Rationale**:
- Ensures the fix resolves compilation errors
- Validates no regressions introduced
- Confirms module can be published successfully

**Alternatives considered**: Manual inspection only - rejected as insufficient for ensuring correctness.

### 4. Should we add validation to prevent future occurrences?

**Decision**: Yes, add a pre-commit or CI check to validate all `.go` files start with `package` declaration.

**Rationale**:
- Prevents regression
- Catches issues before publishing
- Low overhead validation

**Alternatives considered**: 
- Manual review only - rejected as error-prone
- Post-publish validation - rejected as too late

## Technical Decisions

### File Format
- Each mock file must start with: `package <name>`
- Followed by blank line
- Then imports (if needed)
- Then type definitions

### Validation Approach
- Use `go build` to validate syntax
- Use `go test` to validate functionality
- Use `grep` or simple script to validate package declarations exist

### Testing Strategy
- No new tests needed (fixing compilation errors)
- Existing tests should pass after fix
- Verify with `go test ./pkg/...`

## Dependencies
- Go 1.24.0 (already required)
- github.com/stretchr/testify/mock (already in use)

## Risks
- **Low risk**: Simple file correction
- **Compatibility**: No API changes, fully backward compatible
- **Testing**: Existing test suite validates behavior

## Next Steps
1. Add package declarations to 5 mock files
2. Validate with `go build`
3. Run existing tests
4. Consider adding CI validation for package declarations

