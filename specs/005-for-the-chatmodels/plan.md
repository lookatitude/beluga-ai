
# Implementation Plan: ChatModels Package Framework Standards Compliance

**Branch**: `005-for-the-chatmodels` | **Date**: October 5, 2025 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/005-for-the-chatmodels/spec.md`

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
Achieve full framework standards compliance for the ChatModels package by implementing the global registry pattern and complete OTEL integration while preserving existing runnable implementation and interface design. The package currently has excellent foundational architecture with ISP-compliant interfaces and comprehensive testing, but requires specific compliance gaps to be addressed: global registry for provider management, complete OpenTelemetry observability, structured error handling with Op/Err/Code pattern, and factory pattern enhancement.

## Technical Context
**Language/Version**: Go 1.21+ (existing Beluga AI Framework requirement)
**Primary Dependencies**: OpenTelemetry (metrics, tracing, logging), testify/mock (testing), existing ChatModel providers (OpenAI)
**Storage**: N/A (chat model provider APIs - no persistent storage required)
**Testing**: Go test framework with AdvancedMockChatModel, table-driven tests, comprehensive benchmarking
**Target Platform**: Cross-platform Go applications (Linux/macOS/Windows servers and clients)
**Project Type**: Single Go package within larger AI framework (multi-provider package requiring global registry)
**Performance Goals**: Maintain current chat model performance while adding observability overhead <5%
**Constraints**: Preserve existing runnable interface, maintain backward compatibility, zero breaking changes to current API
**Scale/Scope**: Enterprise-grade chat model abstraction supporting multiple concurrent providers with production observability

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Package Structure Compliance
- [x] Package follows standard layout (iface/, internal/, providers/, config.go, metrics.go, errors.go, etc.)
- [ ] Multi-provider packages implement global registry pattern (MISSING - needs registry.go)
- [x] All required files present (test_utils.go, advanced_test.go, README.md)

### Design Principles Compliance  
- [x] Interfaces follow ISP (MessageGenerator, StreamMessageHandler, ModelInfoProvider, HealthChecker - excellently focused)
- [x] Dependencies injected via constructors (DIP compliance - good factory pattern foundation)
- [x] Single responsibility per package/struct (SRP compliance - focused on chat models only)
- [x] Functional options used for configuration (composition over inheritance - well implemented)

### Observability & Quality Standards
- [ ] OTEL metrics implementation mandatory (PARTIAL - interfaces ready but full implementation needed)
- [ ] Structured error handling with Op/Err/Code pattern (PARTIAL - basic errors exist, need Op/Err/Code)
- [x] Comprehensive testing requirements (100% coverage, mocks, benchmarks - AdvancedMockChatModel is excellent)
- [x] Integration testing for cross-package interactions (good foundation exists)

**Compliance Status**: ✅ STRONG FOUNDATION with **3 key gaps**: Global registry, complete OTEL, Op/Err/Code pattern

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
pkg/chatmodels/                    # ChatModels package implementation
├── iface/                         # Interface definitions (ISP compliant)
│   └── chatmodel.go              # Core ChatModel interfaces (excellent design)
├── internal/                     # Private implementations
│   └── mock/                     # Mock implementations for testing
├── providers/                    # Provider implementations  
│   └── openai/                   # OpenAI provider (needs registry compliance)
├── config.go                     # Configuration management (functional options)
├── metrics.go                    # OTEL metrics (needs full implementation)
├── errors.go                     # Custom error types (needs Op/Err/Code pattern)
├── chatmodels.go                 # Main interfaces and factory functions
├── registry.go                   # [TO BE ADDED] Global registry implementation
├── test_utils.go                 # Advanced testing utilities (excellent AdvancedMockChatModel)
├── advanced_test.go              # Comprehensive test suites  
├── chatmodels_test.go            # Core package tests
└── README.md                     # Package documentation

tests/integration/               # Cross-package integration tests
├── end_to_end/                  # Complete workflow tests
└── package_pairs/               # ChatModels package integration tests
    ├── chatmodels_llms_test.go
    ├── chatmodels_memory_test.go
    └── chatmodels_agents_test.go
```

**Structure Decision**: Existing single Go package structure within the Beluga AI Framework. The ChatModels package has excellent foundational architecture following ISP with focused interfaces. Primary work involves adding registry.go for global registry pattern, completing OTEL integration in metrics.go, and enhancing error handling with Op/Err/Code pattern while preserving the excellent existing runnable interface design.

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
- Generate tasks from Phase 1 design docs (registry contracts, metrics contracts, error contracts, data model, quickstart examples)
- Each contract interface → interface test task [P]
- Each data model entity → struct creation and validation task [P]
- Each compliance gap → implementation task with OTEL/registry integration
- Registry pattern implementation tasks
- Error handling enhancement tasks  
- Backward compatibility validation tasks

**Ordering Strategy**:
- TDD order: Contract tests before implementation
- Dependency order: Data models → Registry → OTEL → Error handling → Integration
- Mark [P] for parallel execution (independent compliance features)
- Constitutional compliance order: Registry → Observability → Error patterns → Testing

**Task Categories**:
1. **Registry Implementation**: Global registry pattern with thread-safe operations (Tasks 1-6)
2. **OTEL Integration**: Complete metrics, tracing, and logging implementation (Tasks 7-12)
3. **Error Handling**: Op/Err/Code pattern with structured error types (Tasks 13-18)
4. **Interface Preservation**: Ensure backward compatibility of existing interfaces (Tasks 19-22)
5. **Testing Enhancement**: Validate compliance features work with existing AdvancedMockChatModel (Tasks 23-26)

**Estimated Output**: 24-28 numbered, ordered tasks in tasks.md focused on compliance gaps

**Key Task Examples**:
- Create ChatModelRegistry with thread-safe provider management
- Implement complete OTEL metrics with RecordOperation method
- Enhance error handling with ChatModelError Op/Err/Code pattern
- Add registry integration to existing providers while preserving interfaces
- Create comprehensive compliance testing with existing mock infrastructure
- Validate zero breaking changes to current runnable interface

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
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [x] Complexity deviations documented (none required)

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*
