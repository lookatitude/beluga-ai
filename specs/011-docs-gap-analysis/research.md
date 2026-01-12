# Research: Documentation Gap Analysis and Resource Creation

**Feature**: Documentation Gap Analysis and Resource Creation  
**Date**: 2025-01-27  
**Phase**: 0 - Research

## Research Questions Identified

### 1. Documentation Synchronization Mechanism

**Question**: How should markdown files in root `docs/` directory be synchronized with `website/docs/` for Docusaurus?

**Context**: Clarification states "Markdown files in docs/ are source of truth; website automatically generated/synced from them", but current structure has separate `docs/` and `website/docs/` directories.

**Research Findings**:
- Current structure: `docs/` (root) contains framework documentation, `website/docs/` contains Docusaurus-specific docs
- Docusaurus reads from `website/docs/` directory by default
- Options for synchronization:
  1. **Symlink approach**: Create symlinks from `website/docs/` to `docs/` (may not work on all systems)
  2. **Copy script**: Build script that copies `docs/` to `website/docs/` before Docusaurus build
  3. **Docusaurus path configuration**: Configure Docusaurus to read from root `docs/` directory
  4. **GitHub Actions**: Automated sync in CI/CD pipeline

**Decision**: Use Docusaurus path configuration to read directly from root `docs/` directory

**Rationale**: 
- Simplest approach - no sync needed, single source of truth
- Docusaurus supports custom doc paths via configuration
- Eliminates sync complexity and potential drift
- Aligns with "markdown files are source of truth" requirement

**Alternatives Considered**:
- Copy script: Adds build complexity, risk of drift
- Symlinks: Platform-specific issues, not reliable in CI/CD
- Manual sync: Error-prone, violates automation requirement

**Implementation**: Update `docusaurus.config.js` to set `docs.path` to `../docs` (relative to website directory)

### 2. Documentation Template Structure

**Question**: What template structure should guides, examples, cookbooks, and use cases follow?

**Context**: Need consistent structure across all documentation types to ensure quality and discoverability.

**Research Findings**:
- Existing guides in `docs/guides/` follow pattern: Introduction → Prerequisites → Step-by-step → Examples → Testing → Best Practices
- Existing cookbooks in `docs/cookbook/` follow pattern: Problem → Solution → Code → Explanation
- Existing use cases in `docs/use-cases/` follow pattern: Overview → Requirements → Implementation → Results
- Examples in `examples/` include: README.md, main.go, *_test.go

**Decision**: Use standardized templates for each documentation type:
- **Guides**: Introduction → Prerequisites → Concepts → Step-by-step Tutorial → Code Examples → Testing → Best Practices → Troubleshooting → Related Resources
- **Cookbooks**: Problem Statement → Solution Overview → Code Example → Explanation → Testing → Related Recipes
- **Use Cases**: Overview → Business Context → Requirements → Architecture → Implementation → Results → Lessons Learned → Related Use Cases
- **Examples**: README.md (description, prerequisites, usage), main.go (production-ready code), *_test.go (complete test suite)

**Rationale**: 
- Ensures consistency across all documentation
- Makes it easy for users to find information
- Follows existing patterns where they exist
- Supports cross-referencing between resources

**Alternatives Considered**:
- Free-form structure: Too inconsistent, harder to maintain
- Minimal structure: Insufficient for comprehensive documentation

### 3. Code Example Testing Patterns

**Question**: What specific test patterns should examples demonstrate beyond test_utils.go basics?

**Context**: All examples must include complete, passing test suites that teach testing patterns.

**Research Findings**:
- Framework uses `test_utils.go` with AdvancedMock, MockOption, ConcurrentTestRunner
- `advanced_test.go` includes table-driven tests, benchmarks, concurrency/error handling
- Integration tests use `tests/integration/utils/integration_helper.go`
- Examples should demonstrate: unit tests, integration tests, benchmarks, error scenarios

**Decision**: Each example must include:
1. **Unit tests** using AdvancedMock patterns
2. **Integration tests** for cross-package interactions
3. **Table-driven tests** for multiple scenarios
4. **Error handling tests** for failure cases
5. **Benchmarks** for performance-critical examples (optional but recommended)

**Rationale**:
- Teaches comprehensive testing patterns
- Ensures examples are production-ready
- Demonstrates framework testing capabilities
- Aligns with framework's testing standards

**Alternatives Considered**:
- Minimal tests: Insufficient for teaching patterns
- Only unit tests: Missing integration and error scenarios

### 4. OTEL Instrumentation in Examples

**Question**: How should OTEL instrumentation be demonstrated in code examples?

**Context**: All examples must demonstrate OTEL metrics with standardized naming.

**Research Findings**:
- Framework uses standardized metric naming: `beluga.{package}.operation_duration_seconds}`
- OTEL patterns include: spans for tracing, metrics (counters, histograms), structured logging
- Examples should show: metric creation, span creation, error recording, context propagation

**Decision**: Each example must demonstrate:
1. **Metric creation** with standardized naming
2. **Span creation** for tracing (where applicable)
3. **Error recording** with `span.RecordError()`
4. **Context propagation** across function calls
5. **Structured logging** with OTEL context

**Rationale**:
- Teaches proper observability patterns
- Ensures examples follow framework standards
- Demonstrates production-ready observability
- Aligns with framework's OTEL integration

**Alternatives Considered**:
- Minimal instrumentation: Insufficient for teaching patterns
- Only metrics: Missing tracing and logging aspects

### 5. Documentation Cross-Referencing Strategy

**Question**: How should documentation resources be cross-referenced for discoverability?

**Context**: Success criteria require related resources (guides → examples → cookbooks → use cases) to be easily discoverable and cross-referenced.

**Research Findings**:
- Docusaurus supports markdown links and sidebar navigation
- Existing docs use relative links between related resources
- Sidebar structure in `sidebars.js` organizes documentation hierarchically

**Decision**: Implement cross-referencing via:
1. **"Related Resources" section** at end of each guide/cookbook/use case
2. **Consistent linking pattern**: Use relative paths, include link text and brief description
3. **Sidebar organization**: Group related resources in sidebar categories
4. **Frontmatter metadata**: Add tags/categories for programmatic linking (future enhancement)

**Rationale**:
- Improves discoverability
- Guides users through learning paths
- Maintains consistency
- Supports both manual browsing and future automation

**Alternatives Considered**:
- No cross-referencing: Poor user experience
- Only sidebar: Insufficient, users may not see relationships

## Technical Decisions Summary

| Decision Area | Decision | Rationale |
|--------------|----------|------------|
| Sync Mechanism | Docusaurus reads from root `docs/` | Single source of truth, no sync complexity |
| Template Structure | Standardized templates per doc type | Consistency and maintainability |
| Testing Patterns | Comprehensive test suites (unit, integration, table-driven, error, benchmarks) | Teaches production-ready patterns |
| OTEL Instrumentation | Full observability (metrics, spans, errors, context, logging) | Demonstrates framework capabilities |
| Cross-Referencing | Related Resources sections + sidebar + links | Improves discoverability |

## Implementation Notes

- All research questions resolved - no NEEDS CLARIFICATION markers remain
- Ready to proceed to Phase 1 (Design & Contracts)
- Documentation structure follows existing patterns where possible
- All decisions align with framework standards and user requirements
