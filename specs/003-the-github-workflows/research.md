# Research: GitHub Workflows, Coverage, PR Checks, and Documentation

**Date**: 2025-01-27  
**Feature**: Fix GitHub Workflows, Coverage, PR Checks, and Documentation Generation

## Research Findings

### 1. Test Coverage Calculation Issue

**Problem**: Coverage shows 0% even when tests pass.

**Root Cause Analysis**:
- Coverage file (`coverage.unit.out`) may not be generated if tests fail early or coverage flags aren't properly passed
- The parsing logic in `ci-cd.yml` line 378 uses `go tool cover -func` which requires a valid coverage file
- If `coverage.unit.out` doesn't exist or is empty, the fallback is `echo "0%"` which results in 0% coverage
- The test command at line 346 generates coverage but may fail silently if tests have issues

**Decision**: Fix coverage generation to ensure file is always created, even with partial failures. Add validation to check if coverage file exists before parsing.

**Rationale**: Coverage should reflect actual test execution, not default to 0% on any error condition.

**Alternatives Considered**:
- Skip coverage on test failures - Rejected: Need coverage even for partial failures
- Use separate coverage job - Rejected: Adds complexity, current structure is fine
- Default to 100% on error - Rejected: Misleading, better to fail explicitly

### 2. Release Workflow Consolidation

**Problem**: Two separate release workflows (`release.yml` and `release_please.yml`) that could conflict.

**Current State**:
- `release.yml`: Triggered on version tags (`v*`) and manual dispatch, uses GoReleaser
- `release_please.yml`: Triggered on main branch pushes, uses release-please for semantic versioning

**Decision**: Merge both into a single unified workflow that:
- Supports automated semantic versioning via release-please on main branch
- Supports manual/tag-based releases via workflow_dispatch and tag triggers
- Uses GoReleaser for actual release creation
- Prevents conflicts through conditional job execution

**Rationale**: Single workflow reduces maintenance, eliminates conflicts, and provides both automated and manual release options.

**Alternatives Considered**:
- Keep both workflows with mutual exclusion - Rejected: Still confusing, requires coordination
- Remove one workflow entirely - Rejected: Need both automated and manual release capabilities
- Use separate workflows with different names - Rejected: Doesn't solve the confusion problem

### 3. PR Check Failure Handling

**Problem**: PR checks not passing correctly, unclear which checks are blocking vs advisory.

**Current State**:
- Multiple check jobs: lint, security, unit-tests, integration-tests, coverage, build
- All checks may fail but unclear which are critical vs advisory
- No distinction between blocking and warning checks

**Decision**: Implement check categorization:
- **Critical checks** (blocking): unit-tests, security scans, build
- **Advisory checks** (warnings): lint, coverage threshold (if below 80%)
- Use GitHub's check conclusion API to set appropriate status
- Allow merge with warnings but require explicit override for critical failures

**Rationale**: Provides flexibility for non-critical issues while maintaining quality gates for critical checks.

**Alternatives Considered**:
- All checks blocking - Rejected: Too strict, blocks on minor lint issues
- All checks advisory - Rejected: No quality enforcement
- Configurable per repository setting - Rejected: Adds complexity, hardcoded is sufficient

### 4. Documentation Generation

**Problem**: Documentation not being generated automatically.

**Current State**:
- `scripts/generate-docs.sh` exists and generates API docs using gomarkdoc
- `website_deploy.yml` workflow exists but may not be triggering correctly
- Documentation generation should run on main branch merges

**Decision**: Integrate documentation generation into `website_deploy.yml`:
- Trigger on merges to main branch (already configured)
- Run `make docs-generate` or `./scripts/generate-docs.sh` before building website
- Ensure gomarkdoc is installed in workflow
- Fail workflow if documentation generation fails

**Rationale**: Automatic generation ensures docs stay in sync with code changes.

**Alternatives Considered**:
- Generate on PRs for preview - Rejected: Spec requires only on main merges
- Manual trigger only - Rejected: Spec requires automatic generation
- Separate documentation workflow - Rejected: Consolidation preferred, website_deploy already exists

### 5. Coverage Threshold Enforcement

**Problem**: Need to enforce 80% coverage threshold.

**Current State**:
- Coverage threshold check exists in `ci-cd.yml` line 426 but may not be working correctly
- Threshold is 80% as per clarifications

**Decision**: Ensure coverage threshold check:
- Runs after coverage calculation
- Fails workflow if coverage < 80%
- Provides clear error message with actual vs required coverage
- Distinguishes between "no coverage file" (error) and "low coverage" (warning if advisory)

**Rationale**: Enforces quality standard while allowing override for critical checks.

**Alternatives Considered**:
- No threshold enforcement - Rejected: Spec requires 80% minimum
- Configurable threshold - Rejected: 80% is specified in clarifications
- Warning only - Rejected: Need enforcement, but can be advisory check

### 6. Workflow File Organization

**Decision**: Maintain current structure with improvements:
- Keep `ci-cd.yml` as main CI workflow
- Consolidate `release.yml` and `release_please.yml` into single `release.yml`
- Keep `website_deploy.yml` separate (different purpose)
- Add clear comments explaining workflow triggers and purposes

**Rationale**: Current structure is logical, just needs consolidation and fixes.

**Alternatives Considered**:
- Single monolithic workflow - Rejected: Too large, hard to maintain
- More granular workflows - Rejected: Current granularity is appropriate

## Technical Decisions

1. **Coverage File Handling**: Always generate coverage file, validate existence before parsing, provide meaningful errors
2. **Release Workflow**: Single workflow with conditional execution based on trigger type
3. **Check Status**: Use GitHub Actions `conclusion` field to distinguish success/warning/failure
4. **Documentation**: Integrate into existing website_deploy workflow, fail on generation errors
5. **Error Reporting**: Clear error messages in workflow summaries and step outputs

## Dependencies

- GitHub Actions (platform)
- Go toolchain (for coverage calculation)
- gomarkdoc (for API documentation generation)
- GoReleaser (for release creation)
- release-please-action (for semantic versioning)
- Docusaurus (for website deployment)

## Open Questions

None - all clarifications resolved in specification phase.

