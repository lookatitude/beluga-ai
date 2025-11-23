
# Implementation Plan: Fix GitHub Workflows, Coverage, PR Checks, and Documentation Generation

**Branch**: `003-the-github-workflows` | **Date**: 2025-01-27 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-the-github-workflows/spec.md`

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
Fix GitHub Actions workflows to resolve multiple issues: consolidate conflicting release workflows into a single unified workflow, fix test coverage calculation showing 0% when tests pass, ensure PR checks pass correctly, and enable automatic documentation generation on main branch merges. The fix involves workflow consolidation, coverage calculation debugging, PR check configuration, and documentation workflow integration.

## Technical Context
**Language/Version**: Go 1.24.0  
**Primary Dependencies**: GitHub Actions, gomarkdoc (for API docs), GoReleaser (for releases), release-please (for semantic versioning)  
**Storage**: N/A (workflow configuration files only)  
**Testing**: go test, GitHub Actions workflow validation  
**Target Platform**: GitHub Actions (Linux runners)  
**Project Type**: single (Go library/framework)  
**Performance Goals**: Workflows complete within reasonable time (<15min for full CI, <5min for PR checks)  
**Constraints**: Must maintain backward compatibility with existing workflow triggers, cannot break current CI/CD processes  
**Scale/Scope**: 4 workflow files, ~700 lines of YAML, 14+ packages requiring documentation generation

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Package Structure Compliance
- [x] Package follows standard layout - N/A (workflow files, not Go packages)
- [x] Multi-provider packages implement global registry pattern - N/A (workflow files)
- [x] All required files present - N/A (workflow files)

### Design Principles Compliance  
- [x] Interfaces follow ISP - N/A (workflow files, not Go code)
- [x] Dependencies injected via constructors - N/A (workflow files)
- [x] Single responsibility per package/struct - Applied: Each workflow has single responsibility (CI, release, docs)
- [x] Functional options used for configuration - N/A (YAML configuration)

### Observability & Quality Standards
- [x] OTEL metrics implementation mandatory - N/A (workflow files)
- [x] Structured error handling with Op/Err/Code pattern - N/A (workflow files)
- [x] Comprehensive testing requirements - Applied: Workflow validation, coverage threshold enforcement (80%)
- [x] Integration testing for cross-package interactions - N/A (workflow files)

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
.github/workflows/
├── ci-cd.yml              # Main CI/CD workflow (needs coverage fix)
├── release.yml             # Manual/tag-based releases (to be merged)
├── release_please.yml      # Automated semantic versioning (to be merged)
└── website_deploy.yml      # Documentation deployment (needs doc generation integration)

scripts/
└── generate-docs.sh        # API documentation generation script

website/
├── docs/api/packages/      # Generated API documentation output
└── build/                  # Docusaurus build output
```

**Structure Decision**: Single Go project with GitHub Actions workflows. Workflow files are in `.github/workflows/`, documentation generation script in `scripts/`, and generated docs in `website/docs/api/packages/`. No structural changes needed, only workflow file modifications.

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
- Each validation contract → workflow fix task
- Coverage calculation fix → debugging and fix tasks
- Release workflow consolidation → merge and test tasks
- PR check configuration → update and validation tasks
- Documentation generation → integration and test tasks
- Quickstart validation → end-to-end test tasks

**Ordering Strategy**:
- Fix coverage calculation first (blocks other validations)
- Consolidate release workflows (independent of other fixes)
- Configure PR checks (depends on coverage fix)
- Integrate documentation generation (independent)
- End-to-end validation (depends on all fixes)
- Mark [P] for parallel execution (independent workflow files)

**Estimated Output**: 15-20 numbered, ordered tasks in tasks.md covering:
- Coverage calculation debugging and fix
- Release workflow consolidation
- PR check configuration
- Documentation generation integration
- Validation and testing

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
- [x] Initial Constitution Check: PASS (N/A for workflow files)
- [x] Post-Design Constitution Check: PASS (N/A for workflow files)
- [x] All NEEDS CLARIFICATION resolved (5 clarifications completed)
- [x] Complexity deviations documented (none - workflows are standard)

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*
