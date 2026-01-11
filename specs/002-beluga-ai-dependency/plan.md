
# Implementation Plan: Fix Corrupted Mock Files in Beluga-AI Package

**Branch**: `002-beluga-ai-dependency` | **Date**: 2025-01-27 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-beluga-ai-dependency/spec.md`

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
Fix corrupted mock files in the beluga-ai package that are missing package declarations, preventing compilation of dependent projects. The fix involves adding the correct `package` declaration to 5 mock files: `pkg/core/di_mock.go`, `pkg/prompts/advanced_mock.go`, `pkg/vectorstores/iface/iface_mock.go`, `pkg/vectorstores/advanced_mock.go`, and `pkg/memory/advanced_mock.go`. This is a straightforward file correction that maintains API compatibility and requires no structural changes.

## Technical Context
**Language/Version**: Go 1.24.0  
**Primary Dependencies**: github.com/stretchr/testify/mock (for mock.Mock type)  
**Storage**: N/A (file system only)  
**Testing**: go test, existing test infrastructure  
**Target Platform**: All platforms supported by Go (Linux, macOS, Windows)  
**Project Type**: single (Go package/library)  
**Performance Goals**: N/A (file correction only, no runtime impact)  
**Constraints**: Must maintain API compatibility, no breaking changes, files must compile with `go build`  
**Scale/Scope**: 5 mock files across 4 packages (core, prompts, memory, vectorstores)

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Package Structure Compliance
- [x] Package follows standard layout (iface/, internal/, providers/, config.go, metrics.go, errors.go, etc.) - No changes needed
- [x] Multi-provider packages implement global registry pattern - No changes needed
- [x] All required files present (test_utils.go, advanced_test.go, README.md) - No changes needed

### Design Principles Compliance  
- [x] Interfaces follow ISP (small, focused, "er" suffix for single-method) - No changes needed
- [x] Dependencies injected via constructors (DIP compliance) - No changes needed
- [x] Single responsibility per package/struct (SRP compliance) - No changes needed
- [x] Functional options used for configuration (composition over inheritance) - No changes needed

### Observability & Quality Standards
- [x] OTEL metrics implementation mandatory (no custom metrics) - No changes needed
- [x] Structured error handling with Op/Err/Code pattern - No changes needed
- [x] Comprehensive testing requirements (100% coverage, mocks, benchmarks) - No changes needed
- [x] Integration testing for cross-package interactions - No changes needed

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
pkg/
├── core/
│   └── di_mock.go              # Fix: Add package core
├── prompts/
│   └── advanced_mock.go        # Fix: Add package prompts
├── memory/
│   └── advanced_mock.go        # Fix: Add package memory
└── vectorstores/
    ├── advanced_mock.go        # Fix: Add package vectorstores
    └── iface/
        └── iface_mock.go       # Fix: Add package vectorstores
```

**Structure Decision**: Single Go package structure. The fix involves only adding package declarations to existing mock files in their respective package directories. No structural changes required.

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
- Each mock file → fix task [P] (can be done in parallel)
- Validation tasks → build/test verification
- CI validation task → add pre-commit/CI check for package declarations

**Ordering Strategy**:
- Fix tasks can run in parallel (independent files)
- Validation tasks run after all fixes complete
- CI validation task runs last (optional enhancement)

**Task Breakdown**:
1. Fix `pkg/core/di_mock.go` - Add `package core` [P]
2. Fix `pkg/prompts/advanced_mock.go` - Add `package prompts` [P]
3. Fix `pkg/memory/advanced_mock.go` - Add `package memory` [P]
4. Fix `pkg/vectorstores/advanced_mock.go` - Add `package vectorstores` [P]
5. Fix `pkg/vectorstores/iface/iface_mock.go` - Add `package vectorstores` [P]
6. Validate with `go build ./pkg/...`
7. Validate with `go test ./pkg/...`
8. Validate with `go mod verify`
9. (Optional) Add CI validation for package declarations

**Estimated Output**: 8-9 numbered, ordered tasks in tasks.md

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
- [x] Phase 3: Tasks generated (/tasks command) - tasks.md created
- [x] Phase 4: Implementation complete - All tasks implemented
- [x] Phase 5: Validation passed - Tests passing, validation complete

**Gate Status**:
- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved (N/A - straightforward fix)
- [x] Complexity deviations documented (N/A - no deviations)

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*
