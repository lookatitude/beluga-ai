
# Implementation Plan: Identify and Fix Long-Running Tests

**Branch**: `001-go-package-by` | **Date**: 2025-01-27 | **Spec**: `/specs/001-go-package-by/spec.md`
**Input**: Feature specification from `/specs/001-go-package-by/spec.md`

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
This feature implements an automated analysis and fix system for identifying and resolving long-running test issues across all Go packages in the Beluga AI Framework. The system will analyze test files package-by-package, detect performance issues (infinite loops, missing timeouts, large iterations, actual implementation usage instead of mocks), and automatically apply fixes where safe. The approach uses Go AST parsing for pattern detection, dual validation (interface compatibility + test execution) for safe fixes, and follows the established mock pattern for creating missing mock implementations.

## Technical Context
**Language/Version**: Go 1.24.0 (toolchain go1.24.2)  
**Primary Dependencies**: go/ast, go/parser, go/token (for AST analysis), testify/mock (for mock pattern detection), go/format (for code generation)  
**Storage**: N/A (analysis tool, no persistent storage required)  
**Testing**: Go testing package (go test), testify/assert for assertions  
**Target Platform**: Linux/macOS/Windows (Go cross-platform)  
**Project Type**: single (Go framework/library project)  
**Performance Goals**: Analysis completes in <30 seconds for entire codebase, fix application completes in <5 minutes (baseline: standard developer machine with 4+ CPU cores, 8GB+ RAM, SSD storage)  
**Constraints**: Must not break existing tests, must preserve test intent, must validate fixes before applying, must create backups before modifications  
**Scale/Scope**: 14 packages with ~100+ test files, ~500+ test functions to analyze

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Package Structure Compliance
- [x] Package follows standard layout (iface/, internal/, providers/, config.go, metrics.go, errors.go, etc.) - N/A: This is a tool/utility, not a framework package
- [x] Multi-provider packages implement global registry pattern - N/A: Not applicable for analysis tool
- [x] All required files present (test_utils.go, advanced_test.go, README.md) - N/A: Tool package structure differs

**Note**: This feature creates a standalone analysis tool, not a framework package. The tool will be placed in `scripts/` or `cmd/` directory following Go CLI tool conventions, not the standard package structure. This is acceptable as it's a development utility, not a framework component.

### Design Principles Compliance  
- [x] Interfaces follow ISP (small, focused, "er" suffix for single-method) - Tool will use focused interfaces (Analyzer, Fixer, Reporter)
- [x] Dependencies injected via constructors (DIP compliance) - All dependencies (AST parser, file system, test runner) injected via constructors
- [x] Single responsibility per package/struct (SRP compliance) - Tool has single responsibility: analyze and fix test performance issues
- [x] Functional options used for configuration (composition over inheritance) - Configuration via functional options pattern

### Observability & Quality Standards
- [x] OTEL metrics implementation mandatory (no custom metrics) - Tool will use structured logging, metrics optional for CLI tool
- [x] Structured error handling with Op/Err/Code pattern - Custom error types with Op/Err/Code pattern
- [x] Comprehensive testing requirements (100% coverage, mocks, benchmarks) - Tool itself will have comprehensive tests
- [x] Integration testing for cross-package interactions - Integration tests will verify tool works across all packages

**Note**: As a CLI tool/utility, full OTEL metrics may be optional, but structured logging and error handling will follow framework standards.

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
cmd/
└── test-analyzer/          # CLI tool for test analysis and fixing
    ├── main.go             # Entry point
    ├── analyzer.go         # Core analysis logic
    ├── fixer.go            # Automated fix application
    ├── reporter.go         # Report generation
    └── internal/
        ├── ast/            # AST parsing utilities
        ├── patterns/       # Pattern detection logic
        ├── mocks/          # Mock generation logic
        └── validation/     # Fix validation logic

scripts/
└── analyze-tests.sh       # Convenience wrapper script

pkg/                        # Existing framework packages (analysis target)
├── agents/
├── llms/
├── memory/
└── [12 more packages...]

tests/
└── integration/
    └── test-analyzer/      # Integration tests for the tool
        ├── analyzer_test.go
        ├── fixer_test.go
        └── end_to_end_test.go
```

**Structure Decision**: Single project structure with CLI tool in `cmd/test-analyzer/` following Go standard project layout. The tool analyzes existing packages in `pkg/` directory. This follows Go conventions for CLI tools while keeping the tool separate from framework packages.

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
- Core components: Analyzer, Fixer, Reporter, MockGenerator, PatternDetector
- Each interface → implementation task
- Each data model entity → struct/type definition task [P]
- Each contract method → implementation task
- CLI interface → main.go and command parsing task
- Test utilities → test file creation tasks [P]
- Integration tests → end-to-end test scenarios

**Ordering Strategy**:
- Foundation first: Data models and types
- Core logic: Analyzer and PatternDetector (can be parallel)
- Fix application: Fixer and MockGenerator (depends on Analyzer)
- Reporting: Reporter (depends on Analyzer)
- CLI: Main entry point and command parsing (depends on all components)
- Testing: Unit tests for each component [P], then integration tests
- Mark [P] for parallel execution (independent files/components)

**Task Categories**:
1. **Data Models** (5-7 tasks): Define all entities from data-model.md
2. **AST Parsing** (3-4 tasks): AST utilities, file parsing, function extraction
3. **Pattern Detection** (6-8 tasks): Each pattern detector (infinite loops, timeouts, iterations, etc.)
4. **Mock Generation** (4-5 tasks): Interface analysis, mock code generation, template generation
5. **Fix Application** (4-5 tasks): Code modification, backup/rollback, validation
6. **Reporting** (3-4 tasks): JSON, HTML, Markdown, Plain text formatters
7. **CLI** (2-3 tasks): Command parsing, flag handling, main entry point
8. **Testing** (8-10 tasks): Unit tests for each component, integration tests, end-to-end tests

**Estimated Output**: 35-45 numbered, ordered tasks in tasks.md

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
- [x] Phase 0: Research complete (/plan command) - research.md generated
- [x] Phase 1: Design complete (/plan command) - data-model.md, contracts/, quickstart.md, agent file updated
- [x] Phase 2: Task planning complete (/plan command - describe approach only) - approach documented
- [x] Phase 3: Tasks generated (/tasks command) - tasks.md created
- [x] Phase 4: Implementation complete - All tasks implemented
- [x] Phase 5: Validation passed - Tests passing, quickstart validated

**Gate Status**:
- [x] Initial Constitution Check: PASS - Tool structure approved, principles followed
- [x] Post-Design Constitution Check: PASS - Design follows constitutional principles
- [x] All NEEDS CLARIFICATION resolved - All clarifications from spec.md addressed
- [x] Complexity deviations documented - Tool structure deviation documented and justified

**Artifacts Generated**:
- [x] research.md - Technology decisions and implementation approach
- [x] data-model.md - Complete data model with entities and relationships
- [x] contracts/cli-interface.md - CLI command interface contract
- [x] contracts/analyzer-interface.md - Analyzer interface contract
- [x] contracts/fixer-interface.md - Fixer and MockGenerator interface contracts
- [x] contracts/reporter-interface.md - Reporter interface contract
- [x] quickstart.md - User guide with examples and validation tests
- [x] Agent context file updated - Cursor IDE context updated with new tech stack

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*
