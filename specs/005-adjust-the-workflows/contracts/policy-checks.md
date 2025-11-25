# Contract: Policy Checks (Advisory)

**Feature**: 005-adjust-the-workflows  
**Date**: 2025-01-27  
**Contract ID**: C001

## Purpose
Ensure policy checks (branch naming, PR description, PR size) provide warnings but do not block the release or merge process.

## Input
- Pull request event data (branch name, PR description, PR size)
- GitHub Actions workflow trigger: `pull_request`

## Validation Rules

### Rule 1: Policy Checks Must Be Advisory
- Policy job MUST have `continue-on-error: true`
- Policy check failures MUST emit `::warning::` annotations (not `::error::`)
- Policy job failure MUST NOT block other jobs from running
- Policy job MUST NOT be in the dependency chain of critical jobs

### Rule 2: Branch Name Validation
- Branch name MUST match pattern: `^[0-9]{3}-[a-z0-9-]+$` (###-kebab-case)
- Invalid branch names MUST emit warning: `::warning::Branch name must be ###-kebab-case`
- Warning MUST NOT cause job to exit with error code

### Rule 3: PR Description Validation
- PR description MUST contain spec reference (Spec:, /specs/, Feature Spec, Mini-spec)
- Missing spec reference MUST emit warning: `::warning::PR body must link a spec or include mini-spec`
- Warning MUST NOT cause job to exit with error code

### Rule 4: PR Size Validation
- PRs with >1500 lines MUST emit error (but job continues due to continue-on-error)
- PRs with >600 lines MUST emit warning: `::warning::Large PR (>600 lines). Consider splitting`
- PR size warnings MUST NOT block merge

## Success Criteria
- Policy checks run on all pull requests
- Warnings are displayed in PR checks UI
- Pipeline continues even if policy checks fail
- No blocking behavior from policy checks

## Failure Modes
- Policy job missing continue-on-error → Error: "Policy job must be advisory"
- Policy checks using ::error:: → Error: "Policy checks must use warnings"
- Policy job blocking other jobs → Error: "Policy job must not block critical jobs"

## Test Validation
```bash
# Validate policy job configuration
grep -A 3 "name:.*Policy" .github/workflows/ci-cd.yml | grep -q "continue-on-error: true" || echo "ERROR: Policy job missing continue-on-error"

# Validate warning annotations
grep -q "::warning::" .github/workflows/ci-cd.yml || echo "ERROR: Policy checks not using warnings"

# Validate job is not blocking
# Policy job should not be in 'needs:' of any critical job
```

## Implementation Notes
- Policy job should run early in pipeline (no dependencies)
- Policy checks provide feedback but allow flexibility for edge cases
- Large PRs can proceed with 'large-pr' label (if implemented)

