<!--
Sync Impact Report:
Version change: 1.0.0 → 1.1.0 (Task execution clarification)
Modified principles: Implementation Standards completely restructured
Added sections: Task Execution Requirements with feature type classification, Implementation Workflow Steps
Removed sections: None
Templates requiring updates: ✅ tasks-template.md (add task type classification), ✅ implement.md (clarify execution rules)
Follow-up TODOs: Review existing specs for proper task classification (analysis vs implementation)
Key Changes: Added explicit distinction between NEW FEATURE, ANALYSIS, and CORRECTION tasks with clear file path requirements
-->

# Beluga AI Framework Constitution (Focused)

## Purpose
- Ensure consistent, extensible, configurable, and observable Go packages across the Beluga AI Framework.

## Core Principles (Required)
- Interface Segregation (ISP): small, focused interfaces.
- Dependency Inversion (DIP): depend on abstractions; constructor injection only.
- Single Responsibility (SRP): one clear reason to change per package/struct.
- Composition over Inheritance: prefer interface embedding + functional options.

## Framework Patterns (Required)
- Standard package layout:
  pkg/{name}/
  - iface/ (public interfaces, types)
  - internal/ (private implementation)
  - providers/ (only if multi-backend)
  - config.go (config structs with mapstructure,yaml,env,validate tags; defaults)
  - metrics.go (OpenTelemetry metrics/tracing integration)
  - errors.go (custom error: Op/Err/Code + codes)
  - {name}.go (interfaces + factories)
  - README.md (usage + examples)
- Multi-provider registry (if applicable):
  - Register(name string, creator func(ctx context.Context, cfg Config) (Interface, error))
  - NewProvider(ctx context.Context, name string, cfg Config) (Interface, error)
- Interfaces: "-er" for single-method (Embedder), nouns for multi-method (VectorStore). Use embedding to extend without breaking changes.
- Configuration: functional options; validate at creation; respect context for timeouts/cancellation.
- Observability: OTEL spans in public methods; structured logging; counters/histograms with labels; add health checks where relevant.
- Error handling: custom error type with codes; wrap and preserve chains; never swallow errors; always respect context.
- Dependencies: constructor injection; avoid globals/singletons (except safe registries).
- Testing: table-driven tests; mocks in internal/mock/; integration tests in tests/integration/; benchmarks for perf-critical paths; thread-safety tests when concurrency is involved.
- Documentation: package comment, function docs with examples, README for complex packages.
- Code generation: generate mocks/validation/metrics where beneficial.
- Evolution: SemVer; deprecate with notices; provide migration guides.

**Rationale**: Dependency injection enables testing with mocks, supports multiple provider implementations, and makes the framework extensible without modification.

### III. Single Responsibility Principle (SRP)
Each package MUST have one primary responsibility. Each struct/function MUST have one reason to change. Packages MUST be focused and cohesive around a single AI/ML domain (embeddings, memory, agents, etc.). NO mixing of concerns across package boundaries.

**Rationale**: Single responsibility ensures maintainable code, clear boundaries, and enables teams to work independently on different AI components.

### IV. Composition over Inheritance with Functional Options
MUST prefer embedding interfaces/structs over type hierarchies. ALL configuration MUST use functional options pattern. NO complex inheritance structures. Enable flexible composition of AI behaviors through interface composition and functional options.

**Rationale**: Composition provides flexibility for AI workflows where different combinations of capabilities are needed for different use cases.

## Implementation Standards

### Package Structure (MANDATORY COMPLIANCE)
ALL packages MUST follow this exact structure with NO deviations:

```
pkg/{package_name}/
├── iface/                    # Interfaces and types (REQUIRED)
├── internal/                 # Private implementation details
├── providers/               # Provider implementations (for multi-provider packages)
├── config.go                # Configuration structs and validation (REQUIRED)
├── metrics.go               # OTEL metrics implementation (REQUIRED)
├── errors.go                # Custom error types with Op/Err/Code pattern (REQUIRED)
├── {package_name}.go        # Main interfaces and factory functions
├── factory.go OR registry.go # Global factory/registry for multi-provider packages
├── test_utils.go            # Advanced testing utilities and mocks (REQUIRED)
├── advanced_test.go         # Comprehensive test suites (REQUIRED)
└── README.md                # Package documentation (REQUIRED)
```

**Status**: 100% compliance achieved across all 14 packages. NO exceptions permitted.

### Global Registry Pattern (MULTI-PROVIDER PACKAGES)
ALL multi-provider packages MUST implement the global registry pattern for consistent provider management:

```go
type ProviderRegistry struct {
    mu       sync.RWMutex
    creators map[string]func(ctx context.Context, config Config) (Interface, error)
}

var globalRegistry = NewProviderRegistry()
func RegisterGlobal(name string, creator func(...) (Interface, error))
func NewProvider(ctx context.Context, name string, config Config) (Interface, error)
```

**Mandatory for**: embeddings, memory, agents, vectorstores, llms, chatmodels, retrievers, prompts, orchestration, monitoring, config, server.

### OpenTelemetry Integration (MANDATORY)
ALL packages MUST implement standardized OTEL metrics, tracing, and logging. NO custom metrics implementations. ALL packages MUST include metrics.go with:

```go
func NewMetrics(meter metric.Meter, tracer trace.Tracer) (*Metrics, error)
func (m *Metrics) RecordOperation(ctx context.Context, operation string, duration time.Duration, success bool)
func NoOpMetrics() *Metrics
```

**Status**: 100% OTEL standardization complete across all packages.

### Error Handling (ENFORCED STANDARD)
ALL packages MUST implement structured error handling with Op/Err/Code pattern:

```go
type {Package}Error struct {
    Op   string // operation that failed
    Err  error  // underlying error  
    Code string // error code for programmatic handling
}
```

Standard error codes MUST be defined as constants. ALL errors MUST preserve error chains through Unwrap().

## Testing & Quality Assurance

### Testing Requirements (NON-NEGOTIABLE)
ALL packages MUST implement enterprise-grade testing with 100% compliance:

1. **test_utils.go (REQUIRED)**: Advanced mocking utilities with `AdvancedMock{Package}`, `Mock{Package}Option`, `ConcurrentTestRunner`, performance testing helpers
2. **advanced_test.go (REQUIRED)**: Table-driven tests, concurrency testing, error handling scenarios, performance benchmarks
3. **Integration testing**: Cross-package testing in `tests/integration/` directory
4. **100% test coverage**: ALL public methods MUST have comprehensive test coverage
5. **Performance benchmarks**: ALL critical operations MUST have benchmark tests
6. **Concurrency validation**: ALL packages MUST test thread safety

### Quality Gates (ENFORCED)
- NO code may be merged without passing ALL tests
- NO performance regressions permitted without explicit approval
- ALL new providers MUST pass standardized interface compliance tests
- ALL cross-package interactions MUST have integration tests

**Status**: Complete testing infrastructure implemented across all packages.

## Implementation Standards

### Task Execution Requirements (MANDATORY)

#### Feature Type Classification
ALL task implementations MUST be classified and executed according to their type:

1. **NEW FEATURE IMPLEMENTATION** (`specs/NNN-feature-name/`):
   - **PRIMARY GOAL**: Create NEW files in `pkg/`, `cmd/`, `internal/` directories
   - **TASKS MUST**: Implement actual Go code following package design patterns
   - **FILE TARGETS**: `pkg/{package}/`, `pkg/{package}/providers/`, `pkg/{package}/internal/`
   - **TESTING**: Create test files in same directories as implementation
   - **SPEC USAGE**: Write planning docs (plan.md, research.md, contracts/) THEN implement code
   - **VALIDATION**: Run tests, verify functionality works end-to-end
   - **EXAMPLE**: "Create embeddings package" → writes files to `pkg/embeddings/*.go`

2. **ANALYSIS/AUDIT TASKS** (`specs/NNN-for-the-{package}/`):
   - **PRIMARY GOAL**: Document findings about EXISTING code in `pkg/{package}/`
   - **TASKS MUST**: Read existing files, write analysis to `specs/` directory only
   - **FILE TARGETS**: `specs/NNN-for-the-{package}/findings/`, `specs/NNN-for-the-{package}/analysis/`
   - **NO CODE CHANGES**: Analysis tasks document only, never modify `pkg/` files
   - **VALIDATION**: Ensure analysis is comprehensive and actionable
   - **EXAMPLE**: "Analyze embeddings package" → writes to `specs/008-for-the-embeddings/findings/*.md`

3. **CORRECTION/ENHANCEMENT TASKS** (follow analysis):
   - **PRIMARY GOAL**: Apply fixes/improvements to EXISTING `pkg/` files based on analysis findings
   - **TASKS MUST**: Modify actual Go files in `pkg/` directories following patterns
   - **FILE TARGETS**: `pkg/{package}/*.go`, `pkg/{package}/providers/*.go`, etc.
   - **PREREQUISITE**: Completed analysis phase with documented findings
   - **TESTING**: Update/fix existing tests, add missing coverage
   - **VALIDATION**: Run full test suite, verify no regressions
   - **EXAMPLE**: After analysis, "Fix error handling in pkg/embeddings/providers/openai.go"

#### Implementation Workflow Steps (ENFORCED)

**For NEW FEATURES**:
```
1. /specify → Create spec.md with feature requirements
2. /plan → Generate research.md, contracts/, data-model.md, quickstart.md
3. /tasks → Generate tasks.md with implementation tasks
4. EXECUTE TASKS:
   Phase 3.1 (Setup): Create directory structure in pkg/
   Phase 3.2 (Tests): Write failing tests in pkg/{package}/*_test.go
   Phase 3.3 (Core): Implement code in pkg/{package}/*.go to pass tests
   Phase 3.4 (Integration): Add to registries, wire dependencies
   Phase 3.5 (Polish): Documentation, benchmarks, README.md
5. Run tests: go test ./pkg/{package}/... -v -cover
6. Commit, push, PR to develop branch
```

**For ANALYSIS (Audit Existing Code)**:
```
1. /specify → Create spec.md describing analysis scope
2. /plan → Generate research.md with analysis approach
3. /tasks → Generate analysis tasks (all write to specs/ directory)
4. EXECUTE ANALYSIS TASKS:
   Phase 3.1: Setup analysis tools
   Phase 3.2: Verify contracts (write findings to specs/)
   Phase 3.3: Analyze entities (write analysis to specs/)
   Phase 3.4: Validate scenarios (write validation to specs/)
   Phase 3.5: Generate reports (write reports to specs/)
5. Review findings → create SEPARATE correction spec if needed
```

**For CORRECTIONS (Fix Existing Code)**:
```
1. Start from analysis findings in specs/NNN-for-the-{package}/
2. Create NEW spec: specs/MMM-fix-{package}-{issue}/
3. Generate correction tasks targeting pkg/{package}/ files
4. EXECUTE CORRECTION TASKS:
   Phase 3.1: Setup test environment
   Phase 3.2: Add missing tests (TDD approach)
   Phase 3.3: Fix actual code in pkg/ to pass tests
   Phase 3.4: Verify no regressions with full suite
   Phase 3.5: Update documentation
5. Run tests: go test ./pkg/{package}/... -v -cover
6. Commit, push, PR to develop branch
```

#### Critical Rules for Task Definitions

1. **EXPLICIT FILE PATHS**: Every task MUST specify exact file paths being created/modified
   - ✅ GOOD: "Implement Embedder interface in pkg/embeddings/iface/embedder.go"
   - ✅ GOOD: "Verify error handling in pkg/embeddings/providers/openai.go"
   - ❌ BAD: "Implement embedder interface" (no path specified)
   - ❌ BAD: "Verify error handling" (no target files)

2. **CLEAR ACTION VERBS**: Use precise verbs indicating file operations
   - **For NEW features**: Create, Implement, Add, Build, Write
   - **For ANALYSIS**: Verify, Analyze, Validate, Document, Review
   - **For CORRECTIONS**: Fix, Update, Enhance, Refactor, Improve

3. **PACKAGE COMPLIANCE**: All implementation tasks MUST reference package design patterns
   - Every task must verify ISP, DIP, SRP compliance
   - All tasks must implement OTEL metrics, error handling patterns
   - Factory pattern for multi-provider packages (mandatory reference)

### Post-Implementation Workflow (MANDATORY)
ALL feature implementations MUST follow the standardized post-implementation workflow:

1. **Verify Implementation Completeness**:
   - ✅ All files in `pkg/` directories created/modified as specified
   - ✅ Tests passing: `go test ./pkg/{package}/... -v -cover`
   - ✅ Linter clean: `golangci-lint run ./pkg/{package}/...`
   - ✅ Documentation updated (README.md, godoc comments)

2. **Comprehensive Commit Message**:
   ```
   feat({package}): [concise description]

   CONSTITUTIONAL COMPLIANCE:
   ✅ ISP: [interface segregation achievements]
   ✅ DIP: [dependency injection implementation]
   ✅ SRP: [single responsibility adherence]
   ✅ Composition: [functional options usage]

   IMPLEMENTATION DETAILS:
   - [key file changes with paths]
   - [test coverage statistics]
   - [observability integration]

   FILES MODIFIED:
   - pkg/{package}/file1.go: [what changed]
   - pkg/{package}/file2.go: [what changed]

   TESTING:
   - go test ./pkg/{package}/... → PASS
   - Coverage: XX%
   ```

3. **Branch Push**: `git push origin {branch-name}`

4. **Pull Request Creation**: PR from feature branch to `develop` with:
   - Implementation summary
   - Constitutional compliance checklist
   - Test results and coverage report
   - Breaking changes (if any)

5. **Merge to Develop**: After CI passes and review approval

6. **Post-Merge Validation**:
   ```bash
   git checkout develop
   git pull origin develop
   go test ./... -v -cover
   go test ./... -bench=. -benchmem
   ```

**Rationale**: Clear task classification prevents confusion between analysis documentation and actual implementation. Explicit file paths ensure accountability and traceability.

## Governance

### Amendment Procedure
Constitution amendments require:
1. **Documentation**: Detailed rationale for changes in GitHub issue/PR
2. **Impact Assessment**: Analysis of affected packages and breaking changes
3. **Migration Plan**: Clear upgrade path for existing implementations
4. **Review**: Approval from framework maintainers
5. **Implementation**: Updates to all affected template files and documentation

### Compliance Review
- ALL pull requests MUST verify constitutional compliance
- Quarterly audits of package compliance with principles
- Automated linting rules MUST enforce structural compliance
- Constitutional violations MUST be documented and justified

### Versioning Policy
- **MAJOR**: Backward incompatible principle changes or removals
- **MINOR**: New principles added or material expansions
- **PATCH**: Clarifications, wording improvements, typo fixes

**Version**: 1.1.0 | **Ratified**: 2025-01-05 | **Last Amended**: 2025-10-05
