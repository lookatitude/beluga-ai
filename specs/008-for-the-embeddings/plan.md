
# Implementation Plan: Embeddings Package Corrections

**Branch**: `008-for-the-embeddings` | **Date**: October 5, 2025 | **Spec**: /specs/008-for-the-embeddings/spec.md
**Input**: Feature specification from `/specs/008-for-the-embeddings/spec.md`

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
Analyze the current embeddings package implementation for OpenAI/Ollama providers, global registry functionality, and performance testing, then plan corrections to ensure full compliance with Beluga AI Framework design patterns including ISP, DIP, SRP, and composition principles. Deliver comprehensive analysis findings and correction roadmap.

## Technical Context
**Language/Version**: Go 1.21+
**Primary Dependencies**: go.opentelemetry.io/otel, github.com/go-playground/validator/v10, github.com/sashabaranov/go-openai, github.com/ollama/ollama/api
**Storage**: N/A (stateless embedding operations)
**Testing**: Go testing framework with table-driven tests, mocks, and benchmarks
**Target Platform**: Linux/Windows/macOS server environments
**Project Type**: Single Go package within multi-package framework
**Performance Goals**: <100ms p95 for single embedding operations, support for batch processing (10-1000 documents), concurrent request handling
**Constraints**: Framework compliance with ISP/DIP/SRP principles, OTEL observability, structured error handling, comprehensive testing (90%+ coverage)
**Scale/Scope**: Multi-provider embeddings package supporting OpenAI, Ollama, and mock providers

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Package Structure Compliance
- [x] Package follows standard layout (iface/, internal/, providers/, config.go, metrics.go, errors.go, etc.)
- [x] Multi-provider packages implement global registry pattern
- [x] All required files present (test_utils.go, advanced_test.go, README.md)

### Design Principles Compliance
- [x] Interfaces follow ISP (small, focused, "er" suffix for single-method)
- [x] Dependencies injected via constructors (DIP compliance)
- [x] Single responsibility per package/struct (SRP compliance)
- [x] Functional options used for configuration (composition over inheritance)

### Observability & Quality Standards
- [x] OTEL metrics implementation mandatory (no custom metrics)
- [x] Structured error handling with Op/Err/Code pattern
- [x] Comprehensive testing requirements (100% coverage, mocks, benchmarks)
- [x] Integration testing for cross-package interactions

*Reference: Constitution v1.0.0 - See `.specify/memory/constitution.md`*

## Project Structure

### Documentation (this feature)
```
specs/008-for-the-embeddings/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
pkg/embeddings/
├── iface/               # Interface definitions and error types
│   ├── iface.go        # Embedder interface definition
│   ├── errors.go       # EmbeddingError type with Op/Err/Code pattern
│   └── iface_test.go   # Interface compliance tests
├── providers/          # Multi-provider implementations
│   ├── openai/         # OpenAI provider implementation
│   ├── ollama/         # Ollama provider implementation
│   └── mock/           # Mock provider for testing
├── internal/           # Private implementation details
├── config.go           # Configuration structs and validation
├── metrics.go          # OpenTelemetry metrics implementation
├── factory.go          # Global registry pattern implementation
├── embeddings.go       # Main factory and interface implementations
├── test_utils.go       # Advanced testing utilities and mocks
├── advanced_test.go    # Comprehensive test suites
├── benchmarks_test.go  # Performance benchmarks
├── README.md           # Package documentation
└── *_test.go          # Unit and integration tests

tests/integration/      # Cross-package integration tests
```

**Structure Decision**: Analysis of existing Beluga AI Framework embeddings package structure. Package follows the mandatory framework layout with iface/, internal/, providers/, config.go, metrics.go pattern. This is a single Go package within the multi-package framework architecture.

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
- Generate tasks from Phase 1 design docs (contracts, data model, quickstart)
- Each correction requirement → implementation task
- Each contract violation → fix task [P]
- Each failing test → investigation and fix task
- Each performance gap → enhancement task
- Documentation gaps → enhancement task [P]
- Integration testing gaps → test implementation task

**Ordering Strategy**:
- Priority order: HIGH severity corrections first, then MEDIUM, then LOW
- Dependency order: Framework compliance fixes before enhancements
- Test reliability fixes before new features
- Mark [P] for parallel execution (independent files/components)

**Estimated Output**: 15-20 numbered, prioritized tasks in tasks.md focusing on:
1. Framework compliance corrections (5-7 tasks)
2. Test suite reliability fixes (3-4 tasks)
3. Performance testing enhancements (2-3 tasks)
4. Documentation improvements (2-3 tasks)
5. Integration testing additions (2-3 tasks)

**Task Categories**:
- **Framework Corrections**: ISP/DIP/SRP violations, missing OTEL, error pattern issues
- **Provider Fixes**: Ollama dimension handling, OpenAI optimizations
- **Testing Improvements**: Fix failing tests, add load testing, improve coverage
- **Documentation**: README enhancements, examples, troubleshooting
- **Integration**: Cross-package testing, end-to-end workflows

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Post-Implementation Workflow (mandatory commit, push, PR, merge to develop process)
**Phase 6**: Validation (run tests, execute quickstart.md, performance validation)

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
- [x] Phase 2: Task planning complete (/tasks command)
- [x] Phase 3: Implementation complete
- [x] Phase 4: Validation passed
- [x] Phase 5: Documentation & polish complete

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [ ] Complexity deviations documented

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*
