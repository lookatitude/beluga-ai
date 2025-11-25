# Tasks: Adjust GitHub Workflows and Pipelines with Manual Triggers

**Input**: Design documents from `/specs/005-adjust-the-workflows/`
**Prerequisites**: plan.md, research.md, data-model.md, contracts/, quickstart.md

## Execution Flow (main)
```
1. Load plan.md from feature directory
   → Extract: tech stack, libraries, structure
2. Load optional design documents:
   → data-model.md: Extract entities → workflow fix tasks
   → contracts/: Each contract → validation task
   → research.md: Extract decisions → implementation tasks
3. Generate tasks by category:
   → Setup: backup, verify current state
   → Fixes: coverage, release, PR checks, documentation
   → Manual triggers: workflow_dispatch implementation
   → Changelog: changie/git-based integration
   → Validation: contract tests, end-to-end tests
4. Apply task rules:
   → Different workflow files = mark [P] for parallel
   → Same file = sequential (no [P])
   → Validation before fixes (where applicable)
5. Number tasks sequentially (T001, T002...)
6. Generate dependency graph
7. Create parallel execution examples
8. Validate task completeness
9. Return: SUCCESS (tasks ready for execution)
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions
- Workflow files: `.github/workflows/`
- Validation scripts: `scripts/`
- Configuration files: Repository root (`.goreleaser.yml`, `.changie.yaml`)

## Phase 3.1: Setup & Analysis
- [X] T001 Verify current workflow state - Document existing workflow files and their triggers in `.github/workflows/` (ci-cd.yml, release.yml, website_deploy.yml)
- [X] T002 [P] Backup existing workflows - Create backup copies of `.github/workflows/ci-cd.yml`, `.github/workflows/release.yml`, `.github/workflows/website_deploy.yml` to `.github/workflows/backup/`
- [X] T003 Analyze current workflow configuration - Review `.github/workflows/ci-cd.yml` to identify current job configurations, criticality levels, and missing features

## Phase 3.2: CI/CD Workflow Adjustments (Contract C001-C004)
**CRITICAL: Fix CI/CD workflow first as it's the main pipeline**

- [X] T004 [P] Adjust policy checks job (C001) - Update `.github/workflows/ci-cd.yml` policy job to have `continue-on-error: true` and use `::warning::` annotations instead of `::error::`
- [X] T005 [P] Enable lint auto-fix with commits (C002) - Update `.github/workflows/ci-cd.yml` lint job to use `--fix` flag for golangci-lint, run `gofmt -w`, and automatically commit fixed files back to PR branch
- [X] T006 [P] Fix coverage threshold warnings (C003) - Update `.github/workflows/ci-cd.yml` unit-tests and integration-tests jobs to emit `::warning::` (not `::error::`) when coverage < 80% and ensure coverage calculation failure generates warning but continues pipeline
- [X] T007 [P] Verify security checks are critical (C004) - Ensure `.github/workflows/ci-cd.yml` security job does NOT have `continue-on-error: true` and all security tools (gosec, govulncheck, gitleaks, Trivy) exit with error on failure
- [X] T008 Update build job dependencies - Ensure `.github/workflows/ci-cd.yml` build job depends on critical jobs (security, unit-tests) but not advisory jobs (policy, lint)

## Phase 3.3: Manual Trigger Implementation (Contract C007)
- [X] T009 [P] Add workflow_dispatch to CI/CD workflow - Add `workflow_dispatch` event with input parameters (`run_policy`, `run_lint`, `run_security`, `run_unit_tests`, `run_integration_tests`, `run_build`) to `.github/workflows/ci-cd.yml`
- [X] T010 [P] Add conditional logic to CI/CD jobs - Update all jobs in `.github/workflows/ci-cd.yml` to use conditional `if:` statements based on input parameters (execute if input is true OR if not workflow_dispatch)
- [X] T011 [P] Add workflow_dispatch to release workflow - Add `workflow_dispatch` event with input parameters (`run_pre_release`, `run_release`, `run_docs`, `run_website`) to `.github/workflows/release.yml`
- [X] T012 [P] Add conditional logic to release jobs - Update jobs in `.github/workflows/release.yml` to use conditional `if:` statements based on input parameters
- [X] T013 [P] Add workflow_dispatch to website deploy workflow - Add `workflow_dispatch` event to `.github/workflows/website_deploy.yml` for manual website deployment

## Phase 3.4: Changelog Generation Integration (Contract C008)
- [X] T014 [P] Create changie configuration (optional) - Create `.changie.yaml` configuration file for changelog management following changie patterns, or document git-based approach
- [X] T015 [P] Add changelog generation step to release workflow - Add changelog generation step before GoReleaser in `.github/workflows/release.yml` that supports changie (if configured) or git-based changelog
- [X] T016 [P] Integrate changelog with GoReleaser - Update `.goreleaser.yml` to use generated changelog for release notes, ensuring changelog is included in GitHub release description
- [X] T017 [P] Handle changelog generation failures - Ensure changelog generation failure marks release as incomplete but allows release to continue in `.github/workflows/release.yml`

## Phase 3.5: Release Pipeline Enhancements (Contract C005)
- [X] T018 [P] Verify GoReleaser configuration - Ensure `.goreleaser.yml` is properly configured and `.github/workflows/release.yml` uses `goreleaser/goreleaser-action@v6`
- [X] T019 [P] Add API documentation generation to release - Ensure `.github/workflows/release.yml` runs `scripts/generate-docs.sh` or `make docs-generate` before website update
- [X] T020 [P] Add website update to release workflow - Ensure `.github/workflows/release.yml` builds and deploys Docusaurus website with generated docs, or triggers website_deploy workflow
- [X] T021 [P] Verify release concurrency control - Ensure `.github/workflows/release.yml` has concurrency group configured and manual releases take precedence (cancel automated releases)
- [X] T022 [P] Update release workflow for incomplete releases - Ensure documentation and website update failures mark release as incomplete but allow release to continue

## Phase 3.6: Validation Script Updates (Contract C006)
- [X] T023 [P] Enhance validate-workflows.sh - Update `scripts/validate-workflows.sh` to check for workflow_dispatch, input parameters, and conditional logic
- [X] T024 [P] Enhance validate-pr-checks.sh - Update `scripts/validate-pr-checks.sh` to verify policy and lint jobs have continue-on-error, security and test jobs don't
- [X] T025 [P] Enhance validate-coverage.sh - Update `scripts/validate-coverage.sh` to verify coverage threshold uses warnings (not errors) and doesn't exit with error code
- [X] T026 [P] Enhance validate-release.sh - Update `scripts/validate-release.sh` to check for changelog generation, GoReleaser usage, and manual trigger support
- [X] T027 [P] Create validate-manual-triggers.sh - Create new script `scripts/validate-manual-triggers.sh` to validate workflow_dispatch configuration and input parameters in all workflows
- [X] T028 [P] Create validate-changelog.sh - Create new script `scripts/validate-changelog.sh` to validate changelog generation configuration and integration with GoReleaser

## Phase 3.7: Testing & Verification
- [X] T029 [P] Test workflow YAML syntax - Run `scripts/validate-workflows.sh` and verify all workflow files pass YAML validation
- [X] T030 [P] Test PR check configuration - Run `scripts/validate-pr-checks.sh` and verify critical vs advisory jobs are correctly configured
- [X] T031 [P] Test coverage configuration - Run `scripts/validate-coverage.sh` and verify coverage threshold warnings work correctly
- [X] T032 [P] Test release workflow configuration - Run `scripts/validate-release.sh` and verify release pipeline includes all required steps
- [X] T033 [P] Test manual triggers via gh CLI - Use `gh workflow view` and `gh workflow run` to test manual triggering of workflows with different input combinations
- [X] T034 [P] Test changelog generation - Test changelog generation locally and verify integration with GoReleaser
- [ ] T035 Integration test: Full CI/CD pipeline - Create test PR and verify all workflow steps execute correctly with proper criticality levels
- [ ] T036 Integration test: Manual trigger execution - Test manual triggering of individual steps via GitHub UI and verify only selected steps execute
- [ ] T037 Integration test: Release pipeline - Test release workflow with manual trigger and verify changelog, docs, website, and release publication work correctly
- [X] T038 [P] Update quickstart documentation - Update `specs/005-adjust-the-workflows/quickstart.md` with manual trigger examples and changelog testing steps
- [X] T039 [P] Create workflow usage documentation - Create or update documentation explaining manual trigger inputs, changelog generation, and release process

## Dependencies
- T001-T003 (Setup) must complete before workflow modifications
- T004-T008 (CI/CD adjustments) can run in parallel (different jobs in same file, but sequential to avoid conflicts)
- T009-T013 (Manual triggers) can run in parallel (different workflow files)
- T014-T017 (Changelog) can run in parallel (different files)
- T018-T022 (Release pipeline) can run in parallel (different aspects)
- T023-T028 (Validation scripts) can run in parallel (different scripts)
- T029-T037 (Testing) should run after all implementation tasks
- T038-T039 (Documentation) can run in parallel with testing

## Parallel Execution Examples

### Example 1: CI/CD Workflow Adjustments (T004-T007)
These tasks modify different jobs in the same file, so they should be done sequentially to avoid merge conflicts:
```
Task: "Adjust policy checks job (C001) in .github/workflows/ci-cd.yml"
Task: "Enable lint auto-fix with commits (C002) in .github/workflows/ci-cd.yml"
Task: "Fix coverage threshold warnings (C003) in .github/workflows/ci-cd.yml"
Task: "Verify security checks are critical (C004) in .github/workflows/ci-cd.yml"
```
**Note**: While these are marked [P] for different contracts, they modify the same file and should be done carefully to avoid conflicts.

### Example 2: Manual Triggers Across Workflows (T009-T013)
These tasks modify different workflow files and can run in parallel:
```
Task: "Add workflow_dispatch to CI/CD workflow in .github/workflows/ci-cd.yml"
Task: "Add workflow_dispatch to release workflow in .github/workflows/release.yml"
Task: "Add workflow_dispatch to website deploy workflow in .github/workflows/website_deploy.yml"
```

### Example 3: Validation Script Updates (T023-T028)
These tasks modify different script files and can run in parallel:
```
Task: "Enhance validate-workflows.sh in scripts/validate-workflows.sh"
Task: "Enhance validate-pr-checks.sh in scripts/validate-pr-checks.sh"
Task: "Enhance validate-coverage.sh in scripts/validate-coverage.sh"
Task: "Create validate-manual-triggers.sh in scripts/validate-manual-triggers.sh"
Task: "Create validate-changelog.sh in scripts/validate-changelog.sh"
```

### Example 4: Testing Tasks (T029-T034)
These tasks run validation scripts and can run in parallel:
```
Task: "Test workflow YAML syntax using scripts/validate-workflows.sh"
Task: "Test PR check configuration using scripts/validate-pr-checks.sh"
Task: "Test coverage configuration using scripts/validate-coverage.sh"
Task: "Test release workflow configuration using scripts/validate-release.sh"
```

## Notes
- [P] tasks = different files, no dependencies (but be careful with same-file modifications)
- Workflow file modifications (ci-cd.yml) should be done carefully even if marked [P] - consider sequential execution
- Validation scripts can be updated in parallel as they're independent files
- Test all changes locally using `make ci-local` before committing
- Use `gh workflow run` to test manual triggers
- Commit after each major task completion
- Avoid: modifying same workflow section simultaneously, breaking existing triggers

## Task Generation Rules
*Applied during main() execution*

1. **From Contracts**:
   - C001 (Policy Checks) → T004
   - C002 (Lint/Format) → T005
   - C003 (Test Coverage) → T006
   - C004 (Security) → T007
   - C005 (Release Pipeline) → T018-T022
   - C006 (Workflow Validation) → T023-T028
   - C007 (Manual Triggers) → T009-T013
   - C008 (Changelog) → T014-T017

2. **From Data Model**:
   - WorkflowConfiguration entity → Manual trigger implementation tasks
   - WorkflowInput entity → Input parameter tasks
   - PipelineJob entity → Job adjustment tasks
   - ReleaseArtifact entity → Release pipeline tasks

3. **From Research**:
   - Auto-fix with commits → T005
   - Coverage warnings → T006
   - Manual triggers → T009-T013
   - Changelog patterns → T014-T017

4. **Ordering**:
   - Setup → Workflow Adjustments → Manual Triggers → Changelog → Release → Validation → Testing
   - Dependencies block parallel execution
   - Same file modifications should be sequential

## Validation Checklist
*GATE: Checked by main() before returning*

### Task Quality
- [x] All contracts have corresponding implementation tasks
- [x] All workflow files have modification tasks
- [x] All validation scripts have update tasks
- [x] Manual trigger tasks cover all workflows
- [x] Changelog tasks cover generation and integration
- [x] Testing tasks cover all validation scenarios
- [x] Each task specifies exact file path
- [x] Parallel tasks are truly independent (with notes about same-file caution)

### Completeness
- [x] Setup tasks (T001-T003)
- [x] CI/CD workflow adjustments (T004-T008)
- [x] Manual trigger implementation (T009-T013)
- [x] Changelog generation (T014-T017)
- [x] Release pipeline enhancements (T018-T022)
- [x] Validation script updates (T023-T028)
- [x] Testing and verification (T029-T039)

---
*Based on Constitution v1.0.0 - See `.specify/memory/constitution.md`*

