# Implementation Plan: Adjust GitHub Workflows and Pipelines with Manual Triggers

**Branch**: `005-adjust-the-workflows` | **Date**: 2025-01-27 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/005-adjust-the-workflows/spec.md`

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
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file
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
Adjust GitHub Actions workflows to properly implement CI/CD pipeline steps with correct criticality levels (critical vs advisory), ensure lint/format auto-fix with automatic commits, enforce test coverage thresholds with warnings (not blocking), require security checks to pass (only critical vulnerabilities block), configure release pipeline with GoReleaser, changelog generation, documentation generation, and website updates. Add manual trigger support (`workflow_dispatch`) for all workflow steps to enable flexible execution. Implement common Go library release patterns including changelog generation (changie or git-based), semantic versioning, and automated release notes. Use existing repository tools (scripts, Makefile) and validate workflows using `gh` CLI and validation scripts.

## Technical Context
**Language/Version**: Go 1.24, YAML (GitHub Actions workflows), Bash (validation scripts)  
**Primary Dependencies**: 
- GitHub Actions (ci-cd.yml, release.yml, website_deploy.yml)
- GoReleaser (for release generation)
- Changie (optional, for changelog management) - [github.com/miniscruff/changie](https://github.com/miniscruff/changie)
- golangci-lint (for linting with --fix support)
- gosec, govulncheck, gitleaks, Trivy (for security scanning)
- gomarkdoc (for API documentation generation)
- Docusaurus (for website deployment)
- `gh` CLI (for workflow testing and validation)
  
**Storage**: N/A (workflow configuration files only)  
**Testing**: 
- Workflow validation via `gh workflow view` and `gh workflow run`
- Validation scripts: `scripts/validate-workflows.sh`, `scripts/validate-pr-checks.sh`, `scripts/validate-coverage.sh`, `scripts/validate-release.sh`
- Makefile targets: `make ci-local`, `make lint-fix`, `make test-coverage-threshold`
  
**Target Platform**: GitHub Actions (Ubuntu runners)  
**Project Type**: single (Go framework with GitHub workflows)  
**Performance Goals**: 
- Workflow execution time: <15 minutes for full CI/CD pipeline
- Release pipeline: <10 minutes for documentation generation and release creation
  
**Constraints**: 
- Must maintain backward compatibility with existing workflow triggers
- Must use existing repository tools (scripts, Makefile) where possible
- Must support both automated (release-please) and manual releases
- Must support manual triggering of individual workflow steps
- Coverage threshold: 80% (warnings below, not blocking)
- Manual releases take precedence over automated releases
  
**Scale/Scope**: 
- 3 workflow files (ci-cd.yml, release.yml, website_deploy.yml)
- ~10 validation scripts
- Multiple Makefile targets for local testing
- Manual trigger inputs for 7+ major workflow steps

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Note**: This feature involves GitHub Actions workflow configuration files, not Go package code. Constitution principles apply indirectly through ensuring workflows validate Go code that follows constitutional standards.

### Package Structure Compliance
- [x] N/A - This is workflow infrastructure, not a Go package
- [x] Workflows validate that Go packages follow standard layout (indirect compliance)
- [x] Workflows use existing repository structure and tools

### Design Principles Compliance  
- [x] Workflows enforce SRP: Each job has single responsibility (lint, test, security, build)
- [x] Workflows use composition: Jobs can run in parallel where independent
- [x] Workflows follow DIP: Depend on abstractions (Makefile targets, scripts) not implementations
- [x] Workflows validate ISP compliance in Go code through linting and testing

### Observability & Quality Standards
- [x] Workflows generate test coverage reports and upload artifacts
- [x] Workflows validate error handling through security scans and tests
- [x] Workflows enforce testing requirements: Unit tests, integration tests, coverage thresholds
- [x] Workflows support integration testing through dedicated integration test job

**Constitutional Alignment**: Workflows serve as quality gates ensuring all Go packages comply with constitutional standards. The workflow structure itself follows SRP (single job per concern) and uses composition (parallel jobs, dependency chains, manual trigger inputs).

*Reference: Constitution v1.0.0 - See `.specify/memory/constitution.md`*

## Project Structure

### Documentation (this feature)
```
specs/005-adjust-the-workflows/
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
├── ci-cd.yml              # Main CI/CD workflow (PR checks, tests, security, build) + manual triggers
├── release.yml             # Release workflow (GoReleaser, changelog, docs, tagging) + manual triggers
├── website_deploy.yml      # Website deployment workflow (docs generation, Docusaurus) + manual triggers
└── backup/                 # Backup copies of workflows

scripts/
├── validate-workflows.sh   # Workflow file validation
├── validate-pr-checks.sh    # PR check configuration validation
├── validate-coverage.sh    # Coverage calculation validation
├── validate-release.sh     # Release workflow validation
├── validate-docs.sh         # Documentation generation validation
├── generate-docs.sh         # API documentation generation
└── [other validation scripts]

Makefile                    # Build, test, lint targets (used by workflows)

.goreleaser.yml             # GoReleaser configuration for releases
.changie.yaml               # Changie configuration (optional, for changelog management)
release-please-config.json  # Release-please configuration (automated versioning)

website/
├── docs/api/packages/      # Generated API documentation (output)
└── [Docusaurus structure]
```

**Structure Decision**: Single project structure with GitHub Actions workflows in `.github/workflows/`, validation scripts in `scripts/`, and Makefile targets for local testing. Workflows reference existing repository tools rather than duplicating logic. Manual triggers added via `workflow_dispatch` with input parameters. Changelog management via changie (optional) or git-based approach. This follows the existing repository structure established in previous workflow implementations (spec 003) and common patterns from Go library projects like changie and langchaingo.

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

**Status**: ✅ Complete - research.md includes:
- Auto-fix implementation
- Coverage threshold warnings
- Security check behavior
- Workflow testing with gh CLI
- Release pipeline integration
- Manual trigger implementation (workflow_dispatch)
- Release automation patterns (changie, GoReleaser)
- Changelog generation integration

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - Entity name, fields, relationships
   - Validation rules from requirements
   - State transitions if applicable
   - **NEW**: Manual trigger input entities

2. **Generate API contracts** from functional requirements:
   - For each user action → endpoint
   - Use standard REST/GraphQL patterns
   - Output OpenAPI/GraphQL schema to `/contracts/`
   - **NEW**: Contracts for manual trigger inputs and step execution

3. **Generate contract tests** from contracts:
   - One test file per endpoint
   - Assert request/response schemas
   - Tests must fail (no implementation yet)

4. **Extract test scenarios** from user stories:
   - Each story → integration test scenario
   - Quickstart test = story validation steps
   - **NEW**: Manual trigger test scenarios

5. **Update agent file incrementally** (O(1) operation):
   - Run `.specify/scripts/bash/update-agent-context.sh cursor`
     **IMPORTANT**: Execute it exactly as specified above. Do not add or remove any arguments.
   - If exists: Add only NEW tech from current plan
   - Preserve manual additions between markers
   - Update recent changes (keep last 3)
   - Keep under 150 lines for token efficiency
   - Output to repository root

**Output**: data-model.md, /contracts/*, failing tests, quickstart.md, agent-specific file

**Status**: ✅ Complete - All Phase 1 artifacts generated (data-model.md, contracts/, quickstart.md)

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
- Load `.specify/templates/tasks-template.md` as base
- Generate tasks from Phase 1 design docs (contracts, data model, quickstart)
- Each contract (C001-C006) → workflow adjustment task
- **NEW**: Manual trigger contract → workflow_dispatch implementation tasks
- **NEW**: Changelog generation contract → changie/git-based changelog tasks
- Each validation script → enhancement/verification task [P]
- Each workflow file → modification task
- Testing tasks for workflow validation

**Ordering Strategy**:
- Setup: Backup existing workflows, validate current state
- Core adjustments: Fix CI/CD workflow (policy, lint, tests, security, build)
- **NEW**: Manual triggers: Add workflow_dispatch to all workflows with input parameters
- **NEW**: Changelog integration: Add changelog generation to release workflow
- Release pipeline: Enhance release workflow (GoReleaser, docs, website, changelog)
- Validation: Update and test validation scripts
- Testing: Test workflows with `gh` CLI and validation scripts
- Mark [P] for parallel execution (different workflow files, independent scripts)

**Task Categories**:
1. **Setup & Analysis** (T001-T003): Backup, analyze current state, identify issues
2. **CI/CD Workflow Adjustments** (T004-T010): Policy checks, lint auto-fix, test coverage, security, build
3. **Manual Trigger Implementation** (T011-T015): Add workflow_dispatch to all workflows, implement input parameters, test manual triggers
4. **Changelog & Release Patterns** (T016-T020): Integrate changie (optional), git-based changelog, release notes generation
5. **Release Pipeline Enhancements** (T021-T025): GoReleaser, documentation, website update, changelog integration
6. **Validation Script Updates** (T026-T030): Enhance validation scripts, add manual trigger checks, add changelog validation
7. **Testing & Verification** (T031-T035): Test workflows, verify manual triggers, verify release patterns, document

**Estimated Output**: 35-40 numbered, ordered tasks in tasks.md

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

No violations - workflows are infrastructure, not Go packages. Workflow structure follows SRP and composition principles.

## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command) - research.md created and updated with manual triggers and release patterns
- [x] Phase 1: Design complete (/plan command) - data-model.md, contracts/, quickstart.md created
- [x] Phase 2: Task planning complete (/plan command - describe approach only) - approach documented with new task categories
- [x] Phase 3: Tasks generated (/tasks command) - tasks.md created
- [x] Phase 4: Implementation complete - All tasks implemented
- [x] Phase 5: Validation passed - Workflows validated and working

**Gate Status**:
- [x] Initial Constitution Check: PASS - Workflows serve as quality gates, follow SRP
- [x] Post-Design Constitution Check: PASS - Design aligns with constitutional principles
- [x] All NEEDS CLARIFICATION resolved - All research questions answered in research.md
- [x] Complexity deviations documented - N/A (no deviations, workflows are infrastructure)

---
*Based on Constitution v1.0.0 - See `.specify/memory/constitution.md`*
