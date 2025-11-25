# Contract: Test Coverage with Threshold Warnings

**Feature**: 005-adjust-the-workflows  
**Date**: 2025-01-27  
**Contract ID**: C003

## Purpose
Ensure unit tests and integration tests must pass, but coverage below 80% generates warnings without blocking the pipeline.

## Input
- Test execution results
- Coverage profile files (`coverage.unit.out`, `coverage.integration.out`)

## Validation Rules

### Rule 1: Tests Must Pass (Critical)
- Unit test job MUST NOT have `continue-on-error: true`
- Integration test job MUST NOT have `continue-on-error: true`
- Test failures MUST cause pipeline to fail and block merging
- Test job failures MUST emit `::error::` annotations

### Rule 2: Coverage Calculation
- Coverage file MUST be generated after test execution
- Coverage file MUST be valid Go coverage format (parseable by `go tool cover`)
- Coverage percentage MUST be calculated from coverage file
- Coverage percentage MUST be between 0 and 100

### Rule 3: Coverage Threshold Warning (Advisory)
- If coverage < 80%, MUST emit warning: `::warning::Coverage X% is below 80% threshold`
- Coverage warning MUST NOT cause job to exit with error code
- Coverage warning MUST NOT block pipeline continuation
- Coverage MUST be displayed in PR checks summary

### Rule 4: Separate Coverage for Unit and Integration Tests
- Unit test coverage MUST be calculated separately
- Integration test coverage MUST be calculated separately
- Both coverage percentages MUST be checked against 80% threshold
- Both MUST generate warnings if below threshold (but not block)

## Success Criteria
- Tests must pass for pipeline to succeed
- Coverage is calculated and reported accurately
- Coverage below 80% shows warning but doesn't block
- Coverage percentages are displayed in PR checks
- Coverage artifacts are uploaded for analysis

## Failure Modes
- Test job has continue-on-error → Error: "Test jobs must be critical"
- Coverage file missing → Error: "Coverage file not generated"
- Coverage file invalid → Error: "Coverage file format invalid"
- Coverage calculation fails → Error: "Failed to calculate coverage"
- Coverage threshold check blocks pipeline → Error: "Coverage threshold must be advisory"

## Test Validation
```bash
# Validate test jobs are critical
grep -A 3 "name:.*Unit Tests\|name:.*Integration Tests" .github/workflows/ci-cd.yml | grep -q "continue-on-error: true" && echo "ERROR: Test jobs must not have continue-on-error"

# Validate coverage calculation
grep -q "go tool cover -func" .github/workflows/ci-cd.yml || echo "ERROR: Coverage calculation not found"

# Validate coverage threshold warning (not error)
grep -A 5 "coverage.*80" .github/workflows/ci-cd.yml | grep -q "::warning::" || echo "ERROR: Coverage threshold must use warning, not error"

# Validate coverage threshold doesn't exit with error
grep -A 10 "coverage.*80" .github/workflows/ci-cd.yml | grep -q "exit 1" && echo "ERROR: Coverage threshold must not exit with error"
```

## Implementation Notes
- Use `go test -coverprofile=coverage.unit.out` for unit tests
- Use `go test -coverprofile=coverage.integration.out` for integration tests
- Calculate coverage: `go tool cover -func=coverage.unit.out | tail -n1 | awk '{print $3}'`
- Compare to 80%: `if awk "BEGIN {exit !($pct < 80)}"; then echo "::warning::..."; fi`
- Do NOT exit with error code for coverage threshold violations
- Upload coverage files as artifacts

