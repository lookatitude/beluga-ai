---
description: Execute the implementation plan by processing and executing all tasks defined in tasks.md
---

The user input can be provided directly by the agent or as a command argument - you **MUST** consider it before proceeding with the prompt (if not empty).

User input:

$ARGUMENTS

1. Run `.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks` from repo root and parse FEATURE_DIR and AVAILABLE_DOCS list. All paths must be absolute.

2. Load and analyze the implementation context:
   - **REQUIRED**: Read tasks.md for the complete task list and execution plan
   - **CRITICAL**: Identify task type from tasks.md header (NEW FEATURE, ANALYSIS, or CORRECTION)
   - **REQUIRED**: Read plan.md for tech stack, architecture, and file structure
   - **IF EXISTS**: Read data-model.md for entities and relationships
   - **IF EXISTS**: Read contracts/ for API specifications and test requirements
   - **IF EXISTS**: Read research.md for technical decisions and constraints
   - **IF EXISTS**: Read quickstart.md for integration scenarios

3. Parse tasks.md structure and extract:
   - **Task type classification**: NEW FEATURE (create code) vs ANALYSIS (document) vs CORRECTION (fix code)
   - **File targets**: Verify tasks target correct directories based on type
     * NEW FEATURE/CORRECTION: Tasks MUST modify `pkg/`, `cmd/`, `internal/` files
     * ANALYSIS: Tasks MUST write to `specs/` directory only
   - **Task phases**: Setup, Tests/Verification, Core/Analysis, Integration/Validation, Polish/Reporting
   - **Task dependencies**: Sequential vs parallel execution rules
   - **Task details**: ID, description, EXACT file paths, parallel markers [P]
   - **Execution flow**: Order and dependency requirements

4. Execute implementation following the task plan BY TYPE:
   
   **For NEW FEATURES** (create code in `pkg/`):
   - Phase-by-phase: Setup → Tests → Core → Integration → Polish
   - Respect TDD: Write failing tests BEFORE implementation
   - Constitutional compliance: Ensure config.go, metrics.go, errors.go, test_utils.go
   - Validation: Run `go test ./pkg/{package}/... -v` after each phase
   
   **For ANALYSIS** (document in `specs/`):
   - Phase-by-phase: Setup → Verification → Analysis → Validation → Reporting
   - Read-only: NEVER modify `pkg/` files during analysis
   - Document findings: Write all results to `specs/NNN-for-the-{package}/`
   - Validation: Ensure comprehensive documentation of current state
   
   **For CORRECTIONS** (fix code in `pkg/`):
   - Phase-by-phase: Setup → Tests → Fixes → Verification → Documentation
   - Test first: Add missing tests to verify issue exists
   - Fix implementation: Modify actual code in `pkg/` to pass tests
   - Validation: Run full test suite to check for regressions

5. Implementation execution rules:
   - **Verify task type first**: Confirm file targets match task type classification
   - **Setup first**: Initialize project structure, dependencies, configuration
   - **Tests before code**: If you need to write tests for contracts, entities, and integration scenarios
   - **Core development**: Implement models, services, CLI commands, endpoints
   - **Integration work**: Database connections, middleware, logging, external services
   - **Polish and validation**: Unit tests, performance optimization, documentation

6. Progress tracking and error handling:
   - Report progress after each completed task
   - Halt execution if any non-parallel task fails
   - For parallel tasks [P], continue with successful tasks, report failed ones
   - Provide clear error messages with context for debugging
   - Suggest next steps if implementation cannot proceed
   - **IMPORTANT** For completed tasks, make sure to mark the task off as [X] in the tasks file.

7. Completion validation:
   - Verify all required tasks are completed
   - Check that implemented features match the original specification
   - Validate that tests pass and coverage meets requirements
   - Confirm the implementation follows the technical plan
   - Report final status with summary of completed work

Note: This command assumes a complete task breakdown exists in tasks.md. If tasks are incomplete or missing, suggest running `/tasks` first to regenerate the task list.
