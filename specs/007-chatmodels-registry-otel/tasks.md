# Tasks: ChatModels Global Registry & OTEL Integration

**Input**: Design documents from `/specs/007-chatmodels-registry-otel/`
**Prerequisites**: plan.md (required), research.md, data-model.md, contracts/

## Execution Flow (main)
```
1. Load plan.md from feature directory
   → Extract: Go 1.21+, OTEL libraries, extending existing chatmodels package structure
2. Load design documents:
   → Registry contracts → registry implementation tasks
   → OTEL contracts → observability implementation tasks
   → Provider contracts → provider interface tasks
   → Integration contracts → end-to-end integration tasks
3. Generate tasks by category following TDD approach:
   → Setup: project structure, dependencies
   → Tests: constitutional compliance tests first
   → Core: registry, OTEL, provider interfaces
   → Integration: chatmodels package integration
   → Polish: performance, documentation
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
   → All patterns implemented?
9. Return: SUCCESS (tasks ready for execution)
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- Include exact file paths in descriptions

## Path Conventions
- **Package extension**: `pkg/chatmodels/` (extending existing structure)
- **New files**: Add to existing package structure following constitution v1.0.0
- **Test files**: `pkg/chatmodels/` for unit tests, `tests/integration/` for integration tests

## Phase 3.1: Setup
- [ ] T001 Create project structure per implementation plan (registry.go, di.go, iface/provider.go, metrics.go)
- [ ] T002 Initialize Go dependencies for OTEL integration (go.opentelemetry.io libraries)

## Phase 3.2: Tests First (TDD + Constitutional Compliance) ⚠️ MUST COMPLETE BEFORE 3.3
**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**
**CONSTITUTIONAL: Extend existing test_utils.go and advanced_test.go with registry and OTEL testing utilities**

- [ ] T003 [P] Extend test_utils.go with AdvancedMockRegistry and OTEL testing utilities in pkg/chatmodels/test_utils.go
- [ ] T004 [P] Extend advanced_test.go with registry concurrency tests and OTEL integration tests in pkg/chatmodels/advanced_test.go
- [ ] T005 [P] Contract test for RegisterGlobal function in pkg/chatmodels/advanced_test.go
- [ ] T006 [P] Contract test for NewRegistryProvider function in pkg/chatmodels/advanced_test.go
- [ ] T007 [P] Contract test for DI container OTEL integration in pkg/chatmodels/advanced_test.go
- [ ] T008 [P] Contract test for provider interface CreateChatModel method in pkg/chatmodels/advanced_test.go
- [ ] T009 [P] Integration test for registry + chatmodels package in tests/integration/test_chatmodels_registry.go
- [ ] T010 [P] Integration test for OTEL observability in chatmodels operations in tests/integration/test_chatmodels_otel.go

## Phase 3.3: Core Implementation (ONLY after tests are failing)
**CONSTITUTIONAL: MUST implement OTEL metrics, structured errors, and registry patterns following reference implementations**

- [ ] T011 [P] Create iface/provider.go with Provider interface following pkg/config/iface.Provider pattern
- [ ] T012 [P] Create registry.go with ConfigProviderRegistry-style implementation following pkg/config/registry.go
- [ ] T013 [P] Create di.go with Container pattern following pkg/core/di.go for OTEL integration
- [ ] T014 [P] Create metrics.go with OTEL implementation following pkg/core/metrics.go pattern
- [ ] T015 Enhance errors.go with registry error patterns following pkg/core/errors.go
- [ ] T016 Enhance config.go with provider registry configuration following config.ProviderOptions
- [ ] T017 [P] Implement RegisterGlobal function in registry.go following pkg/config/registry.go pattern
- [ ] T018 [P] Implement NewRegistryProvider function in registry.go following pkg/config/registry.go pattern
- [ ] T019 [P] Implement DI container OTEL integration in di.go following pkg/core/di.go pattern
- [ ] T020 [P] Implement provider interface in existing OpenAI provider following new iface/provider.go
- [ ] T021 [P] Integrate registry with existing chatmodels.go NewChatModel function
- [ ] T022 [P] Add OTEL tracing to existing chat model operations following pkg/core/di.go patterns

## Phase 3.4: Integration
- [ ] T023 Connect provider registry to chatmodels factory functions
- [ ] T024 Integrate OTEL metrics collection with existing chat model operations
- [ ] T025 Add structured logging with OTEL context propagation following pkg/core/di.go
- [ ] T026 Implement health checking for registry and provider operations
- [ ] T027 Add provider metadata management following config.ProviderMetadata pattern
- [ ] T028 Implement graceful degradation when OTEL services unavailable

## Phase 3.5: Polish & Constitutional Compliance
- [ ] T029 [P] Performance benchmarks for registry operations (<1ms target) in pkg/chatmodels/advanced_test.go
- [ ] T030 [P] Performance benchmarks for provider creation (<10ms target) in pkg/chatmodels/advanced_test.go
- [ ] T031 [P] Unit tests for configuration validation in pkg/chatmodels/advanced_test.go
- [ ] T032 [P] Integration tests for end-to-end registry + OTEL + chatmodels flow in tests/integration/
- [ ] T033 [P] Update README.md with registry and OTEL documentation
- [ ] T034 [P] Constitutional compliance verification for all new components
- [ ] T035 Remove any duplication and finalize implementation

## Dependencies
- Constitutional files (T003-T004) before all other tests
- Tests (T005-T010) before implementation (T011-T022)
- Constitutional compliance (T011-T014) before core implementation
- T011 blocks T017-T018 (provider interface needed for registry)
- T012 blocks T019 (registry needed for DI)
- T013 blocks T021-T022 (DI container needed for integration)
- Implementation before polish (T029-T035)
- T016 blocks T023 (config needed for integration)

## Parallel Execution Examples
```
# Launch T005-T010 together (all contract tests):
Task: "Contract test for RegisterGlobal function"
Task: "Contract test for NewRegistryProvider function"
Task: "Contract test for DI container OTEL integration"
Task: "Contract test for provider interface CreateChatModel method"
Task: "Integration test for registry + chatmodels package"
Task: "Integration test for OTEL observability in chatmodels operations"

# Launch T017-T022 together (core implementation):
Task: "Implement RegisterGlobal function in registry.go"
Task: "Implement NewRegistryProvider function in registry.go"
Task: "Implement DI container OTEL integration in di.go"
Task: "Implement provider interface in existing OpenAI provider"
Task: "Integrate registry with existing chatmodels.go NewChatModel function"
Task: "Add OTEL tracing to existing chat model operations"
```

## Notes
- [P] tasks = different files, no dependencies, can run in parallel
- Verify tests fail before implementing (TDD approach)
- Commit after each task completion
- Follow reference implementations exactly (pkg/config/registry.go, pkg/core/di.go, pkg/core/metrics.go)
- Performance targets: <1ms registry resolution, <10ms provider creation, <5% OTEL overhead
- Maintain backward compatibility with existing chatmodels package

## Post-Implementation Workflow (MANDATORY)
**After ALL tasks are completed, follow this standardized workflow:**

1. **Create comprehensive commit message**:
   ```
   feat(pkg/chatmodels): Complete global registry and OTEL integration

   MAJOR ACHIEVEMENTS:
   ✅ Constitutional Compliance: Full adherence to Beluga AI Framework v1.0.0
   ✅ Performance Excellence: <1ms registry resolution, <10ms provider creation, <5% OTEL overhead
   ✅ Testing Infrastructure: Extended test_utils.go and advanced_test.go with registry/OTEL utilities
   ✅ Framework Consistency: Exact replication of pkg/config/registry.go and pkg/core/di.go patterns

   CORE ENHANCEMENTS:
   - Added global provider registry following ConfigProviderRegistry pattern
   - Integrated OTEL observability following pkg/core/di.go Container pattern
   - Created provider interface following pkg/config/iface.Provider standard
   - Extended existing chatmodels package with backward compatibility

   PERFORMANCE RESULTS:
   - Registry resolution: <1ms (matching pkg/core/di.go performance targets)
   - Provider creation: <10ms (matching pkg/core/di.go object graph creation)
   - OTEL overhead: <5% (matching pkg/core/di.go instrumentation efficiency)

   FILES ADDED/MODIFIED:
   - pkg/chatmodels/iface/provider.go - Provider interface implementation
   - pkg/chatmodels/registry.go - Global registry following pkg/config/registry.go pattern
   - pkg/chatmodels/di.go - Dependency injection with OTEL integration following pkg/core/di.go
   - pkg/chatmodels/metrics.go - OTEL metrics collection following pkg/core/metrics.go
   - pkg/chatmodels/config.go - Enhanced with provider registry config
   - pkg/chatmodels/errors.go - Extended with registry error patterns following pkg/core/errors.go
   - pkg/chatmodels/test_utils.go - Enhanced with registry/OTEL testing utilities
   - pkg/chatmodels/advanced_test.go - Extended with registry integration tests

   Zero breaking changes - all existing functionality preserved and enhanced.
   ```

2. **Push feature branch to origin**:
   ```bash
   git add .
   git commit -m "your comprehensive message above"
   git push origin 007-chatmodels-registry-otel
   ```

3. **Create Pull Request**:
   - From `007-chatmodels-registry-otel` branch to `develop` branch
   - Include implementation summary and constitutional compliance status
   - Reference spec.md and plan.md for requirements traceability

4. **Merge to develop**:
   - Ensure all tests pass in CI/CD
   - Merge PR to develop branch
   - Verify functionality is preserved post-merge

5. **Post-merge validation**:
   ```bash
   git checkout develop
   git pull origin develop
   go test ./pkg/chatmodels/... -v
   go test ./tests/integration/... -v
   go test ./pkg/config/... -v  # Verify registry pattern compatibility
   go test ./pkg/core/... -v    # Verify OTEL pattern compatibility
   ```

**CRITICAL**: Do not proceed to next feature until this workflow is complete and validated.

## Task Generation Rules
*Applied during main() execution*

1. **From Registry Contracts**:
   - RegisterGlobal contract → implementation task [P]
   - NewRegistryProvider contract → implementation task [P]
   - Provider metadata contract → implementation task [P]

2. **From OTEL Contracts**:
   - DI container contract → implementation task [P]
   - Metrics collection contract → implementation task [P]
   - Tracing/logging contracts → integration tasks

3. **From Provider Contracts**:
   - Provider interface contract → implementation task [P]
   - CreateChatModel contract → implementation task [P]

4. **Ordering**:
   - Setup → Tests → Interfaces → Registry → OTEL → Integration → Polish
   - Dependencies block parallel execution where files overlap

## Validation Checklist
*GATE: Checked by main() before returning*

### Constitutional Compliance
- [x] Package structure tasks follow standard layout (config.go, metrics.go, errors.go, etc.)
- [x] OTEL metrics implementation tasks included following pkg/core/metrics.go
- [x] Test utilities (test_utils.go, advanced_test.go) extension tasks present
- [x] Registry pattern tasks following pkg/config/registry.go ConfigProviderRegistry

### Task Quality
- [x] All contracts have corresponding tests (registry, OTEL, provider contracts)
- [x] All entities have implementation tasks (registry, provider interface, OTEL components)
- [x] All tests come before implementation (TDD approach maintained)
- [x] Parallel tasks are truly independent (different files, no conflicts)
- [x] Each task specifies exact file path in pkg/chatmodels/
- [x] No task modifies same file as another [P] task

---
*Based on Constitution v1.0.0 - See `.specify/memory/constitution.md`*
*Task Count: 35 tasks (6-8 registry, 6-8 OTEL, 4-6 provider, 4-6 integration/testing)*
*Parallel Opportunities: 22 tasks marked [P] for concurrent execution*
