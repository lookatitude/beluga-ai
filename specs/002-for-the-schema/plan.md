
# Implementation Plan: Schema Package Standards Adherence

**Branch**: `002-for-the-schema` | **Date**: October 5, 2025 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/home/swift/Projects/lookatitude/beluga-ai/specs/002-for-the-schema/spec.md`

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
Bring the schema package into full compliance with Beluga AI Framework design patterns by adding missing infrastructure components: comprehensive benchmark tests, organized mock implementations in internal/mock/, health check interfaces, enhanced table-driven testing patterns, and complete OTEL tracing with span management. Preserve all existing schema functionality while adding the missing standardization gaps identified in the constitutional requirements.

## Technical Context
**Language/Version**: Go 1.24.0  
**Primary Dependencies**: go-playground/validator/v10, go.opentelemetry.io/otel (metric/trace), github.com/stretchr/testify (mock/assert)  
**Storage**: N/A (schema/data structures package)  
**Testing**: Go standard testing + testify/mock + testify/assert  
**Target Platform**: Cross-platform Go package (Linux, macOS, Windows)
**Project Type**: single (Go package within larger framework)  
**Performance Goals**: <1ms message creation/validation, <100μs factory functions, efficient memory allocation  
**Constraints**: Zero breaking changes to existing API, backward compatibility required, maintain extensibility patterns  
**Scale/Scope**: Central data contract layer for entire Beluga AI Framework, used by all 14+ packages

**Additional Context from User**: For the 'schema' package: Plan corrections with Go interfaces, validation tags. Include integration test additions.

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Package Structure Compliance
- [x] Package follows standard layout (iface/, internal/, ✓config.go, ✓metrics.go, errors.go via iface/errors.go)
- [x] Multi-provider packages implement global registry pattern (N/A - schema is single-purpose data structures)
- [x] All required files present (✓test_utils.go, ❌advanced_test.go missing, ✓README.md)
- [❌] Missing: internal/mock/ directory structure (currently mocks in test_utils.go)
- [❌] Missing: dedicated advanced_test.go with comprehensive benchmarks

### Design Principles Compliance  
- [x] Interfaces follow ISP (Message, ChatHistory are focused, single responsibility)
- [x] Dependencies injected via constructors (factory functions with functional options)
- [x] Single responsibility per package/struct (schema = data contracts only)
- [x] Functional options used for configuration (WithMaxMessages, WithPersistence, etc.)

### Observability & Quality Standards
- [x] OTEL metrics implementation mandatory (comprehensive metrics.go with 1000+ lines)
- [x] Structured error handling with Op/Err/Code pattern (iface/errors.go with error codes)
- [❌] Comprehensive testing requirements (missing benchmark tests, need enhanced coverage)
- [❌] Integration testing for cross-package interactions (need integration test additions)

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
```
pkg/schema/                    # Target package for standards compliance
├── iface/                     # ✅ Interface definitions (existing)
│   ├── message.go             # ✅ Message and ChatHistory interfaces
│   └── errors.go              # ✅ Structured error types and codes
├── internal/                  # ✅ Private implementations (existing)
│   ├── message.go             # ✅ Message type implementations
│   ├── document.go            # ✅ Document implementation
│   ├── history.go             # ✅ Chat history implementation
│   ├── agent_io.go            # ✅ Agent I/O, A2A communication, and event types
│   └── mock/                  # ❌ Missing - organized mock implementations
│       ├── message.go         # 📋 To add - message mocks
│       ├── history.go         # 📋 To add - history mocks
│       └── generated.go       # 📋 To add - code generated mocks
├── config.go                  # ✅ Configuration structs and validation
├── schema.go                  # ✅ Main package API and factory functions
├── metrics.go                 # ✅ OpenTelemetry metrics integration
├── test_utils.go              # ✅ Testing utilities (existing)
├── advanced_test.go           # ❌ Missing - comprehensive test suite
├── schema_test.go             # ✅ Basic package tests (existing)
└── README.md                  # ✅ Documentation

tests/integration/            # 📋 To add - integration test directory
├── schema_integration_test.go # 📋 To add - cross-package integration tests
└── benchmark_test.go          # 📋 To add - performance benchmarks
```

**Structure Decision**: Single Go package enhancement within existing Beluga AI Framework. Focus on pkg/schema/ directory with addition of missing constitutional requirements: internal/mock/ structure, advanced_test.go, and integration testing directory.

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

**Task Generation Strategy**:
- Load `.specify/templates/tasks-template.md` as base
- Generate tasks from Phase 1 design docs (contracts, data-model.md, quickstart.md)
- Each contract requirement → contract validation task [P]
- Each data model entity → implementation task [P]
- Each benchmark specification → benchmark implementation task [P]
- Each mock specification → mock generation/implementation task [P]
- Integration test requirements → integration test tasks
- Health check requirements → health check implementation tasks
- Documentation updates → documentation enhancement tasks

**Specific Task Categories**:
1. **Mock Infrastructure Tasks** (Parallel):
   - Create internal/mock/ directory structure
   - Implement mockery configuration (.mockery.yaml)
   - Add go generate directives to existing files
   - Generate mocks for Message and ChatHistory interfaces
   - Migrate existing test utilities to use organized mocks

2. **Benchmark Testing Tasks** (Parallel):
   - Create advanced_test.go with comprehensive benchmark suite
   - Implement BenchmarkNewHumanMessage with memory allocation tracking
   - Implement BenchmarkMessageValidation performance tests
   - Implement BenchmarkFactoryFunctions timing tests
   - Implement BenchmarkConcurrentMessageCreation concurrency tests
   - Add benchmark CI/CD integration for regression detection

3. **Health Check Tasks**:
   - Design and implement ValidationHealthChecker
   - Design and implement ConfigurationHealthChecker  
   - Design and implement MetricsHealthChecker
   - Add health check interfaces to main package API
   - Integrate health checks with existing OTEL metrics

4. **Enhanced Testing Tasks** (Sequential dependencies):
   - Extend existing table-driven tests with comprehensive edge cases
   - Add error scenario testing for all error codes
   - Implement concurrency testing for thread-safe operations
   - Add cross-package integration tests in tests/integration/
   - Create contract validation tests based on contract specifications

5. **OTEL Tracing Enhancement Tasks** (Parallel):
   - Add span management to all factory functions in schema.go
   - Enhance context propagation in factory functions
   - Add relevant attributes to spans (message type, operation results)
   - Ensure proper span completion and error recording
   - Validate tracing integration with existing metrics

6. **Documentation and Migration Tasks**:
   - Update README.md with new testing and observability features
   - Create migration guide for adopting new patterns
   - Add usage examples for all new infrastructure components
   - Update package documentation with constitutional compliance status

**Ordering Strategy**:
- **Phase 1**: Mock infrastructure and benchmark setup (parallel foundation tasks)
- **Phase 2**: Health checks and OTEL enhancements (build on foundation)  
- **Phase 3**: Enhanced testing implementation (use new mock infrastructure)
- **Phase 4**: Integration tests (validate everything works together)
- **Phase 5**: Documentation and migration guides (final deliverables)

**Dependencies**:
- Mock generation tasks must complete before enhanced testing tasks
- Benchmark infrastructure must be in place before performance validation
- Health check interfaces must be designed before implementation
- Integration tests depend on all other components being functional

**Estimated Output**: 28-32 numbered, ordered tasks in tasks.md covering all constitutional requirements

**Validation Approach**:
- Each task includes acceptance criteria based on contracts
- Performance tasks include specific benchmark targets
- Integration tasks include cross-package validation
- Documentation tasks include executable examples

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
- [x] Phase 1: Design complete (/plan command) - Generated data-model.md, contracts/, quickstart.md, updated agent context
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS (documented compliance gaps)
- [x] Post-Design Constitution Check: PASS (design addresses all constitutional requirements)
- [x] All NEEDS CLARIFICATION resolved (technical context complete)
- [x] Complexity deviations documented (none - straightforward package enhancement)

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*
