# Research: Comprehensive Test Coverage Improvement

**Feature**: 001-comprehensive-test-coverage  
**Date**: 2026-01-16  
**Status**: Complete

## Research Questions & Findings

### 1. Coverage Analysis Tools and Best Practices

**Question**: What tools and approaches are best for achieving 100% test coverage in Go projects?

**Decision**: Use Go's built-in `go tool cover` with coverage profiles, supplemented by existing Makefile targets (`make test-coverage`)

**Rationale**:
- Go's native coverage tooling is well-integrated and provides accurate metrics
- Existing Makefile already has coverage targets configured
- HTML and machine-readable (JSON) formats are supported natively
- No additional dependencies required

**Alternatives Considered**:
- Third-party coverage tools (gocov, goveralls) - Rejected: adds dependencies, native tools sufficient
- Coverage thresholds per package - Rejected: spec requires 100% for all testable paths

**Implementation Notes**:
- Use `go test -coverprofile=coverage.out -covermode=atomic` for accurate coverage
- Generate HTML reports with `go tool cover -html=coverage.out`
- Parse coverage.out for machine-readable JSON format
- Maintain exclusion lists in package-level documentation

### 2. Mock Implementation Patterns

**Question**: How should mocks be structured to support all common error scenarios while maintaining consistency?

**Decision**: Follow existing AdvancedMock pattern with functional options for error simulation

**Rationale**:
- Framework already has established AdvancedMock pattern (see `pkg/llms/test_utils.go`)
- Pattern supports configurable behavior via functional options
- Maintains consistency across all packages
- Supports both success and error scenarios

**Alternatives Considered**:
- Using third-party mock generators (mockery, gomock) - Rejected: doesn't match existing patterns, adds dependencies
- Interface-based mocks only - Rejected: AdvancedMock pattern provides more flexibility

**Implementation Notes**:
- All mocks must implement interface methods with configurable behavior
- Support error simulation via `WithError()` or similar options
- Support network delay simulation via `WithNetworkDelay()`
- Support health state simulation via `WithHealthState()`
- Document all supported error types in mock implementations

### 3. Integration Test Scope Definition

**Question**: How to systematically identify all direct package dependencies for integration testing?

**Decision**: Use static analysis of Go imports to identify direct dependencies, then create integration tests for each pair

**Rationale**:
- Direct dependencies are clearly defined by import statements
- Systematic approach ensures no combinations are missed
- Existing `tests/integration/package_pairs/` structure supports this
- Matches spec requirement for "all direct package dependencies"

**Alternatives Considered**:
- Manual identification - Rejected: error-prone, may miss dependencies
- Testing all possible combinations - Rejected: too broad, doesn't match spec requirement

**Implementation Notes**:
- Use `go list -f '{{.Imports}}'` to identify direct imports
- Create integration test files for each direct dependency pair
- Focus on realistic usage scenarios, not exhaustive combinations
- Maintain test organization in `tests/integration/package_pairs/`

### 4. Handling Untestable Code Paths

**Question**: How to document and manage exclusions for genuinely untestable code paths?

**Decision**: Maintain exclusion lists in package-level documentation with justification

**Rationale**:
- Provides transparency and accountability
- Allows review of exclusions over time
- Maintains 100% coverage goal while acknowledging practical limits
- Matches spec clarification requirement

**Alternatives Considered**:
- Coverage tool exclusions (//nolint comments) - Rejected: doesn't provide justification context
- Separate exclusion file - Rejected: harder to maintain, less discoverable

**Implementation Notes**:
- Document exclusions in package README.md or test_utils.go
- Include: file path, function/method, reason for exclusion, date
- Review exclusions periodically to see if testing approach can be improved
- Example format:
  ```go
  // EXCLUSION: pkg/example/example.go:handlePanic()
  // Reason: Panic handler cannot be tested without causing actual panic
  // Date: 2026-01-16
  ```

### 5. Coverage Report Formats

**Question**: How to provide both HTML and machine-readable coverage reports?

**Decision**: Use Go's native coverage tooling for HTML, parse coverage.out for JSON/XML

**Rationale**:
- Go's `go tool cover -html` provides HTML reports natively
- `coverage.out` format is well-documented and parseable
- Can generate JSON/XML from coverage.out parsing
- No additional tooling required

**Alternatives Considered**:
- Third-party report generators - Rejected: adds dependencies, native tools sufficient
- Custom report format - Rejected: standard formats (JSON/XML) are more useful

**Implementation Notes**:
- HTML: `go tool cover -html=coverage.out -o coverage.html`
- JSON/XML: Parse coverage.out format (block-based coverage data)
- Include package-level and file-level coverage metrics
- Identify uncovered code paths with line numbers

### 6. Test Execution Performance

**Question**: How to ensure test suite completes in under 10 minutes?

**Decision**: Optimize test execution through parallelization, test organization, and mock usage

**Rationale**:
- Go's test runner supports parallel execution (`-parallel` flag)
- Mocks eliminate network latency from unit tests
- Integration tests can run in parallel where independent
- Existing Makefile already uses race detection which may slow tests

**Alternatives Considered**:
- Separate fast/slow test suites - Rejected: adds complexity, spec requires single suite
- Disable race detection - Rejected: race detection is required for quality

**Implementation Notes**:
- Use `t.Parallel()` in tests where safe
- Ensure mocks are fast (no actual network calls)
- Organize integration tests to minimize setup/teardown overhead
- Monitor test execution time and optimize slow tests

## Technology Decisions

### Testing Framework
- **Decision**: Continue using Go standard testing + testify
- **Rationale**: Already established, well-understood, no migration needed

### Coverage Tooling
- **Decision**: Go native `go tool cover`
- **Rationale**: Built-in, accurate, supports required formats

### Mock Pattern
- **Decision**: AdvancedMock pattern with functional options
- **Rationale**: Matches existing framework patterns, provides flexibility

### Integration Test Organization
- **Decision**: Maintain existing `tests/integration/` structure
- **Rationale**: Already established, supports required test types

## Open Questions Resolved

1. ✅ **Untestable code paths** → Document with justification in package
2. ✅ **Mock error scenarios** → Support all common error types via AdvancedMock options
3. ✅ **Integration test scope** → All direct package dependencies
4. ✅ **Unmockable dependencies** → Document as exclusions with justification
5. ✅ **Coverage report formats** → HTML + machine-readable (JSON/XML)

## References

- Go Testing Documentation: https://pkg.go.dev/testing
- Go Coverage Tool: https://go.dev/blog/cover
- Existing AdvancedMock Pattern: `pkg/llms/test_utils.go`
- Framework Testing Standards: `.cursor/rules/beluga-test-standards.mdc`
- Framework Design Patterns: `.cursor/rules/beluga-design-patterns.mdc`
