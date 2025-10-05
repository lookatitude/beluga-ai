# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`
**Prerequisites**: plan.md (required), research.md, data-model.md, contracts/

## Execution Flow (main)
```
1. Load plan.md from feature directory
   → If not found: ERROR "No implementation plan found"
   → Extract: tech stack, libraries, structure
2. Load optional design documents:
   → data-model.md: Extract entities → model tasks
   → contracts/: Each file → contract test task
   → research.md: Extract decisions → setup tasks
3. Generate tasks by category:
   → Setup: project init, dependencies, linting
   → Tests: contract tests, integration tests
   → Core: models, services, CLI commands
   → Integration: DB, middleware, logging
   → Polish: unit tests, performance, docs
4. Apply task rules:
   → Different files = mark [P] for parallel
   → Same file = sequential (no [P])
   → Tests before implementation (TDD)
5. Number tasks sequentially (T001, T002...)
6. Generate dependency graph
7. Create parallel execution examples
8. Validate task completeness:
   → All contracts have tests?
   → All entities have models?
   → All endpoints implemented?
9. Return: SUCCESS (tasks ready for execution)
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions
- **Single project**: `src/`, `tests/` at repository root
- **Web app**: `backend/src/`, `frontend/src/`
- **Mobile**: `api/src/`, `ios/src/` or `android/src/`
- Paths shown below assume single project - adjust based on plan.md structure

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

---
*Based on Constitution v1.0.0 - See `.specify/memory/constitution.md`*