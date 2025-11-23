# Tasks: Fix GitHub Workflows, Coverage, PR Checks, and Documentation Generation

**Input**: Design documents from `/specs/003-the-github-workflows/`
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
- Documentation script: `scripts/generate-docs.sh`
- Generated docs: `website/docs/api/packages/`

## Phase 3.1: Setup & Analysis
- [x] T001 Verify current workflow state - Document existing workflow files and their triggers in `.github/workflows/`
- [x] T002 [P] Backup existing workflows - Create backup copies of `.github/workflows/ci-cd.yml`, `.github/workflows/release.yml`, `.github/workflows/release_please.yml`, `.github/workflows/website_deploy.yml`
- [x] T003 Analyze coverage calculation issue - Review `.github/workflows/ci-cd.yml` lines 342-432 to identify why coverage shows 0%

## Phase 3.2: Fix Coverage Calculation
**CRITICAL: Fix coverage calculation first as it blocks other validations**
- [x] T004 Fix coverage file generation - Ensure `coverage.unit.out` is always created in `.github/workflows/ci-cd.yml` unit-tests job, even on partial test failures
- [x] T005 Fix coverage parsing logic - Update coverage parsing in `.github/workflows/ci-cd.yml` Parse test results step (line 376-384) to validate file exists before parsing
- [x] T006 Fix coverage threshold check - Ensure coverage threshold check in `.github/workflows/ci-cd.yml` (line 422-431) properly fails when coverage < 80% and provides clear error message
- [x] T007 [P] Add coverage validation contract test - Create validation script in `scripts/validate-coverage.sh` to test coverage calculation per Contract 1

## Phase 3.3: Consolidate Release Workflows
- [x] T008 Merge release workflows - Consolidate `.github/workflows/release.yml` and `.github/workflows/release_please.yml` into single unified `.github/workflows/release.yml` supporting automated, manual, and tag triggers
- [x] T009 Add release conflict prevention - Implement conditional job execution in `.github/workflows/release.yml` to prevent simultaneous releases
- [x] T010 [P] Add release validation contract test - Create validation script in `scripts/validate-release.sh` to test release workflow per Contract 2
- [x] T011 Remove old release_please.yml - Delete `.github/workflows/release_please.yml` after consolidation is complete and tested

## Phase 3.4: Fix PR Check Configuration
- [x] T012 Configure critical vs advisory checks - Update `.github/workflows/ci-cd.yml` to distinguish critical checks (unit-tests, security, build) from advisory checks (lint, coverage threshold)
- [x] T013 Set check conclusions appropriately - Use GitHub Actions check conclusion API in `.github/workflows/ci-cd.yml` to set `failure` for critical checks and `neutral`/`warning` for advisory checks
- [x] T014 [P] Add PR check validation contract test - Create validation script in `scripts/validate-pr-checks.sh` to test PR check status per Contract 3

## Phase 3.5: Integrate Documentation Generation
- [x] T015 Add documentation generation to website_deploy - Integrate `make docs-generate` or `./scripts/generate-docs.sh` into `.github/workflows/website_deploy.yml` before website build step
- [x] T016 Ensure gomarkdoc installation - Add gomarkdoc installation step in `.github/workflows/website_deploy.yml` if not already present
- [x] T017 Configure documentation failure handling - Ensure `.github/workflows/website_deploy.yml` fails workflow if documentation generation fails
- [x] T018 [P] Add documentation validation contract test - Create validation script in `scripts/validate-docs.sh` to test documentation generation per Contract 4

## Phase 3.6: Workflow File Validation
- [x] T019 [P] Validate all workflow YAML syntax - Create validation script in `scripts/validate-workflows.sh` to check all `.github/workflows/*.yml` files for valid YAML per Contract 5
- [x] T020 Add workflow comments - Add clear comments to all workflow files explaining triggers, purposes, and job dependencies

## Phase 3.7: End-to-End Validation
- [x] T021 Validate coverage calculation fix - Run quickstart validation step 1 to verify coverage shows accurate percentage (not 0%)
- [x] T022 Validate release workflow consolidation - Run quickstart validation step 2 to verify unified release workflow supports all trigger types
- [x] T023 Validate PR check configuration - Run quickstart validation step 3 to verify PR checks accurately reflect status
- [x] T024 Validate documentation generation - Run quickstart validation step 4 to verify documentation generates on main merges
- [x] T025 Validate workflow files - Run quickstart validation step 5 to verify all workflow files are valid YAML
- [x] T026 Create test PR - Create a test PR to validate all fixes work together end-to-end per quickstart step 6

## Dependencies
- T001-T003 (Setup) before all other tasks
- T004-T007 (Coverage fixes) can run in parallel with T008-T011 (Release consolidation) but must complete before T012-T014 (PR checks)
- T008-T011 (Release consolidation) are sequential within the group
- T012-T014 (PR checks) depend on T004-T007 (Coverage fixes) completing
- T015-T018 (Documentation) can run in parallel with other fixes (different workflow file)
- T019-T020 (Validation) can run in parallel
- T021-T026 (End-to-end) depend on all previous fixes completing

## Parallel Execution Examples

### Example 1: Initial Setup (can run in parallel)
```
Task: "Backup existing workflows - Create backup copies of .github/workflows/ci-cd.yml, .github/workflows/release.yml, .github/workflows/release_please.yml, .github/workflows/website_deploy.yml"
Task: "Analyze coverage calculation issue - Review .github/workflows/ci-cd.yml lines 342-432"
```

### Example 2: Coverage and Release Fixes (can run in parallel - different files)
```
Task: "Fix coverage file generation - Ensure coverage.unit.out is always created in .github/workflows/ci-cd.yml"
Task: "Merge release workflows - Consolidate .github/workflows/release.yml and .github/workflows/release_please.yml"
Task: "Add documentation generation to website_deploy - Integrate make docs-generate into .github/workflows/website_deploy.yml"
```

### Example 3: Validation Scripts (can run in parallel - different files)
```
Task: "Add coverage validation contract test - Create validation script in scripts/validate-coverage.sh"
Task: "Add release validation contract test - Create validation script in scripts/validate-release.sh"
Task: "Add PR check validation contract test - Create validation script in scripts/validate-pr-checks.sh"
Task: "Add documentation validation contract test - Create validation script in scripts/validate-docs.sh"
Task: "Validate all workflow YAML syntax - Create validation script in scripts/validate-workflows.sh"
```

## Notes
- [P] tasks = different files, no dependencies
- Coverage fixes (T004-T007) should be completed first as they block PR check validation
- Release consolidation (T008-T011) is independent but should be tested before removing old workflow
- Documentation integration (T015-T018) is independent and can be done in parallel
- All validation tasks (T007, T010, T014, T018, T019) can be created in parallel
- End-to-end validation (T021-T026) must wait for all fixes to complete
- Commit after each major fix (coverage, release, PR checks, documentation)
- Test each fix individually before moving to next

## Task Generation Rules
*Applied during main() execution*

1. **From Contracts**:
   - Each contract file → validation script task [P]
   - Contract 1 (Coverage) → T007
   - Contract 2 (Release) → T010
   - Contract 3 (PR Checks) → T014
   - Contract 4 (Documentation) → T018
   - Contract 5 (Workflow Files) → T019

2. **From Data Model**:
   - Workflow Configuration → T008-T009 (release workflow consolidation)
   - Test Coverage Report → T004-T007 (coverage fixes)
   - PR Check Status → T012-T014 (PR check configuration)
   - Documentation Artifact → T015-T018 (documentation integration)

3. **From Research**:
   - Coverage calculation issue → T004-T007
   - Release workflow consolidation → T008-T011
   - PR check failure handling → T012-T014
   - Documentation generation → T015-T018
   - Coverage threshold enforcement → T006

4. **From Quickstart**:
   - Each validation scenario → T021-T026 (end-to-end validation)

5. **Ordering**:
   - Setup → Coverage Fixes → Release Consolidation → PR Checks → Documentation → Validation → End-to-End

## Validation Checklist
*GATE: Checked by main() before returning*

### Task Quality
- [x] All contracts have corresponding validation scripts
- [x] All research findings have implementation tasks
- [x] All quickstart scenarios have validation tasks
- [x] Parallel tasks truly independent (different files)
- [x] Each task specifies exact file path
- [x] No task modifies same file as another [P] task
- [x] Dependencies clearly documented

### Coverage
- [x] Coverage calculation fix tasks (T004-T007)
- [x] Release workflow consolidation tasks (T008-T011)
- [x] PR check configuration tasks (T012-T014)
- [x] Documentation integration tasks (T015-T018)
- [x] Validation tasks for all contracts (T007, T010, T014, T018, T019)
- [x] End-to-end validation tasks (T021-T026)

---
*Based on Implementation Plan - See `plan.md`*

