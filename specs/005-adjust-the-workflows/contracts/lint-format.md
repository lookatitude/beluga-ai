# Contract: Linting and Formatting (Advisory with Auto-Fix)

**Feature**: 005-adjust-the-workflows  
**Date**: 2025-01-27  
**Contract ID**: C002

## Purpose
Ensure linting and formatting checks automatically fix issues where possible and provide warnings (not blocking) for remaining issues.

## Input
- Code changes in pull request
- GitHub Actions workflow trigger: `pull_request` or `push`

## Validation Rules

### Rule 1: Auto-Fix Must Be Enabled
- `golangci-lint` MUST run with `--fix` flag
- `gofmt` MUST run with write mode (`-w` flag) or equivalent
- Auto-fixed files SHOULD be uploaded as artifacts (optional, for review)

### Rule 2: Lint Job Must Be Advisory
- Lint job MUST have `continue-on-error: true`
- Lint failures MUST emit `::warning::` annotations (not `::error::`)
- Lint job failure MUST NOT block other jobs from running
- Lint job MUST NOT be in dependency chain of critical jobs

### Rule 3: Formatting Check
- `gofmt -l` MUST check for unformatted files
- Unformatted files MUST be listed in warning message
- Formatting issues MUST NOT cause job to exit with error code

### Rule 4: Package Declaration Validation
- All `.go` files MUST have package declarations
- Missing package declarations MUST emit error (but job continues)
- Error MUST be logged but not block pipeline

## Success Criteria
- Linting runs with auto-fix enabled
- Formatting issues are automatically fixed where possible
- Remaining issues generate warnings in PR checks
- Pipeline continues even if linting fails
- Fixed files are available for review (if artifacts uploaded)

## Failure Modes
- Lint job missing continue-on-error → Error: "Lint job must be advisory"
- golangci-lint missing --fix flag → Error: "Auto-fix must be enabled"
- Lint job blocking other jobs → Error: "Lint job must not block critical jobs"

## Test Validation
```bash
# Validate lint job configuration
grep -A 3 "name:.*Lint" .github/workflows/ci-cd.yml | grep -q "continue-on-error: true" || echo "ERROR: Lint job missing continue-on-error"

# Validate auto-fix flag
grep -q "golangci-lint.*--fix\|--fix.*golangci-lint" .github/workflows/ci-cd.yml || echo "ERROR: Auto-fix not enabled"

# Validate gofmt usage
grep -q "gofmt.*-w\|gofumpt.*-w" .github/workflows/ci-cd.yml || echo "WARNING: gofmt write mode not found"
```

## Implementation Notes
- Use `golangci/golangci-lint-action@v3` with `args: --fix`
- Run `gofmt -w .` or `gofumpt -l -w .` for formatting
- Consider uploading fixed files as artifacts for PR review
- Lint job should run early (can run in parallel with policy checks)

