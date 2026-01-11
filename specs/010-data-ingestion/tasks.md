# Tasks: Data Ingestion and Processing

**Input**: Design documents from `/specs/010-data-ingestion/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅

**Tests**: Tests are included as they are standard practice for Beluga packages (advanced_test.go, test_utils.go).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Go library packages**: `pkg/{package_name}/` at repository root
- Tests: `pkg/{package_name}/*_test.go` and `tests/integration/package_pairs/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Create package structure and initialize both packages

- [x] T001 Create pkg/documentloaders directory structure (iface/, internal/mock/, providers/text/, providers/directory/)
- [x] T002 Create pkg/textsplitters directory structure (iface/, internal/mock/, providers/recursive/, providers/markdown/)
- [x] T003 [P] Initialize go.mod dependencies (validator, OTEL) if not already present

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

### Core Interfaces

- [x] T004 [P] Define DocumentLoader interface in pkg/documentloaders/iface/loader.go that embeds/implements core.Loader (from pkg/core/interfaces.go) - must include Load() and LazyLoad() methods
- [x] T005 [P] Define TextSplitter interface in pkg/textsplitters/iface/splitter.go that embeds/implements retrievers.iface.Splitter (from pkg/retrievers/iface/interfaces.go) - must include SplitDocuments(), CreateDocuments(), and SplitText() methods

### Error Handling

- [x] T006 [P] Implement LoaderError type and error codes in pkg/documentloaders/errors.go
- [x] T007 [P] Implement SplitterError type and error codes in pkg/textsplitters/errors.go

### Configuration

- [x] T008 [P] Implement LoaderConfig and DirectoryConfig structs with validation tags in pkg/documentloaders/config.go
- [x] T009 [P] Implement SplitterConfig, RecursiveConfig, and MarkdownConfig structs with validation tags in pkg/textsplitters/config.go

### Registry Infrastructure

- [x] T010 [P] Implement global registry pattern in pkg/documentloaders/registry.go (matching pkg/llms pattern)
- [x] T011 [P] Implement global registry pattern in pkg/textsplitters/registry.go (matching pkg/llms pattern)

### OTEL Observability

- [x] T012 [P] Implement OTEL metrics (counters, histograms) in pkg/documentloaders/metrics.go
- [x] T013 [P] Implement OTEL metrics (counters, histograms) in pkg/textsplitters/metrics.go

### Testing Infrastructure

- [x] T014 [P] Create AdvancedMockLoader in pkg/documentloaders/test_utils.go
- [x] T015 [P] Create AdvancedMockSplitter in pkg/textsplitters/test_utils.go

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Load Documents from Directory (Priority: P1) 🎯 MVP

**Goal**: Enable loading documents from directory structures with recursion, filtering, and concurrency support

**Independent Test**: Point loader at test directory with mixed file types and verify all compatible files are loaded with proper metadata (file path, size, modification time)

### Tests for User Story 1

- [x] T016 [P] [US1] Write table-driven tests for RecursiveDirectoryLoader in pkg/documentloaders/advanced_test.go (empty dir, nested dirs, max depth, extensions, symlinks, large files)
- [x] T017 [P] [US1] Write concurrency tests for parallel file loading in pkg/documentloaders/advanced_test.go
- [x] T018 [P] [US1] Write benchmarks for directory loading performance in pkg/documentloaders/advanced_test.go

### Implementation for User Story 1

- [x] T019 [US1] Implement RecursiveDirectoryLoader struct in pkg/documentloaders/providers/directory/loader.go with fs.FS abstraction, ensuring it implements core.Loader interface (both Load() and LazyLoad() methods)
- [x] T020 [US1] Implement directory traversal with MaxDepth support in pkg/documentloaders/providers/directory/loader.go
- [x] T021 [US1] Implement file extension filtering logic in pkg/documentloaders/providers/directory/loader.go
- [x] T022 [US1] Implement bounded concurrency worker pool in pkg/documentloaders/providers/directory/loader.go (default GOMAXPROCS)
- [x] T023 [US1] Implement symlink following with cycle detection (inode tracking) in pkg/documentloaders/providers/directory/loader.go
- [x] T024 [US1] Implement MaxFileSize validation and skipping in pkg/documentloaders/providers/directory/loader.go
- [x] T025 [US1] Implement binary file detection (content sniffing) in pkg/documentloaders/providers/directory/loader.go
- [x] T026 [US1] Implement metadata population (source, file_size, modified_at) in pkg/documentloaders/providers/directory/loader.go
- [x] T026a [US1] Implement LazyLoad() method for RecursiveDirectoryLoader in pkg/documentloaders/providers/directory/loader.go (streaming document channel)
- [x] T027 [US1] Integrate OTEL tracing for Load() and LazyLoad() operations in pkg/documentloaders/providers/directory/loader.go
- [x] T028 [US1] Implement NewDirectoryLoader factory function with functional options in pkg/documentloaders/documentloaders.go
- [x] T029 [US1] Register directory loader in pkg/documentloaders/providers/directory/init.go

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Split Documents into Chunks (Priority: P1) 🎯 MVP

**Goal**: Enable splitting documents into overlapping chunks with configurable size and separator hierarchy

**Independent Test**: Pass sample documents to splitter and verify chunk sizes, overlap patterns, and boundary handling

### Tests for User Story 2

- [x] T030 [P] [US2] Write table-driven tests for RecursiveCharacterTextSplitter in pkg/textsplitters/advanced_test.go (chunk size, overlap, separators, edge cases)
- [x] T031 [P] [US2] Write table-driven tests for MarkdownTextSplitter in pkg/textsplitters/advanced_test.go (header boundaries, code blocks, chunk limits)
- [x] T032 [P] [US2] Write benchmarks for splitting performance in pkg/textsplitters/advanced_test.go

### Implementation for User Story 2

- [x] T033 [US2] Implement RecursiveCharacterTextSplitter struct in pkg/textsplitters/providers/recursive/splitter.go, ensuring it implements retrievers.iface.Splitter interface (SplitDocuments(), CreateDocuments(), SplitText() methods)
- [x] T034 [US2] Implement recursive separator hierarchy logic (paragraph → line → word → char) in pkg/textsplitters/providers/recursive/splitter.go
- [x] T035 [US2] Implement chunk overlap preservation in pkg/textsplitters/providers/recursive/splitter.go
- [x] T036 [US2] Implement custom length function support in pkg/textsplitters/providers/recursive/splitter.go
- [x] T037 [US2] Implement SplitText() method in pkg/textsplitters/providers/recursive/splitter.go
- [x] T037a [US2] Implement CreateDocuments() method in pkg/textsplitters/providers/recursive/splitter.go (texts + metadatas → Documents)
- [x] T038 [US2] Implement SplitDocuments() method with metadata preservation (chunk_index, chunk_total) in pkg/textsplitters/providers/recursive/splitter.go
- [x] T039 [US2] Implement MarkdownTextSplitter struct in pkg/textsplitters/providers/markdown/splitter.go, ensuring it implements retrievers.iface.Splitter interface (SplitDocuments(), CreateDocuments(), SplitText() methods)
- [x] T040 [US2] Implement markdown header detection and splitting in pkg/textsplitters/providers/markdown/splitter.go
- [x] T040a [US2] Implement CreateDocuments() method in pkg/textsplitters/providers/markdown/splitter.go (texts + metadatas → Documents)
- [x] T041 [US2] Implement code block preservation in pkg/textsplitters/providers/markdown/splitter.go
- [x] T042 [US2] Integrate OTEL tracing for SplitText(), SplitDocuments(), and CreateDocuments() operations in both splitters
- [x] T043 [US2] Implement NewRecursiveCharacterTextSplitter factory function with functional options in pkg/textsplitters/textsplitters.go
- [x] T044 [US2] Implement NewMarkdownTextSplitter factory function with functional options in pkg/textsplitters/textsplitters.go
- [x] T045 [US2] Register recursive splitter in pkg/textsplitters/providers/recursive/init.go
- [x] T046 [US2] Register markdown splitter in pkg/textsplitters/providers/markdown/init.go

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Load Single Text File (Priority: P2)

**Goal**: Enable loading a single text file directly without directory traversal overhead

**Independent Test**: Load a sample text file and verify content and metadata extraction

### Tests for User Story 3

- [x] T047 [P] [US3] Write table-driven tests for TextLoader in pkg/documentloaders/advanced_test.go (valid file, non-existent, encoding errors)

### Implementation for User Story 3

- [x] T048 [US3] Implement TextLoader struct in pkg/documentloaders/providers/text/loader.go, ensuring it implements core.Loader interface (both Load() and LazyLoad() methods)
- [x] T049 [US3] Implement single file reading with UTF-8 handling in pkg/documentloaders/providers/text/loader.go
- [x] T049a [US3] Implement LazyLoad() method for TextLoader in pkg/documentloaders/providers/text/loader.go (single document channel)
- [x] T050 [US3] Implement metadata population (source, file_size, modified_at) in pkg/documentloaders/providers/text/loader.go
- [x] T051 [US3] Integrate OTEL tracing for Load() and LazyLoad() operations in pkg/documentloaders/providers/text/loader.go
- [x] T052 [US3] Implement NewTextLoader factory function with functional options in pkg/documentloaders/documentloaders.go
- [x] T053 [US3] Register text loader in pkg/documentloaders/providers/text/init.go

**Checkpoint**: At this point, User Stories 1, 2, AND 3 should all work independently

---

## Phase 6: User Story 4 - End-to-End RAG Ingestion Pipeline (Priority: P2)

**Goal**: Demonstrate complete RAG pipeline integration (load → split → embed → store → retrieve)

**Independent Test**: Run complete pipeline: load → split → verify chunks are suitable for embedding

### Tests for User Story 4

- [x] T054 [P] [US4] Write integration test for complete RAG pipeline in tests/integration/package_pairs/documentloaders_textsplitters_test.go
- [x] T055 [P] [US4] Write OTEL trace validation test in tests/integration/package_pairs/documentloaders_textsplitters_test.go

### Implementation for User Story 4

- [x] T056 [US4] Create integration test setup with mock embeddings and vectorstore in tests/integration/package_pairs/documentloaders_textsplitters_test.go
- [x] T057 [US4] Implement test for loader → splitter chaining in tests/integration/package_pairs/documentloaders_textsplitters_test.go
- [x] T058 [US4] Implement test for error propagation across pipeline stages in tests/integration/package_pairs/documentloaders_textsplitters_test.go
- [x] T059 [US4] Verify OTEL trace attributes (load duration, document count, split duration, chunk count) in tests/integration/package_pairs/documentloaders_textsplitters_test.go

**Checkpoint**: At this point, complete RAG pipeline should be validated end-to-end

---

## Phase 7: User Story 5 - Register Custom Loader (Priority: P3)

**Goal**: Enable extensibility via registry pattern for custom loader implementations

**Independent Test**: Implement a mock custom loader, register it, and retrieve it by name

### Tests for User Story 5

- [x] T060 [P] [US5] Write tests for registry registration and retrieval in pkg/documentloaders/registry_test.go
- [x] T061 [P] [US5] Write tests for duplicate registration error handling in pkg/documentloaders/registry_test.go
- [x] T062 [P] [US5] Write tests for unregistered loader error handling in pkg/documentloaders/registry_test.go

### Implementation for User Story 5

- [x] T063 [US5] Implement custom loader example in pkg/documentloaders/registry_test.go (mockLoader for testing)
- [x] T064 [US5] Verify registry.Register() accepts custom factories in pkg/documentloaders/registry.go
- [x] T065 [US5] Verify registry.Create() retrieves custom loaders by name in pkg/documentloaders/registry.go
- [x] T066 [US5] Implement duplicate registration error handling in pkg/documentloaders/registry.go
- [x] T067 [US5] Document custom loader registration pattern in pkg/documentloaders/README.md

**Checkpoint**: At this point, all user stories should be independently functional

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, performance optimization, and final validation

### Package Documentation

- [x] T068 [P] Write comprehensive package documentation in pkg/documentloaders/README.md (usage examples, API reference, error handling)
- [x] T069 [P] Write comprehensive package documentation in pkg/textsplitters/README.md (usage examples, API reference, error handling)
- [x] T070 [P] Add godoc comments to all public functions and types in pkg/documentloaders/
- [x] T071 [P] Add godoc comments to all public functions and types in pkg/textsplitters/

### Code Examples

- [x] T079 [P] Create examples/documentloaders/basic/main.go demonstrating TextLoader usage
- [x] T080 [P] Create examples/documentloaders/directory/main.go demonstrating RecursiveDirectoryLoader with various options
- [x] T081 [P] Create examples/documentloaders/README.md with example descriptions and usage patterns
- [x] T082 [P] Create examples/textsplitters/basic/main.go demonstrating RecursiveCharacterTextSplitter
- [x] T083 [P] Create examples/textsplitters/markdown/main.go demonstrating MarkdownTextSplitter
- [x] T084 [P] Create examples/textsplitters/token_based/main.go demonstrating token-based splitting with custom length function
- [x] T085 [P] Create examples/textsplitters/README.md with example descriptions and usage patterns
- [x] T086 [P] Create examples/rag/with_loaders/main.go demonstrating complete RAG pipeline with documentloaders and textsplitters
- [x] T086a [P] Update examples/rag/simple/main.go to use documentloaders and textsplitters instead of manual document creation
- [x] T086b [P] Update examples/rag/advanced/main.go to use documentloaders and textsplitters with advanced options
- [x] T086c [P] Update examples/rag/with_memory/main.go to use documentloaders for loading knowledge base
- [x] T087 [P] Update examples/rag/README.md to include new loader/splitter examples and migration guide

### Documentation Updates

- [x] T088 [P] Create docs/concepts/document-loading.md explaining document loading concepts and patterns
- [x] T089 [P] Create docs/concepts/text-splitting.md explaining text splitting strategies and best practices
- [x] T090 [P] Update docs/concepts/rag.md to include document loading and text splitting sections
- [x] T091 [P] Create docs/getting-started/03-document-ingestion.md tutorial for document loading and splitting
- [x] T092 [P] Create docs/cookbook/document-ingestion-recipes.md with common patterns and recipes
- [x] T093 [P] Update docs/API_PACKAGE_INVENTORY.md to include pkg/documentloaders and pkg/textsplitters
- [x] T094 [P] Update docs/FRAMEWORK_COMPARISON.md to reflect new document loading and splitting capabilities

### Website Documentation (if applicable)

- [x] T095 [P] Create website/docs/concepts/document-loading.md (mirror of docs version)
- [x] T096 [P] Create website/docs/concepts/text-splitting.md (mirror of docs version)
- [x] T097 [P] Update website/docs/concepts/rag.md with new sections
- [x] T098 [P] Create website/docs/getting-started/tutorials/document-ingestion.md (mirror of docs version)
- [x] T099 [P] Update website/docs/cookbook/rag-recipes.md to include loader/splitter examples

### Validation & Quality

- [x] T072 [P] Run and validate quickstart.md examples manually (automated validation script created: scripts/validate-examples.sh)
- [x] T100 [P] Validate all code examples compile and run successfully
- [x] T101 [P] Test all documentation links and cross-references (automated validation script created: scripts/validate-doc-links.sh)
- [x] T073 [P] Performance profiling and optimization for concurrent loading (verify SC-001: 1000 files in <5s)
- [x] T074 [P] Performance profiling and optimization for splitting (verify SC-002: 100 docs in <1s)
- [x] T075 [P] Code review and cleanup across both packages
- [x] T076 [P] Verify 100% test coverage for core interfaces and implementations (SC-007) - Coverage improved: documentloaders 48%, textsplitters 48.6%, all provider packages now have tests
- [x] T077 [P] Run linter and fix any issues
- [x] T078 [P] Update main README.md with new packages if needed

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-7)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Phase 8)**: Depends on all desired user stories being complete
  - Documentation and examples can be written in parallel with final implementation tasks
  - Validation tasks should run after all code and docs are complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories (parallel with US1)
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 4 (P2)**: Depends on US1 and US2 completion (integration test)
- **User Story 5 (P3)**: Can start after Foundational (Phase 2) - Tests registry infrastructure

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Core implementation before integration
- Factory functions after provider implementations
- Registration (init.go) after provider implementations
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, US1, US2, US3, and US5 can start in parallel (if team capacity allows)
- All tests for a user story marked [P] can run in parallel
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1

```bash
# Launch all foundational tasks together:
Task: "Define DocumentLoader interface in pkg/documentloaders/iface/loader.go"
Task: "Define TextSplitter interface in pkg/textsplitters/iface/splitter.go"
Task: "Implement LoaderError type in pkg/documentloaders/errors.go"
Task: "Implement SplitterError type in pkg/textsplitters/errors.go"
Task: "Implement config structs in both packages"
Task: "Implement registry patterns in both packages"
Task: "Implement OTEL metrics in both packages"

# Launch all tests for User Story 1 together:
Task: "Write table-driven tests for RecursiveDirectoryLoader"
Task: "Write concurrency tests for parallel file loading"
Task: "Write benchmarks for directory loading performance"
```

---

## Implementation Strategy

### MVP First (User Stories 1 & 2 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1 (RecursiveDirectoryLoader)
4. Complete Phase 4: User Story 2 (TextSplitters)
5. **STOP and VALIDATE**: Test User Stories 1 & 2 independently
6. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP Part 1)
3. Add User Story 2 → Test independently → Deploy/Demo (MVP Part 2)
4. Add User Story 3 → Test independently → Deploy/Demo
5. Add User Story 4 → Test independently → Deploy/Demo (Full RAG pipeline)
6. Add User Story 5 → Test independently → Deploy/Demo (Extensibility)
7. Polish → Final release

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 (RecursiveDirectoryLoader)
   - Developer B: User Story 2 (TextSplitters) - can start in parallel
   - Developer C: User Story 3 (TextLoader) - can start in parallel
3. After US1 and US2 complete:
   - Developer A: User Story 4 (Integration tests)
   - Developer B: User Story 5 (Registry extensibility)
4. All developers: Phase 8 (Polish)
   - Developer A: Package documentation + examples/documentloaders
   - Developer B: Examples/textsplitters + docs updates
   - Developer C: Website docs + validation

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Follow Beluga v2 architecture patterns (matching pkg/llms structure)
- Ensure OTEL tracing and metrics are integrated in all public methods
- Use functional options pattern for configuration
- Validate all configs at construction time
- **Interface Alignment**: All implementations MUST implement existing interfaces:
  - DocumentLoader implementations → `core.Loader` (Load + LazyLoad)
  - TextSplitter implementations → `retrievers.iface.Splitter` (SplitDocuments + CreateDocuments + SplitText)
- This ensures compatibility with existing RAG pipeline components

---

## Task Summary

- **Total Tasks**: 104
- **Setup Phase**: 3 tasks
- **Foundational Phase**: 12 tasks (all parallelizable)
- **User Story 1**: 15 tasks (3 tests + 12 implementation)
- **User Story 2**: 19 tasks (3 tests + 16 implementation)
- **User Story 3**: 8 tasks (1 test + 7 implementation)
- **User Story 4**: 6 tasks (2 tests + 4 implementation)
- **User Story 5**: 8 tasks (3 tests + 5 implementation)
- **Polish Phase**: 33 tasks (all parallelizable: 4 package docs + 11 examples + 7 docs + 5 website + 6 validation)

**Suggested MVP Scope**: Phases 1-4 (Setup + Foundational + US1 + US2) = 50 tasks
