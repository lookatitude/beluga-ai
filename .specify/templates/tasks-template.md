# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`
**Prerequisites**: plan.md (required), research.md, data-model.md, contracts/

## Task Type Classification (CRITICAL - Read First!)

**IDENTIFY THE TASK TYPE** before generating tasks:

### 1️⃣ NEW FEATURE IMPLEMENTATION (`specs/NNN-feature-name/`)
- **Goal**: Create NEW code in `pkg/`, `cmd/`, `internal/` directories
- **File Targets**: All tasks write to actual codebase files (`.go`, `.py`, etc.)
- **Example Paths**: `pkg/{package}/*.go`, `pkg/{package}/providers/*.go`, `cmd/{app}/main.go`
- **Task Verbs**: Create, Implement, Add, Build, Write
- **Validation**: Run tests on actual implementation files
- **When**: Building new packages, features, or capabilities from scratch

### 2️⃣ ANALYSIS/AUDIT (`specs/NNN-for-the-{package}/`)
- **Goal**: Document findings about EXISTING code without modifying it
- **File Targets**: All tasks write to `specs/` directory only (`.md` files)
- **Example Paths**: `specs/NNN-for-the-{package}/findings/*.md`, `specs/NNN-for-the-{package}/analysis/*.md`
- **Task Verbs**: Verify, Analyze, Validate, Document, Review, Audit
- **Validation**: Ensure comprehensive documentation of current state
- **When**: Auditing existing packages for compliance, understanding architecture

### 3️⃣ CORRECTION/ENHANCEMENT (follows analysis)
- **Goal**: Fix/improve EXISTING code in `pkg/` based on analysis findings
- **File Targets**: All tasks modify actual codebase files in `pkg/` directory
- **Example Paths**: `pkg/{package}/errors.go`, `pkg/{package}/providers/openai.go`
- **Task Verbs**: Fix, Update, Enhance, Refactor, Improve, Correct
- **Validation**: Run tests to verify fixes, check for regressions
- **When**: Applying fixes after analysis phase completes

**⚠️ IMPORTANT**: Analysis tasks (Type 2) NEVER modify `pkg/` files. Only Type 1 and Type 3 modify actual code.

## Execution Flow (main)
```
1. Identify task type (NEW FEATURE, ANALYSIS, or CORRECTION)
   → Check spec directory name pattern
   → NEW FEATURE: specs/NNN-feature-name/
   → ANALYSIS: specs/NNN-for-the-{package}/
   → CORRECTION: specs/NNN-fix-{package}-{issue}/
2. Load plan.md from feature directory
   → If not found: ERROR "No implementation plan found"
   → Extract: tech stack, libraries, structure, task type
3. Load optional design documents:
   → data-model.md: Extract entities → model/analysis tasks
   → contracts/: Each file → contract test/verification task
   → research.md: Extract decisions → setup/analysis tasks
4. Generate tasks by category AND type:
   
   FOR NEW FEATURES:
   → Setup: project init in pkg/{package}/, dependencies, config files
   → Tests: test files in pkg/{package}/*_test.go
   → Core: implementation files in pkg/{package}/*.go
   → Integration: registry, factory patterns
   → Polish: benchmarks, README.md, documentation
   
   FOR ANALYSIS:
   → Setup: analysis tools, verification scripts
   → Verification: contract checks → findings/*.md
   → Analysis: entity examination → analysis/*.md
   → Validation: scenario testing → validation/*.md
   → Reporting: consolidated reports → report/*.md
   
   FOR CORRECTIONS:
   → Setup: test environment, reproduction
   → Tests: add missing tests in pkg/{package}/*_test.go
   → Fixes: modify code in pkg/{package}/*.go
   → Verification: run test suite, check regressions
   → Documentation: update README.md, godoc
5. Apply task rules:
   → Different files = mark [P] for parallel
   → Same file = sequential (no [P])
   → Tests before implementation (TDD for features/corrections)
   → Verification before fixes (for corrections)
6. Number tasks sequentially (T001, T002...)
7. Generate dependency graph
8. Create parallel execution examples
9. Validate task completeness based on type:
   
   NEW FEATURES:
   → All contracts have tests?
   → All entities have models?
   → All endpoints implemented?
   → Constitutional files present (config.go, metrics.go, errors.go)?
   
   ANALYSIS:
   → All contracts verified with findings?
   → All entities analyzed?
   → All scenarios validated?
   → Reports comprehensive?
   
   CORRECTIONS:
   → All issues have test coverage?
   → All fixes target correct files?
   → No regressions possible?
10. Return: SUCCESS (tasks ready for execution)
```

## Format: `[ID] [P?] Description with file path`
- **[P]**: Can run in parallel (different files, no dependencies)
- **MUST include exact file paths** in every task description
- Use action verbs matching task type (Create/Fix/Analyze)

## Path Conventions by Task Type

### For NEW FEATURES (Create new code):
- **Go packages**: `pkg/{package}/*.go`, `pkg/{package}/providers/*.go`, `pkg/{package}/internal/*.go`
- **Tests**: `pkg/{package}/*_test.go`, `tests/integration/{package}_test.go`
- **Commands**: `cmd/{app}/main.go`, `cmd/{app}/commands/*.go`
- **Config**: `pkg/{package}/config.go`, `configs/{package}.yaml`

### For ANALYSIS (Document existing code):
- **Findings**: `specs/NNN-for-the-{package}/findings/{topic}-finding.md`
- **Analysis**: `specs/NNN-for-the-{package}/analysis/{entity}-analysis.md`
- **Validation**: `specs/NNN-for-the-{package}/validation/{scenario}-validation.md`
- **Reports**: `specs/NNN-for-the-{package}/report/{report-type}.md`

### For CORRECTIONS (Fix existing code):
- **Code fixes**: `pkg/{package}/*.go` (existing files being modified)
- **Test additions**: `pkg/{package}/*_test.go` (add missing tests)
- **Documentation**: `pkg/{package}/README.md`, godoc comments

## Phase 3.1: Setup
- [ ] T001 Create project structure per implementation plan
- [ ] T002 Initialize [language] project with [framework] dependencies
- [ ] T003 [P] Configure linting and formatting tools

## Phase 3.2: Tests First (TDD + Constitutional Compliance) ⚠️ MUST COMPLETE BEFORE 3.3
**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**
**CONSTITUTIONAL: All packages MUST include test_utils.go and advanced_test.go**
- [ ] T004 [P] Create test_utils.go with AdvancedMock{Package} and testing utilities  
- [ ] T005 [P] Create advanced_test.go with table-driven tests and concurrency tests
- [ ] T006 [P] Contract test POST /api/users in tests/contract/test_users_post.py
- [ ] T007 [P] Contract test GET /api/users/{id} in tests/contract/test_users_get.py
- [ ] T008 [P] Integration test user registration in tests/integration/test_registration.py
- [ ] T009 [P] Integration test auth flow in tests/integration/test_auth.py

## Phase 3.3: Core Implementation (ONLY after tests are failing)
**CONSTITUTIONAL: MUST implement OTEL metrics, structured errors, and registry patterns**
- [ ] T010 [P] Create config.go with validation and functional options
- [ ] T011 [P] Create metrics.go with OTEL implementation (NewMetrics pattern)
- [ ] T012 [P] Create errors.go with Op/Err/Code pattern
- [ ] T013 [P] User model in src/models/user.py
- [ ] T014 [P] UserService CRUD in src/services/user_service.py
- [ ] T015 [P] CLI --create-user in src/cli/user_commands.py
- [ ] T016 POST /api/users endpoint
- [ ] T017 GET /api/users/{id} endpoint
- [ ] T018 Input validation with constitutional error patterns
- [ ] T019 Registry/factory implementation (if multi-provider package)

## Phase 3.4: Integration
- [ ] T020 Connect UserService to DB
- [ ] T021 Auth middleware
- [ ] T022 OTEL metrics and tracing integration
- [ ] T023 Request/response logging with structured format
- [ ] T024 CORS and security headers

## Phase 3.5: Polish & Constitutional Compliance
- [ ] T025 [P] Unit tests for validation in tests/unit/test_validation.py
- [ ] T026 Performance benchmarks (constitutional requirement)
- [ ] T027 [P] Integration tests in tests/integration/ directory
- [ ] T028 [P] Update README.md with constitutional compliance
- [ ] T029 Constitutional compliance verification
- [ ] T030 Remove duplication and finalize

## Dependencies
- Constitutional files (T004-T005) before all other tests
- Tests (T006-T009) before implementation (T010-T019)
- Constitutional compliance (T010-T012) before core implementation
- T013 blocks T014, T020
- T021 blocks T024
- Implementation before polish (T025-T030)

## Parallel Example
```
# Launch T004-T007 together:
Task: "Contract test POST /api/users in tests/contract/test_users_post.py"
Task: "Contract test GET /api/users/{id} in tests/contract/test_users_get.py"
Task: "Integration test registration in tests/integration/test_registration.py"
Task: "Integration test auth in tests/integration/test_auth.py"
```

## Notes
- [P] tasks = different files, no dependencies
- Verify tests fail before implementing
- Commit after each task
- Avoid: vague tasks, same file conflicts

## Post-Implementation Workflow (MANDATORY)
**After ALL tasks are completed, follow this standardized workflow:**

1. **Create comprehensive commit message**:
   ```
   feat([package]): Complete constitutional compliance with [achievements]
   
   MAJOR ACHIEVEMENTS:
   ✅ Constitutional Compliance: [compliance summary]
   ✅ Performance Excellence: [performance results]
   ✅ Testing Infrastructure: [testing achievements]
   ✅ [Other key achievements]
   
   CORE ENHANCEMENTS:
   - [List major enhancements]
   - [Include file additions/modifications]
   
   PERFORMANCE RESULTS:
   - [Specific performance metrics]
   - [Benchmark results vs targets]
   
   FILES ADDED/MODIFIED:
   - [List of key files with descriptions]
   
   Zero breaking changes - all existing functionality preserved and enhanced.
   ```

2. **Push feature branch to origin**:
   ```bash
   git add .
   git commit -m "your comprehensive message above"
   git push origin [feature-branch-name]
   ```

3. **Create Pull Request**:
   - From feature branch to `develop` branch
   - Include implementation summary and constitutional compliance status
   - Reference any issues or specifications addressed

4. **Merge to develop**:
   - Ensure all tests pass in CI/CD
   - Merge PR to develop branch
   - Verify functionality is preserved post-merge

5. **Post-merge validation**:
   ```bash
   git checkout develop
   git pull origin develop
   go test ./pkg/... -v
   go test ./tests/integration/... -v
   ```

**CRITICAL**: Do not proceed to next feature until this workflow is complete and validated.

## Task Generation Rules
*Applied during main() execution*

1. **From Contracts**:
   - Each contract file → contract test task [P]
   - Each endpoint → implementation task
   
2. **From Data Model**:
   - Each entity → model creation task [P]
   - Relationships → service layer tasks
   
3. **From User Stories**:
   - Each story → integration test [P]
   - Quickstart scenarios → validation tasks

4. **Ordering**:
   - Setup → Tests → Models → Services → Endpoints → Polish
   - Dependencies block parallel execution

## Validation Checklist
*GATE: Checked by main() before returning*

### Constitutional Compliance
- [ ] Package structure tasks follow standard layout (config.go, metrics.go, errors.go, etc.)
- [ ] OTEL metrics implementation tasks included
- [ ] Test utilities (test_utils.go, advanced_test.go) tasks present
- [ ] Registry pattern tasks for multi-provider packages

### Task Quality
- [ ] All contracts have corresponding tests
- [ ] All entities have model tasks
- [ ] All tests come before implementation
- [ ] Parallel tasks truly independent
- [ ] Each task specifies exact file path
- [ ] No task modifies same file as another [P] task

## Quick Task Type Reference

| Task Type | File Targets | Task Verbs | Example |
|-----------|-------------|------------|---------|
| **NEW FEATURE** | `pkg/`, `cmd/`, `internal/` | Create, Implement, Add, Build, Write | T001 Create Embedder interface in pkg/embeddings/iface/embedder.go |
| **ANALYSIS** | `specs/NNN-for-the-{package}/` | Verify, Analyze, Validate, Document, Review | T001 Analyze error handling patterns in specs/008-for-the-embeddings/findings/error-handling.md |
| **CORRECTION** | `pkg/` (existing files) | Fix, Update, Enhance, Refactor, Improve | T001 Fix error wrapping in pkg/embeddings/providers/openai.go |

**REMEMBER**: Analysis tasks write to `specs/` only. Implementation/Correction tasks write to `pkg/` only.

---
*Based on Constitution v1.1.0 - See `.specify/memory/constitution.md`*
