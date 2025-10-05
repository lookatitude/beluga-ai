
# Implementation Plan: Core Package Constitutional Compliance Enhancement

**Branch**: `001-for-the-core` | **Date**: 2025-01-05 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/home/swift/Projects/lookatitude/beluga-ai/specs/001-for-the-core/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from file system structure or context (web=frontend+backend, mobile=app+api)
   → Set Structure Decision based on project type
3. Fill the Constitution Check section based on the content of the constitution document.
4. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
5. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file (e.g., `CLAUDE.md` for Claude Code, `.github/copilot-instructions.md` for GitHub Copilot, `GEMINI.md` for Gemini CLI, `QWEN.md` for Qwen Code, or `AGENTS.md` for all other agents).
7. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary
Enhance the core package to achieve 100% constitutional compliance while preserving all existing functionality (dependency injection, Runnable interface, utilities, models). Add missing constitutional files (config.go, test_utils.go, advanced_test.go), ensure proper package structure with iface/ directory, and implement comprehensive testing infrastructure. Technical approach: incremental enhancement with backward compatibility, OTEL observability integration, and advanced testing utilities following established framework patterns.

## Technical Context
**Language/Version**: Go 1.21+  
**Primary Dependencies**: OpenTelemetry (go.opentelemetry.io/otel), testify (github.com/stretchr/testify), validator (github.com/go-playground/validator)  
**Storage**: N/A (foundational package)  
**Testing**: Go test framework with testify, table-driven tests, benchmarks, integration tests  
**Target Platform**: Multi-platform (Linux, macOS, Windows) - foundational library package
**Project Type**: single - Go package library enhancement  
**Performance Goals**: <1ms DI resolution, <100µs Runnable invoke overhead, 10,000+ ops/sec throughput  
**Constraints**: Zero breaking changes, preserve all existing APIs, maintain thread safety, negligible memory overhead  
**Scale/Scope**: Foundation for 14 framework packages, supports complex AI workflows with thousands of components

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Package Structure Compliance
- [x] Package has metrics.go ✅ (already implemented with OTEL)
- [x] Package has errors.go ✅ (FrameworkError types implemented)
- [x] Package has interfaces.go ✅ (Runnable, Retriever, HealthChecker)
- [x] Package has README.md ✅ (comprehensive documentation)
- [ ] Package missing iface/ directory (need to create and move interfaces)
- [ ] Package missing config.go (need constitutional configuration management)
- [ ] Package missing test_utils.go (need advanced testing utilities)
- [ ] Package missing advanced_test.go (need comprehensive test suites)
- [x] Multi-provider pattern N/A (core is foundational, not multi-provider)

### Design Principles Compliance  
- [x] Interfaces follow ISP ✅ (Runnable, Retriever, HealthChecker are focused)
- [x] Dependencies injected via constructors ✅ (DI container implements DIP)
- [x] Single responsibility per package/struct ✅ (core focuses on foundational utilities)
- [x] Functional options used for configuration ✅ (DIOption pattern implemented)

### Observability & Quality Standards
- [x] OTEL metrics implementation ✅ (NewMetrics with meter.Meter, tracer.Tracer)
- [x] Structured error handling ✅ (FrameworkError with Op/Err/Code pattern)
- [ ] Comprehensive testing requirements (basic tests exist, need constitutional upgrade)
- [ ] Integration testing for cross-package interactions (need to add to tests/integration/)

*Reference: Constitution v1.0.0 - See `.specify/memory/constitution.md`*

## Project Structure

### Documentation (this feature)
```
specs/[###-feature]/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
**Current Core Package Structure:**
```
pkg/core/
├── benchmark_test.go          # Existing benchmarks
├── di_test.go                 # DI container tests
├── di.go                      # Dependency injection container ✅
├── errors_test.go             # Error handling tests
├── errors.go                  # Framework error types ✅
├── integration_test.go        # Basic integration tests
├── interfaces.go              # Core interface definitions ✅
├── metrics.go                 # OTEL metrics implementation ✅
├── model/                     # Core data models ✅
├── README.md                  # Package documentation ✅
├── runnable_test.go           # Runnable tests
├── runnable.go                # Runnable interface ✅
├── traced_runnable.go         # Tracing instrumentation ✅
└── utils/                     # Utility functions ✅
```

**Target Constitutional Structure:**
```
pkg/core/
├── iface/                     # 🆕 Interfaces moved for constitutional compliance
│   ├── interfaces.go          # Core interfaces (Runnable, Retriever, HealthChecker)
│   ├── options.go             # Option interface and implementations
│   └── errors.go              # Error interface definitions
├── internal/                  # Private implementation details
│   ├── di_impl.go             # DI container implementation
│   └── utils_impl.go          # Internal utilities
├── config.go                  # 🆕 Configuration management with functional options
├── metrics.go                 # ✅ OTEL metrics (already compliant)
├── errors.go                  # ✅ Framework error types (already compliant)
├── core.go                    # Main package interfaces and factories
├── test_utils.go              # 🆕 Advanced testing utilities and mocks
├── advanced_test.go           # 🆕 Comprehensive test suites
├── model/                     # ✅ Core data models (preserve)
├── utils/                     # ✅ Utility functions (preserve)
└── README.md                  # ✅ Package documentation (enhance)
```

**Structure Decision**: Enhance existing Go package structure to achieve constitutional compliance while preserving all current functionality. Move interfaces to iface/ directory, add missing constitutional files, and upgrade testing infrastructure to enterprise-grade standards. All changes maintain backward compatibility.

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - For each NEEDS CLARIFICATION → research task
   - For each dependency → best practices task
   - For each integration → patterns task

2. **Generate and dispatch research agents**:
   ```
   For each unknown in Technical Context:
     Task: "Research {unknown} for {feature context}"
   For each technology choice:
     Task: "Find best practices for {tech} in {domain}"
   ```

3. **Consolidate findings** in `research.md` using format:
   - Decision: [what was chosen]
   - Rationale: [why chosen]
   - Alternatives considered: [what else evaluated]

**Output**: research.md with all NEEDS CLARIFICATION resolved

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - Entity name, fields, relationships
   - Validation rules from requirements
   - State transitions if applicable

2. **Generate API contracts** from functional requirements:
   - For each user action → endpoint
   - Use standard REST/GraphQL patterns
   - Output OpenAPI/GraphQL schema to `/contracts/`

3. **Generate contract tests** from contracts:
   - One test file per endpoint
   - Assert request/response schemas
   - Tests must fail (no implementation yet)

4. **Extract test scenarios** from user stories:
   - Each story → integration test scenario
   - Quickstart test = story validation steps

5. **Update agent file incrementally** (O(1) operation):
   - Run `.specify/scripts/bash/update-agent-context.sh cursor`
     **IMPORTANT**: Execute it exactly as specified above. Do not add or remove any arguments.
   - If exists: Add only NEW tech from current plan
   - Preserve manual additions between markers
   - Update recent changes (keep last 3)
   - Keep under 150 lines for token efficiency
   - Output to repository root

**Output**: data-model.md, /contracts/*, failing tests, quickstart.md, agent-specific file

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy for Core Package**:
- Load `.specify/templates/tasks-template.md` as base with constitutional compliance additions
- **Constitutional Structure Tasks**: Create iface/ directory, config.go, test_utils.go, advanced_test.go
- **Interface Migration Tasks**: Move interfaces to iface/ with backward compatibility re-exports
- **Testing Enhancement Tasks**: Upgrade existing tests to constitutional standards
- **Integration Tasks**: Add cross-package integration tests to tests/integration/
- **Performance Tasks**: Add comprehensive benchmarks for all critical operations
- **Documentation Tasks**: Update README.md and add package documentation

**Core Package Specific Ordering**:
- **Phase 1**: Constitutional structure (iface/, config.go, test_utils.go, advanced_test.go) [P]
- **Phase 2**: Interface migration with re-exports for compatibility [Sequential]
- **Phase 3**: Advanced testing infrastructure implementation [P]  
- **Phase 4**: Integration testing and performance benchmarks [P]
- **Phase 5**: Documentation updates and final validation

**Estimated Output**: 15-20 numbered, ordered tasks in tasks.md focusing on constitutional compliance

**Core Package Specifics**:
- Preserve ALL existing functionality and APIs
- Maintain backward compatibility through re-exports
- Focus on additive changes rather than modifications
- Prioritize testing infrastructure for framework foundation role

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |


## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS (4 gaps identified, no violations)
- [x] Post-Design Constitution Check: PASS (no new violations introduced)
- [x] All NEEDS CLARIFICATION resolved (none present)
- [x] Complexity deviations documented (none present)

---
*Based on Constitution v1.0.0 - See `.specify/memory/constitution.md`*
