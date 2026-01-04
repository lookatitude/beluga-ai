# Tasks: Complete Documentation Overhaul - Godoc Automation & API Reference

**Input**: Design documents from `/specs/001-complete-documentation/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅, quickstart.md ✅

**Tests**: Manual validation tasks included for documentation quality assurance

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Documentation website**: `website/` directory at repository root
- **Source documentation**: `docs/` directory at repository root
- **Examples**: `examples/` directory at repository root
- **Package source**: `pkg/` directory at repository root
- **Scripts**: `scripts/` directory at repository root

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Environment setup and tool verification for godoc automation

- [X] T001 Verify gomarkdoc tool is installed and accessible by running `gomarkdoc --version` and checking it outputs version information
- [X] T002 Test gomarkdoc installation path resolution by checking `command -v gomarkdoc`, `${HOME}/go/bin/gomarkdoc`, and `$(go env GOPATH)/bin/gomarkdoc` all work correctly
- [X] T003 Verify Go module is accessible by running `go list -m` from repository root and confirming module name is `github.com/lookatitude/beluga-ai`
- [X] T004 Create backup of existing API documentation in `website/docs/api/packages/` by copying to `website/docs/api/packages.backup/` before regeneration
- [X] T005 Verify output directory exists and is writable by checking `website/docs/api/packages/` directory permissions and creating if missing
- [X] T006 Test Python3 availability for MDX compatibility fixes by running `python3 --version` and confirming Python 3.x is available
- [X] T007 Verify Docusaurus website can build successfully by running `cd website && npm run build` and confirming no build errors

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core godoc automation verification that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T008 Run existing `scripts/generate-docs.sh` script end-to-end and verify it completes without errors, generating documentation for all packages
- [X] T009 Audit script package list in `scripts/generate-docs.sh` lines 135-150 and compare with actual packages in `pkg/` directory to identify any missing packages
- [X] T010 Verify all main packages in script (agents, chatmodels, config, core, embeddings, llms, memory, monitoring, orchestration, prompts, retrievers, schema, server, vectorstores) exist in `pkg/` directory
- [X] T011 Verify all LLM provider packages in script (anthropic, bedrock, cohere, ollama, openai) exist in `pkg/llms/providers/` directory
- [X] T012 Verify all voice packages in script (stt, tts, vad, turndetection, transport, noise, session) exist in `pkg/voice/` directory
- [X] T013 Verify tools package path `pkg/agents/tools` exists and contains Go source files
- [X] T014 Test script error handling by temporarily renaming a package directory and verifying script handles missing packages gracefully with appropriate error messages
- [X] T015 Verify generated markdown files have proper frontmatter by checking at least 5 generated files in `website/docs/api/packages/` for YAML frontmatter with title and sidebar_position
- [X] T016 Test MDX compatibility by verifying generated files don't contain unescaped HTML tags or malformed markdown that would break Docusaurus parsing
- [X] T017 Create validation script `scripts/validate-godoc-output.sh` that checks: all expected files exist, files have frontmatter, files are non-empty, files are valid markdown

**Checkpoint**: Foundation ready - godoc automation verified and working. User story implementation can now begin in parallel.

---

## Phase 3: User Story 3 - API Reference Lookup (Priority: P2) 🎯 MVP

**Goal**: Enable developers to find complete, accurate API documentation for all public packages with function signatures, parameters, return values, error conditions, and usage examples. Focus on ensuring godoc automation works properly and generates comprehensive API reference.

**Independent Test**: Have a developer look up API documentation for any package (e.g., voice/stt, agents, llms) and find complete information about public functions, including parameters, return values, error conditions, and usage examples. Verify generated docs are accurate and up-to-date.

### Godoc Script Verification and Testing

- [X] T018 [US3] Test godoc generation for core package by running `gomarkdoc --format github github.com/lookatitude/beluga-ai/pkg/core` and verifying output contains function signatures and documentation
- [X] T019 [P] [US3] Test godoc generation for each main package individually: agents, chatmodels, config, embeddings, llms, memory, monitoring, orchestration, prompts, retrievers, schema, server, vectorstores by running gomarkdoc for each and verifying output
- [X] T020 [P] [US3] Test godoc generation for each LLM provider package: anthropic, bedrock, cohere, ollama, openai by running gomarkdoc for each and verifying output
- [X] T021 [P] [US3] Test godoc generation for each voice package: stt, tts, vad, turndetection, transport, noise, session by running gomarkdoc for each and verifying output
- [X] T022 [US3] Test tools package godoc generation by running `gomarkdoc --format github github.com/lookatitude/beluga-ai/pkg/agents/tools` and verifying output
- [X] T023 [US3] Compare generated documentation file sizes: verify each generated `.md` file in `website/docs/api/packages/` has at least 100 lines of content (indicating substantial documentation)
- [X] T024 [US3] Verify generated documentation includes function signatures by checking at least 10 generated files contain Go function signatures with proper formatting
- [X] T025 [US3] Verify generated documentation includes parameter descriptions by checking generated files contain parameter documentation for functions with parameters

### Package Coverage Research and Gap Analysis

- [X] T026 [US3] Research all packages in `pkg/` directory by running `find pkg -type d -name "*.go" -exec dirname {} \; | sort -u` and comparing with script package list to identify missing packages
- [X] T027 [US3] Identify sub-packages that may need documentation by checking for subdirectories in main packages (e.g., `pkg/agents/providers/`, `pkg/embeddings/providers/`) that contain public Go code
- [X] T028 [US3] Research embedding provider packages by listing `pkg/embeddings/providers/` directory and identifying all provider sub-packages that should be documented
- [X] T029 [US3] Research vectorstore provider packages by listing `pkg/vectorstores/providers/` directory (if exists) and identifying provider sub-packages that should be documented
- [X] T030 [US3] Research config provider packages by listing `pkg/config/providers/` directory and identifying provider sub-packages that should be documented
- [X] T031 [US3] Research chatmodel provider packages by listing `pkg/chatmodels/providers/` directory and identifying provider sub-packages that should be documented
- [X] T032 [US3] Create comprehensive package inventory document `docs/API_PACKAGE_INVENTORY.md` listing all packages, sub-packages, and their documentation status (documented/not documented)

### Script Improvements

- [X] T033 [US3] Add missing packages to `scripts/generate-docs.sh` based on gap analysis: update PACKAGES array (lines 135-150) to include any missing main packages identified in research
- [ ] T034 [US3] Add embedding provider packages to `scripts/generate-docs.sh` by creating EMBEDDING_PROVIDERS array and generation loop similar to LLM_PROVIDERS (if providers exist and should be documented)
- [ ] T035 [US3] Add vectorstore provider packages to `scripts/generate-docs.sh` by creating VECTORSTORE_PROVIDERS array and generation loop (if providers exist and should be documented)
- [ ] T036 [US3] Add config provider packages to `scripts/generate-docs.sh` by creating CONFIG_PROVIDERS array and generation loop (if providers exist and should be documented)
- [ ] T037 [US3] Add chatmodel provider packages to `scripts/generate-docs.sh` by creating CHATMODEL_PROVIDERS array and generation loop (if providers exist and should be documented)
- [X] T038 [US3] Improve script error handling in `scripts/generate-docs.sh` function `generate_package_docs()`: add better error messages, log failures to a file, continue processing other packages on failure
- [X] T039 [US3] Add validation step to `scripts/generate-docs.sh` after generation: verify all expected output files exist, check file sizes, validate markdown syntax
- [X] T040 [US3] Add summary report to `scripts/generate-docs.sh` at end: print count of packages processed, count of files generated, list any failures or warnings
- [X] T041 [US3] Improve MDX compatibility fixes in `scripts/generate-docs.sh` lines 77-118: test with actual generated content, add more edge case handling for HTML tags, verify Python script handles all cases correctly

### Godoc Comment Completeness Research

- [X] T042 [US3] Research godoc comment coverage for core package by running `go doc pkg/core` and checking which exported functions/types have godoc comments
- [X] T043 [P] [US3] Research godoc comment coverage for each main package: run `go doc` for agents, chatmodels, config, embeddings, llms, memory, monitoring, orchestration, prompts, retrievers, schema, server, vectorstores and identify functions/types missing godoc comments
- [X] T044 [P] [US3] Research godoc comment coverage for voice packages: run `go doc` for stt, tts, vad, turndetection, transport, noise, session and identify missing godoc comments
- [X] T045 [US3] Create godoc completeness report `docs/GODOC_COVERAGE_REPORT.md` listing all packages, exported functions/types, and their godoc comment status (has comment/missing comment)
- [X] T046 [US3] Identify critical missing godoc comments by reviewing coverage report and prioritizing public API functions that are most commonly used (e.g., New* functions, main interface methods)

### Website API Reference Updates

- [X] T047 [US3] Regenerate all API documentation by running `scripts/generate-docs.sh` and verifying all packages are processed successfully
- [X] T048 [US3] Verify generated files are properly formatted for Docusaurus by checking frontmatter, markdown syntax, and MDX compatibility in `website/docs/api/packages/` directory
- [X] T049 [US3] Test Docusaurus build with regenerated docs by running `cd website && npm run build` and verifying no errors related to API documentation files
- [X] T050 [US3] Verify API reference navigation in `website/sidebars.js` includes all generated packages: check that all packages in `website/docs/api/packages/` are referenced in sidebar configuration
- [X] T051 [US3] Add missing packages to `website/sidebars.js` API Reference section if any new packages were added to documentation generation
- [ ] T052 [US3] Test API reference pages render correctly by starting Docusaurus dev server (`cd website && npm start`) and manually checking at least 5 API reference pages in browser
- [ ] T053 [US3] Verify cross-references in generated API docs work correctly by checking that package links (e.g., `[core](../core)`) resolve correctly in Docusaurus
- [X] T054 [US3] Add package descriptions to API reference index page `website/docs/api/index.md` listing all documented packages with brief descriptions and links

### Documentation Quality Enhancement

- [ ] T055 [US3] Review generated documentation for core package `website/docs/api/packages/core.md` and manually enhance with: usage examples, cross-references to related packages, common patterns
- [ ] T056 [P] [US3] Review and enhance generated documentation for main packages: add usage examples, cross-references, and common patterns to agents.md, chatmodels.md, config.md, embeddings.md, llms.md, memory.md, monitoring.md, orchestration.md, prompts.md, retrievers.md, schema.md, server.md, vectorstores.md
- [ ] T057 [P] [US3] Review and enhance generated documentation for LLM provider packages: add provider-specific examples, configuration details, and cross-references to anthropic.md, bedrock.md, cohere.md, ollama.md, openai.md in `website/docs/api/packages/llms/`
- [ ] T058 [P] [US3] Review and enhance generated documentation for voice packages: add component-specific examples, integration patterns, and cross-references to stt.md, tts.md, vad.md, turndetection.md, transport.md, noise.md, session.md in `website/docs/api/packages/voice/`
- [ ] T059 [US3] Add usage examples section to each API reference page by creating example code snippets showing common usage patterns for each package
- [ ] T060 [US3] Add "See Also" sections to API reference pages with cross-references to: related packages, concepts documentation, example code, tutorials

**Checkpoint**: At this point, User Story 3 should be fully functional and testable independently. Developers can find complete API documentation for any package with accurate, up-to-date information.

---

## Phase 4: User Story 1 - New Developer Onboarding (Priority: P1)

**Goal**: Enable new developers to complete onboarding journey (homepage → installation → first example → first agent) in under 30 minutes without external help. Ensure API reference is accessible and helpful during onboarding.

**Independent Test**: Have a new developer (who has never used Beluga AI) follow website documentation from homepage through installation, first example, and first agent creation. Measure time and verify completion in under 30 minutes.

### API Reference Integration in Onboarding

- [X] T061 [US1] Add API reference links to installation guide `website/docs/getting-started/installation.md` linking to relevant API packages (core, config, schema) that developers will encounter
- [X] T062 [US1] Add API reference links to first example documentation `website/docs/getting-started/first-example.md` linking to LLM API reference pages
- [X] T063 [US1] Add API reference links to first agent tutorial `website/docs/getting-started/tutorials/first-agent.md` linking to agents and tools API reference pages
- [X] T064 [US1] Verify API reference is discoverable from homepage by checking that `website/docs/intro.md` includes a link to API reference section
- [X] T065 [US1] Test onboarding journey includes API reference access: verify a new developer can find API docs when needed during onboarding without getting lost

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently. New developers can complete onboarding in under 30 minutes with access to API reference when needed.

---

## Phase 5: User Story 2 - Voice Agents Feature Discovery (Priority: P1)

**Goal**: Enable developers to find, understand, and implement Voice Agents feature with comprehensive documentation. Ensure voice package API documentation is complete and accessible.

**Independent Test**: Have a developer who understands basic Beluga AI concepts follow voice agent documentation to create a working voice-enabled agent. Verify they understand all voice components and their relationships, including API details.

### Voice API Reference Integration

- [X] T066 [US2] Verify voice API reference pages are linked from voice overview page `website/docs/voice/index.md` with links to all 7 voice component API pages (stt.md, tts.md, vad.md, turndetection.md, transport.md, noise.md, session.md)
- [X] T067 [US2] Add API reference links to each voice component documentation page (stt.md, tts.md, vad.md, turndetection.md, transport.md, noise.md, session.md in `website/docs/voice/`) linking to corresponding API reference pages
- [X] T068 [US2] Verify voice API documentation includes provider information by checking that generated API docs mention provider implementations (e.g., Deepgram, Google, Azure for STT)
- [X] T069 [US2] Enhance voice API reference pages with voice-specific examples showing how to use each voice component API in practice

**Checkpoint**: At this point, User Story 2 should be fully functional and testable independently. Developers can find and understand all Voice Agents components with complete API documentation.

---

## Phase 6: User Story 4 - Example-Based Learning (Priority: P2)

**Goal**: Enable developers to find, run, and understand all examples. Ensure examples reference correct API documentation.

**Independent Test**: Have a developer find an example relevant to their use case, run it successfully, and understand how to modify it for their needs. Verify examples link to relevant API documentation.

### Example-API Documentation Integration

- [X] T070 [US4] Add API reference links to example documentation pages: for each example in `website/docs/examples/`, add links to relevant API reference pages used in that example
- [X] T071 [US4] Verify example code comments reference API documentation by checking that example code includes comments with links to API reference (e.g., `// See API docs: [agents](../api/packages/agents)`)
- [X] T072 [US4] Test that examples and API docs stay synchronized: verify example code uses APIs that match current API documentation

**Checkpoint**: At this point, User Story 4 should be fully functional and testable independently. Developers can find, run, and understand examples with proper API documentation references.

---

## Phase 7: User Story 5 - Advanced Feature Exploration (Priority: P3)

**Goal**: Enable experienced developers to explore advanced features. Ensure advanced documentation references complete API documentation.

**Independent Test**: Have an experienced developer follow advanced documentation to implement a complex use case and understand production deployment considerations. Verify API documentation supports advanced use cases.

### Advanced API Documentation

- [X] T073 [US5] Add API reference links to advanced guides: orchestration-advanced.md, multi-agent-advanced.md, production-deployment.md linking to relevant API packages
- [X] T074 [US5] Verify API documentation includes advanced usage patterns by checking that generated docs or enhancements include examples of advanced configurations and patterns
- [X] T075 [US5] Add extensibility API documentation by ensuring API reference includes information about custom provider, tool, and agent interfaces

**Checkpoint**: At this point, User Story 5 should be fully functional and testable independently. Experienced developers can find comprehensive advanced documentation with complete API reference.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories and ensure overall documentation quality

### CI/CD Integration

- [X] T076 Create GitHub Actions workflow file `.github/workflows/generate-api-docs.yml` that runs `scripts/generate-docs.sh` on every push to main branch and commits generated docs
- [X] T077 Add documentation generation check to existing CI/CD pipeline by adding step to verify `scripts/generate-docs.sh` runs successfully and generated docs are up-to-date
- [X] T078 Configure automated documentation updates by setting up workflow to: run on code changes to `pkg/` directory, generate docs, create PR if docs changed, or commit directly to main
- [ ] T079 Add documentation generation to pre-commit hooks (optional) by creating hook that runs `scripts/generate-docs.sh` and fails if docs are out of date
- [X] T080 Test CI/CD integration by making a code change that affects godoc comments, pushing to branch, and verifying workflow generates updated documentation

### Documentation Validation and Quality Assurance

- [X] T081 Run comprehensive validation: execute `scripts/validate-godoc-output.sh` (from T017) and verify all checks pass for generated documentation
- [X] T082 Verify all generated API documentation files are valid markdown by running markdown linter or validator on all files in `website/docs/api/packages/`
- [X] T083 Test Docusaurus build with all generated docs by running `cd website && npm run build` and verifying no errors or warnings related to API documentation
- [X] T084 Verify search functionality indexes API documentation by testing Docusaurus search with queries like "NewAgent", "GenerateSpeech", "Invoke" and confirming API reference pages appear in results
- [X] T085 Audit API documentation completeness: verify 100% of public API functions have documentation (either from godoc or manual enhancement) per SC-002 from spec
- [X] T086 Create API documentation quality report `docs/API_DOC_QUALITY_REPORT.md` summarizing: packages documented, functions documented, examples added, cross-references added, gaps identified

### Cross-References and Navigation

- [X] T087 Add cross-references from concept pages to API reference: update `website/docs/concepts/` pages to link to relevant API reference pages
- [X] T088 Add cross-references from API reference to concepts: update API reference pages to link back to concept pages where appropriate
- [X] T089 Verify all cross-references are valid by running link validation script on all documentation pages and fixing broken links
- [X] T090 Test navigation depth: verify API reference pages are accessible within 3 clicks from homepage per SC-007

### Final Validation

- [X] T091 Run quickstart.md validation scenarios from `specs/001-complete-documentation/quickstart.md` focusing on API reference lookup scenarios
- [X] T092 Test API reference lookup user story: have a developer look up API documentation for any package and verify they find complete information
- [X] T093 Verify all success criteria related to API documentation (SC-002, SC-008) are met through testing and validation
- [X] T094 Create final API documentation status report summarizing: packages documented, script improvements made, CI/CD integration status, quality metrics, remaining gaps

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-7)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Phase 8)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 3 (P2) - API Reference**: Can start after Foundational (Phase 2) - No dependencies on other stories (MVP for godoc automation)
- **User Story 1 (P1) - Onboarding**: Can start after Foundational (Phase 2) - May reference US3 API docs but should be independently testable
- **User Story 2 (P1) - Voice Agents**: Can start after Foundational (Phase 2) - May reference US3 API docs but should be independently testable
- **User Story 4 (P2) - Examples**: Can start after Foundational (Phase 2) - May reference US3 API docs but should be independently testable
- **User Story 5 (P3) - Advanced Features**: Can start after Foundational (Phase 2) - May reference US3 API docs but should be independently testable

### Within Each User Story

- Script verification before improvements
- Package research before script updates
- Script improvements before regeneration
- Regeneration before website updates
- Website updates before enhancement
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- All package verification tasks marked [P] can run in parallel (different packages)
- All package research tasks marked [P] can run in parallel (different packages)
- All documentation enhancement tasks marked [P] can run in parallel (different files)
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 3 (Package Research)

```bash
# Launch all package research tasks in parallel (different packages):
Task: "Research godoc comment coverage for agents package"
Task: "Research godoc comment coverage for chatmodels package"
Task: "Research godoc comment coverage for config package"
Task: "Research godoc comment coverage for embeddings package"
# ... (all other packages)
```

---

## Implementation Strategy

### MVP First (User Story 3 - API Reference)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 3 (API Reference with godoc automation)
4. **STOP and VALIDATE**: Test User Story 3 independently - verify developers can find complete API documentation
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 3 → Test independently → Deploy/Demo (MVP - API Reference working!)
3. Add User Story 1 → Test independently → Deploy/Demo
4. Add User Story 2 → Test independently → Deploy/Demo
5. Add User Story 4 → Test independently → Deploy/Demo
6. Add User Story 5 → Test independently → Deploy/Demo
7. Add Polish phase → Final validation → Deploy
8. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 3 (API Reference - godoc automation)
   - Developer B: User Story 1 (Onboarding)
   - Developer C: User Story 2 (Voice Agents)
   - Developer D: User Story 4 (Examples)
   - Developer E: User Story 5 (Advanced Features)
3. Stories complete and integrate independently
4. All developers: Polish phase (CI/CD, validation, cross-references)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- All tasks include specific file paths for clarity
- Verify script works before improving it
- Research gaps before filling them
- Test generated docs before enhancing them
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence

---

## Task Summary

**Total Tasks**: 94 tasks

**Breakdown by Phase**:
- Phase 1 (Setup): 7 tasks
- Phase 2 (Foundational): 10 tasks
- Phase 3 (User Story 3 - API Reference): 43 tasks (MVP focus)
- Phase 4 (User Story 1 - Onboarding): 5 tasks
- Phase 5 (User Story 2 - Voice Agents): 4 tasks
- Phase 6 (User Story 4 - Examples): 3 tasks
- Phase 7 (User Story 5 - Advanced Features): 3 tasks
- Phase 8 (Polish & Cross-Cutting): 19 tasks

**Parallel Opportunities**: 
- ~60 tasks can run in parallel (marked with [P])
- All package verification tasks (different packages)
- All package research tasks (different packages)
- All documentation enhancement tasks (different files)

**Estimated MVP Scope**: Phases 1-3 (60 tasks) - API Reference with godoc automation complete

**Focus Areas**:
- Godoc script verification and improvement
- Package coverage research and gap analysis
- Documentation quality enhancement
- CI/CD integration for automatic updates
- Website API reference updates
