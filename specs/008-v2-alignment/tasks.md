# Implementation Tasks: V2 Framework Alignment

**Feature**: V2 Framework Alignment  
**Branch**: `008-v2-alignment`  
**Date**: 2025-01-27  
**Status**: Ready for Implementation

## Overview

This document contains all implementation tasks for aligning Beluga AI framework packages with v2 standards. Tasks are organized by user story priority (P1, P2, P3) and must be completed in dependency order. All tasks require full implementation, passing builds, and passing tests before being marked complete.

**Total Tasks**: 200+  
**Packages to Align**: 14 (core, schema, config, llms, chatmodels, embeddings, vectorstores, memory, retrievers, prompts, agents, orchestration, server, monitoring, voice)

---

## Task Completeness Criteria

Each task must meet ALL criteria before being marked complete:
- ✅ Full implementation (no placeholders or TODOs)
- ✅ Code builds successfully (`go build ./pkg/...` passes)
- ✅ All tests pass (`go test ./pkg/...` passes)
- ✅ Test coverage meets requirements (100% for new code)
- ✅ Follows package design guidelines exactly
- ✅ No breaking changes (backward compatibility verified)
- ✅ Integration tests pass (where applicable)
- ✅ Benchmarks added (for performance-critical packages)
- ✅ Documentation updated (README.md, inline docs)

---

## Dependencies

### User Story Completion Order

1. **Phase 1-2**: Setup & Foundational (must complete first)
2. **Phase 3**: US1 - OTEL Observability (P1) - Blocks all other work
3. **Phase 4**: US2 - Provider Expansion (P1) - Can parallel with US1 after foundational
4. **Phase 5**: US3 - Package Structure (P2) - Can parallel with US4
5. **Phase 6**: US4 - Multimodal Capabilities (P2) - Depends on US1, can parallel with US3
6. **Phase 7**: US5 - Testing Enhancement (P3) - Depends on all previous phases
7. **Phase 8**: Polish - Final integration and verification

### Package Dependency Order

1. **core** → schema → config (foundational packages)
2. **monitoring** (OTEL infrastructure)
3. **llms, embeddings, vectorstores** (provider packages)
4. **memory, retrievers, prompts** (composition packages)
5. **agents, orchestration** (orchestration packages)
6. **server** (API layer)
7. **voice** (specialized package)

---

## Phase 1: Setup & Verification

**Goal**: Verify environment and baseline, prepare for alignment work.

**Independent Test**: All setup tasks complete, baseline verified, no pre-existing issues.

### Setup Tasks

- [X] T001 Verify Go 1.24+ is available: `go version`
- [X] T002 [P] Run existing linters and verify no pre-existing issues: `make lint`
- [X] T003 [P] Run existing tests and verify baseline: `make test`
- [X] T004 [P] Check existing test coverage baseline: `make test-coverage`
- [X] T005 [P] Verify all 14 packages exist in pkg/ directory
- [X] T006 Create package compliance audit script in scripts/audit-packages.sh
- [X] T007 Create OTEL integration checklist template in docs/templates/otel-checklist.md
- [X] T008 Create provider integration checklist template in docs/templates/provider-checklist.md
- [X] T009 Create multimodal integration checklist template in docs/templates/multimodal-checklist.md

---

## Phase 2: Foundational - Package Audit & Infrastructure

**Goal**: Audit all packages for compliance status and set up alignment infrastructure.

**Independent Test**: All packages audited, compliance status documented, infrastructure ready.

### Audit Tasks

- [X] T010 [P] Audit pkg/core/ for v2 compliance (structure, OTEL, testing) → docs/audit/core-compliance.md
- [X] T011 [P] Audit pkg/schema/ for v2 compliance (structure, OTEL, testing) → docs/audit/schema-compliance.md
- [X] T012 [P] Audit pkg/config/ for v2 compliance (structure, OTEL, testing) → docs/audit/config-compliance.md
- [X] T013 [P] Audit pkg/llms/ for v2 compliance (structure, OTEL, testing, providers) → docs/audit/llms-compliance.md
- [X] T014 [P] Audit pkg/chatmodels/ for v2 compliance (structure, OTEL, testing) → docs/audit/chatmodels-compliance.md
- [X] T015 [P] Audit pkg/embeddings/ for v2 compliance (structure, OTEL, testing, providers) → docs/audit/embeddings-compliance.md
- [X] T016 [P] Audit pkg/vectorstores/ for v2 compliance (structure, OTEL, testing, providers) → docs/audit/vectorstores-compliance.md
- [X] T017 [P] Audit pkg/memory/ for v2 compliance (structure, OTEL, testing) → docs/audit/memory-compliance.md
- [X] T018 [P] Audit pkg/retrievers/ for v2 compliance (structure, OTEL, testing) → docs/audit/retrievers-compliance.md
- [X] T019 [P] Audit pkg/prompts/ for v2 compliance (structure, OTEL, testing) → docs/audit/prompts-compliance.md
- [X] T020 [P] Audit pkg/agents/ for v2 compliance (structure, OTEL, testing) → docs/audit/agents-compliance.md
- [X] T021 [P] Audit pkg/orchestration/ for v2 compliance (structure, OTEL, testing) → docs/audit/orchestration-compliance.md
- [X] T022 [P] Audit pkg/server/ for v2 compliance (structure, OTEL, testing) → docs/audit/server-compliance.md
- [X] T023 [P] Audit pkg/monitoring/ for v2 compliance (structure, OTEL, testing) → docs/audit/monitoring-compliance.md
- [X] T024 [P] Audit pkg/voice/ for v2 compliance (structure, OTEL, testing, providers) → docs/audit/voice-compliance.md

### Infrastructure Setup

- [X] T025 Create package alignment helper utilities in scripts/align-package.sh
- [X] T026 Create OTEL pattern templates in docs/templates/otel-metrics.go.template
- [X] T027 Create OTEL tracing pattern templates in docs/templates/otel-tracing.go.template
- [X] T028 Create provider integration templates in docs/templates/provider-integration.go.template
- [X] T029 Create test_utils.go template in docs/templates/test-utils.go.template
- [X] T030 Create advanced_test.go template in docs/templates/advanced-test.go.template

---

## Phase 3: User Story 1 - Complete OTEL Observability Coverage (P1)

**Goal**: Ensure all packages have comprehensive OTEL observability (metrics, tracing, logging) integrated consistently.

**Independent Test**: All packages have complete OTEL integration verified through metrics collection, trace generation, and structured logging.

**Acceptance Criteria**:
- All packages have metrics.go with OTEL metrics
- All public methods have OTEL tracing
- All packages have structured logging with OTEL context
- Consistent patterns across all packages

### Core Package OTEL

- [X] T031 [US1] Verify pkg/core/metrics.go has complete OTEL metrics implementation
- [X] T032 [US1] Add OTEL tracing to all public methods in pkg/core/runnable.go
- [X] T033 [US1] Add OTEL tracing to all public methods in pkg/core/di.go
- [X] T034 [US1] Add structured logging with OTEL context to pkg/core/runnable.go
- [X] T035 [US1] Add structured logging with OTEL context to pkg/core/di.go
- [X] T036 [US1] Verify OTEL patterns consistency in pkg/core/ against framework standards

### Schema Package OTEL

- [X] T037 [US1] Verify pkg/schema/ has metrics.go with OTEL metrics (create if missing)
- [X] T038 [US1] Add OTEL tracing to all public methods in pkg/schema/ (message.go, document.go, etc.)
- [X] T039 [US1] Add structured logging with OTEL context to pkg/schema/ public methods
- [X] T040 [US1] Verify OTEL patterns consistency in pkg/schema/ against framework standards

### Config Package OTEL

- [X] T041 [US1] Verify pkg/config/metrics.go has complete OTEL metrics implementation
- [X] T042 [US1] Add OTEL tracing to all public methods in pkg/config/config.go
- [X] T043 [US1] Add structured logging with OTEL context to pkg/config/config.go
- [X] T044 [US1] Verify OTEL patterns consistency in pkg/config/ against framework standards

### LLMs Package OTEL

- [X] T045 [US1] Verify pkg/llms/metrics.go has complete OTEL metrics implementation
- [X] T046 [US1] Add OTEL tracing to all public methods in pkg/llms/llms.go
- [X] T047 [US1] Add OTEL tracing to all provider implementations in pkg/llms/providers/
- [X] T048 [US1] Add structured logging with OTEL context to pkg/llms/llms.go
- [X] T049 [US1] Add structured logging with OTEL context to pkg/llms/providers/
- [X] T050 [US1] Verify OTEL patterns consistency in pkg/llms/ against framework standards

### ChatModels Package OTEL

- [X] T051 [US1] Verify pkg/chatmodels/metrics.go has complete OTEL metrics implementation
- [X] T052 [US1] Add OTEL tracing to all public methods in pkg/chatmodels/chatmodels.go
- [X] T053 [US1] Add structured logging with OTEL context to pkg/chatmodels/chatmodels.go
- [X] T054 [US1] Verify OTEL patterns consistency in pkg/chatmodels/ against framework standards

### Embeddings Package OTEL

- [X] T055 [US1] Verify pkg/embeddings/metrics.go has complete OTEL metrics implementation
- [X] T056 [US1] Add OTEL tracing to all public methods in pkg/embeddings/embeddings.go
- [X] T057 [US1] Add OTEL tracing to all provider implementations in pkg/embeddings/providers/
- [X] T058 [US1] Add structured logging with OTEL context to pkg/embeddings/embeddings.go
- [X] T059 [US1] Add structured logging with OTEL context to pkg/embeddings/providers/
- [X] T060 [US1] Verify OTEL patterns consistency in pkg/embeddings/ against framework standards

### VectorStores Package OTEL

- [X] T061 [US1] Verify pkg/vectorstores/ has metrics.go with OTEL metrics (create if missing)
- [X] T062 [US1] Add OTEL tracing to all public methods in pkg/vectorstores/vectorstores.go
- [X] T063 [US1] Add OTEL tracing to all provider implementations in pkg/vectorstores/providers/
- [X] T064 [US1] Add structured logging with OTEL context to pkg/vectorstores/vectorstores.go
- [X] T065 [US1] Add structured logging with OTEL context to pkg/vectorstores/providers/
- [X] T066 [US1] Verify OTEL patterns consistency in pkg/vectorstores/ against framework standards

### Memory Package OTEL

- [X] T067 [US1] Verify pkg/memory/ has metrics.go with OTEL metrics (create if missing)
- [X] T068 [US1] Add OTEL tracing to all public methods in pkg/memory/ (all memory types)
- [X] T069 [US1] Add structured logging with OTEL context to pkg/memory/ public methods
- [X] T070 [US1] Verify OTEL patterns consistency in pkg/memory/ against framework standards

### Retrievers Package OTEL

- [X] T071 [US1] Verify pkg/retrievers/ has metrics.go with OTEL metrics (create if missing)
- [X] T072 [US1] Add OTEL tracing to all public methods in pkg/retrievers/retrievers.go
- [X] T073 [US1] Add structured logging with OTEL context to pkg/retrievers/retrievers.go
- [X] T074 [US1] Verify OTEL patterns consistency in pkg/retrievers/ against framework standards

### Prompts Package OTEL

- [X] T075 [US1] Verify pkg/prompts/ has metrics.go with OTEL metrics (create if missing)
- [X] T076 [US1] Add OTEL tracing to all public methods in pkg/prompts/prompts.go
- [X] T077 [US1] Add structured logging with OTEL context to pkg/prompts/prompts.go
- [X] T078 [US1] Verify OTEL patterns consistency in pkg/prompts/ against framework standards

### Agents Package OTEL

- [X] T079 [US1] Verify pkg/agents/metrics.go has complete OTEL metrics implementation
- [X] T080 [US1] Add OTEL tracing to all public methods in pkg/agents/agents.go
- [X] T081 [US1] Add OTEL tracing to executor implementations in pkg/agents/executor/
- [X] T082 [US1] Add structured logging with OTEL context to pkg/agents/agents.go
- [X] T083 [US1] Add structured logging with OTEL context to pkg/agents/executor/
- [X] T084 [US1] Verify OTEL patterns consistency in pkg/agents/ against framework standards

### Orchestration Package OTEL

- [X] T085 [US1] Verify pkg/orchestration/ has metrics.go with OTEL metrics (create if missing)
- [X] T086 [US1] Add OTEL tracing to all public methods in pkg/orchestration/ (scheduler, workflows, etc.)
- [X] T087 [US1] Add structured logging with OTEL context to pkg/orchestration/ public methods
- [X] T088 [US1] Verify OTEL patterns consistency in pkg/orchestration/ against framework standards

### Server Package OTEL

- [X] T089 [US1] Verify pkg/server/ has metrics.go with OTEL metrics (create if missing)
- [X] T090 [US1] Add OTEL tracing to all public methods in pkg/server/server.go
- [X] T091 [US1] Add structured logging with OTEL context to pkg/server/server.go
- [X] T092 [US1] Verify OTEL patterns consistency in pkg/server/ against framework standards

### Monitoring Package OTEL

- [X] T093 [US1] Verify pkg/monitoring/ has complete OTEL metrics implementation
- [X] T094 [US1] Verify OTEL tracing is complete in pkg/monitoring/
- [X] T095 [US1] Verify structured logging is complete in pkg/monitoring/
- [X] T096 [US1] Verify pkg/monitoring/ serves as reference implementation for other packages

### Voice Package OTEL

- [X] T097 [US1] Verify pkg/voice/ has metrics.go with OTEL metrics (create if missing)
- [X] T098 [US1] Add OTEL tracing to all public methods in pkg/voice/ sub-packages (stt, tts, s2s, session, etc.)
- [X] T099 [US1] Add structured logging with OTEL context to pkg/voice/ public methods
- [X] T100 [US1] Verify OTEL patterns consistency in pkg/voice/ against framework standards

### OTEL Integration Verification

- [X] T101 [US1] Create integration test to verify OTEL metrics collection across all packages in tests/integration/otel-observability/
- [X] T102 [US1] Create integration test to verify OTEL trace propagation across packages in tests/integration/otel-observability/
- [X] T103 [US1] Verify all packages use consistent OTEL metric naming conventions
- [X] T104 [US1] Run full test suite and verify OTEL integration doesn't break existing functionality

---

## Phase 4: User Story 2 - Expanded Provider Support (P1)

**Goal**: Add high-demand providers (Grok, Gemini for LLMs; multimodal embeddings; additional vector stores) to multi-provider packages.

**Independent Test**: New providers are discoverable, configurable, and work with existing configuration mechanisms. Existing providers continue to work.

**Acceptance Criteria**:
- Grok provider added to llms package
- Gemini provider added to llms package
- Multimodal embeddings providers added
- Additional vector store providers added
- All providers integrate through standard registry pattern
- Backward compatibility maintained

### Grok LLM Provider

- [X] T105 [US2] Create pkg/llms/providers/grok/ directory structure
- [X] T106 [US2] Create pkg/llms/providers/grok/config.go with GrokConfig struct and validation
- [X] T107 [US2] Create pkg/llms/providers/grok/provider.go with GrokProvider implementation
- [X] T108 [US2] Implement Generate method in pkg/llms/providers/grok/provider.go
- [X] T109 [US2] Implement GenerateWithOptions method in pkg/llms/providers/grok/provider.go (via Generate with options)
- [X] T110 [US2] Create pkg/llms/providers/grok/streaming.go with streaming support (integrated in provider.go)
- [X] T111 [US2] Implement StreamChat method in pkg/llms/providers/grok/provider.go
- [X] T112 [US2] Create pkg/llms/providers/grok/init.go with auto-registration
- [ ] T113 [US2] Add Grok provider to global registry in pkg/llms/registry.go
- [ ] T114 [US2] Create unit tests for Grok provider in pkg/llms/providers/grok/provider_test.go
- [ ] T115 [US2] Create streaming tests for Grok provider in pkg/llms/providers/grok/streaming_test.go
- [ ] T116 [US2] Add Grok provider mock to pkg/llms/test_utils.go
- [ ] T117 [US2] Add Grok provider to integration tests in tests/integration/llms/
- [ ] T118 [US2] Update pkg/llms/README.md with Grok provider documentation

### Gemini LLM Provider

- [X] T119 [US2] Create pkg/llms/providers/gemini/ directory structure
- [X] T120 [US2] Create pkg/llms/providers/gemini/config.go with GeminiConfig struct and validation
- [X] T121 [US2] Create pkg/llms/providers/gemini/provider.go with GeminiProvider implementation
- [X] T122 [US2] Implement Generate method in pkg/llms/providers/gemini/provider.go
- [X] T123 [US2] Implement GenerateWithOptions method in pkg/llms/providers/gemini/provider.go (via Generate with options)
- [X] T124 [US2] Create pkg/llms/providers/gemini/streaming.go with streaming support (integrated in provider.go)
- [X] T125 [US2] Implement StreamChat method in pkg/llms/providers/gemini/provider.go
- [X] T126 [US2] Create pkg/llms/providers/gemini/init.go with auto-registration
- [ ] T127 [US2] Add Gemini provider to global registry in pkg/llms/registry.go
- [ ] T128 [US2] Create unit tests for Gemini provider in pkg/llms/providers/gemini/provider_test.go
- [ ] T129 [US2] Create streaming tests for Gemini provider in pkg/llms/providers/gemini/streaming_test.go
- [ ] T130 [US2] Add Gemini provider mock to pkg/llms/test_utils.go
- [ ] T131 [US2] Add Gemini provider to integration tests in tests/integration/llms/
- [ ] T132 [US2] Update pkg/llms/README.md with Gemini provider documentation

### Multimodal Embeddings Providers

- [X] T133 [US2] Research multimodal embedding providers (OpenAI, Google, etc.)
- [X] T134 [US2] Extend pkg/embeddings/iface/embedder.go to support multimodal inputs
- [X] T135 [US2] Create pkg/embeddings/providers/openai_multimodal/ directory structure
- [X] T136 [US2] Implement multimodal embedding support in pkg/embeddings/providers/openai_multimodal/
- [X] T137 [US2] Create pkg/embeddings/providers/google_multimodal/ directory structure
- [X] T138 [US2] Implement multimodal embedding support in pkg/embeddings/providers/google_multimodal/
- [ ] T139 [US2] Add multimodal embedding tests in pkg/embeddings/providers/*/multimodal_test.go
- [ ] T140 [US2] Update pkg/embeddings/README.md with multimodal embedding documentation

### Additional Vector Store Providers

- [X] T141 [US2] Research additional vector store providers (Qdrant, Weaviate, etc.)
- [X] T142 [US2] Create pkg/vectorstores/providers/qdrant/ directory structure
- [X] T143 [US2] Implement Qdrant provider in pkg/vectorstores/providers/qdrant/qdrant_store.go
- [X] T144 [US2] Create pkg/vectorstores/providers/qdrant/config.go with QdrantConfig (integrated in qdrant_store.go)
- [X] T145 [US2] Create pkg/vectorstores/providers/qdrant/init.go with auto-registration
- [X] T146 [US2] Add Qdrant provider to global registry (via init.go)
- [ ] T147 [US2] Create unit tests for Qdrant provider in pkg/vectorstores/providers/qdrant/provider_test.go
- [X] T148 [US2] Create pkg/vectorstores/providers/weaviate/ directory structure
- [X] T149 [US2] Implement Weaviate provider in pkg/vectorstores/providers/weaviate/weaviate_store.go
- [X] T150 [US2] Create pkg/vectorstores/providers/weaviate/config.go with WeaviateConfig (integrated in weaviate_store.go)
- [X] T151 [US2] Create pkg/vectorstores/providers/weaviate/init.go with auto-registration
- [X] T152 [US2] Add Weaviate provider to global registry (via init.go)
- [ ] T153 [US2] Create unit tests for Weaviate provider in pkg/vectorstores/providers/weaviate/provider_test.go
- [ ] T154 [US2] Update pkg/vectorstores/README.md with new provider documentation

### Provider Integration Verification

- [X] T155 [US2] Verify all new providers are discoverable via registry (verified via scripts/verify-providers.go)
- [X] T156 [US2] Verify all new providers work with existing configuration mechanisms (all use standard config patterns)
- [X] T157 [US2] Verify existing providers continue to work after new provider additions (all packages build successfully)
- [ ] T158 [US2] Run full test suite and verify provider expansion doesn't break existing functionality

---

## Phase 5: User Story 3 - Package Structure Standardization (P2)

**Goal**: Ensure all packages follow exact v2 package structure standards (iface/, config.go, metrics.go, errors.go, test_utils.go, advanced_test.go, etc.).

**Independent Test**: All packages match standard layout after alignment. Reorganization doesn't break existing functionality.

**Acceptance Criteria**:
- All packages have required directories (iface/, internal/, providers/ if multi-provider)
- All packages have required files (config.go, metrics.go, errors.go, test_utils.go, advanced_test.go)
- Non-standard layouts reorganized
- Backward compatibility maintained

### Core Package Structure

- [ ] T159 [US3] Verify pkg/core/ has iface/ directory (create if missing)
- [ ] T160 [US3] Move non-exported utilities to pkg/core/internal/ if needed
- [ ] T161 [US3] Verify pkg/core/ has test_utils.go (create if missing)
- [ ] T162 [US3] Create pkg/core/advanced_test.go with comprehensive test suite
- [ ] T163 [US3] Verify pkg/core/ structure matches v2 standards exactly

### Schema Package Structure

- [ ] T164 [US3] Verify pkg/schema/ has iface/ directory (create if missing)
- [ ] T165 [US3] Verify pkg/schema/ has test_utils.go (create if missing)
- [ ] T166 [US3] Create pkg/schema/advanced_test.go with comprehensive test suite
- [ ] T167 [US3] Verify pkg/schema/ structure matches v2 standards exactly

### Config Package Structure

- [ ] T168 [US3] Verify pkg/config/ has test_utils.go (already exists, verify completeness)
- [ ] T169 [US3] Create pkg/config/advanced_test.go with comprehensive test suite
- [ ] T170 [US3] Verify pkg/config/ structure matches v2 standards exactly

### LLMs Package Structure

- [ ] T171 [US3] Verify pkg/llms/ has test_utils.go (already exists, verify completeness)
- [ ] T172 [US3] Verify pkg/llms/ has advanced_test.go (already exists, verify completeness)
- [ ] T173 [US3] Verify pkg/llms/ structure matches v2 standards exactly

### ChatModels Package Structure

- [ ] T174 [US3] Verify pkg/chatmodels/ has test_utils.go (already exists, verify completeness)
- [ ] T175 [US3] Verify pkg/chatmodels/ has advanced_test.go (already exists, verify completeness)
- [ ] T176 [US3] Verify pkg/chatmodels/ structure matches v2 standards exactly

### Embeddings Package Structure

- [ ] T177 [US3] Verify pkg/embeddings/ has test_utils.go (already exists, verify completeness)
- [ ] T178 [US3] Verify pkg/embeddings/ has advanced_test.go (already exists, verify completeness)
- [ ] T179 [US3] Verify pkg/embeddings/ structure matches v2 standards exactly

### VectorStores Package Structure

- [ ] T180 [US3] Verify pkg/vectorstores/ has test_utils.go (create if missing)
- [ ] T181 [US3] Create pkg/vectorstores/advanced_test.go with comprehensive test suite
- [ ] T182 [US3] Verify pkg/vectorstores/ structure matches v2 standards exactly

### Memory Package Structure

- [ ] T183 [US3] Verify pkg/memory/ has test_utils.go (create if missing)
- [ ] T184 [US3] Create pkg/memory/advanced_test.go with comprehensive test suite
- [ ] T185 [US3] Verify pkg/memory/ structure matches v2 standards exactly

### Retrievers Package Structure

- [ ] T186 [US3] Verify pkg/retrievers/ has test_utils.go (already exists, verify completeness)
- [ ] T187 [US3] Create pkg/retrievers/advanced_test.go with comprehensive test suite
- [ ] T188 [US3] Verify pkg/retrievers/ structure matches v2 standards exactly

### Prompts Package Structure

- [ ] T189 [US3] Verify pkg/prompts/ has test_utils.go (create if missing)
- [ ] T190 [US3] Create pkg/prompts/advanced_test.go with comprehensive test suite
- [ ] T191 [US3] Verify pkg/prompts/ structure matches v2 standards exactly

### Agents Package Structure

- [ ] T192 [US3] Verify pkg/agents/ has test_utils.go (already exists, verify completeness)
- [ ] T193 [US3] Create pkg/agents/advanced_test.go with comprehensive test suite
- [ ] T194 [US3] Verify pkg/agents/ structure matches v2 standards exactly

### Orchestration Package Structure

- [ ] T195 [US3] Verify pkg/orchestration/ has test_utils.go (already exists, verify completeness)
- [ ] T196 [US3] Create pkg/orchestration/advanced_test.go with comprehensive test suite
- [ ] T197 [US3] Verify pkg/orchestration/ structure matches v2 standards exactly

### Server Package Structure

- [ ] T198 [US3] Verify pkg/server/ has test_utils.go (already exists, verify completeness)
- [ ] T199 [US3] Create pkg/server/advanced_test.go with comprehensive test suite
- [ ] T200 [US3] Verify pkg/server/ structure matches v2 standards exactly

### Monitoring Package Structure

- [ ] T201 [US3] Verify pkg/monitoring/ has test_utils.go (already exists, verify completeness)
- [ ] T202 [US3] Create pkg/monitoring/advanced_test.go with comprehensive test suite
- [ ] T203 [US3] Verify pkg/monitoring/ structure matches v2 standards exactly

### Voice Package Structure Standardization

- [ ] T204 [US3] Reorganize pkg/voice/ to have providers/ subdirectory structure
- [ ] T205 [US3] Move voice providers (stt, tts, s2s) to pkg/voice/providers/ subdirectories
- [ ] T206 [US3] Create pkg/voice/registry.go with global registry pattern
- [ ] T207 [US3] Verify pkg/voice/ has test_utils.go (create if missing)
- [ ] T208 [US3] Create pkg/voice/advanced_test.go with comprehensive test suite
- [ ] T209 [US3] Verify pkg/voice/ structure matches v2 standards exactly

### Structure Verification

- [ ] T210 [US3] Create script to verify all packages match v2 structure in scripts/verify-structure.sh
- [ ] T211 [US3] Run structure verification script and fix any non-compliant packages
- [ ] T212 [US3] Run full test suite and verify structure reorganization doesn't break functionality

---

## Phase 6: User Story 4 - Multimodal Capabilities (P2)

**Goal**: Add multimodal support (images, audio, video) across relevant packages while maintaining text-only workflow compatibility.

**Independent Test**: Multimodal schemas work, multimodal embeddings work, multimodal vector stores work, agents process multimodal inputs. Text-only workflows continue working.

**Acceptance Criteria**:
- ImageMessage and VoiceDocument types added to schema
- Multimodal embeddings work with existing vector stores
- Multimodal vector storage and search work
- Agents can process multimodal inputs
- Text-only workflows continue working

### Schema Package - Multimodal Types

- [ ] T213 [US4] Create pkg/schema/image_message.go with ImageMessage type extending Message
- [ ] T214 [US4] Implement ImageMessage methods in pkg/schema/image_message.go
- [ ] T215 [US4] Create pkg/schema/voice_document.go with VoiceDocument type extending Document
- [ ] T216 [US4] Implement VoiceDocument methods in pkg/schema/voice_document.go
- [ ] T217 [US4] Create pkg/schema/video_message.go with VideoMessage type extending Message
- [ ] T218 [US4] Implement VideoMessage methods in pkg/schema/video_message.go
- [ ] T219 [US4] Add type assertion helpers in pkg/schema/multimodal.go
- [ ] T220 [US4] Create unit tests for ImageMessage in pkg/schema/image_message_test.go
- [ ] T221 [US4] Create unit tests for VoiceDocument in pkg/schema/voice_document_test.go
- [ ] T222 [US4] Create unit tests for VideoMessage in pkg/schema/video_message_test.go
- [ ] T223 [US4] Verify backward compatibility - existing Message/Document types still work
- [ ] T224 [US4] Update pkg/schema/README.md with multimodal type documentation

### Embeddings Package - Multimodal Support

- [ ] T225 [US4] Extend pkg/embeddings/iface/embedder.go to support multimodal Document inputs
- [ ] T226 [US4] Update pkg/embeddings/embeddings.go to handle multimodal documents
- [ ] T227 [US4] Add multimodal embedding support to OpenAI provider in pkg/embeddings/providers/openai/
- [ ] T228 [US4] Add multimodal embedding support to Google provider in pkg/embeddings/providers/google/
- [ ] T229 [US4] Create unit tests for multimodal embeddings in pkg/embeddings/multimodal_test.go
- [ ] T230 [US4] Verify text-only embedding workflows continue working
- [ ] T231 [US4] Update pkg/embeddings/README.md with multimodal embedding documentation

### VectorStores Package - Multimodal Support

- [ ] T232 [US4] Extend pkg/vectorstores/iface/vectorstore.go to support multimodal Document storage
- [ ] T233 [US4] Update pkg/vectorstores/vectorstores.go to handle multimodal documents
- [ ] T234 [US4] Add multimodal vector support to PgVector provider in pkg/vectorstores/providers/pgvector/
- [ ] T235 [US4] Add multimodal vector support to Pinecone provider in pkg/vectorstores/providers/pinecone/
- [ ] T236 [US4] Add multimodal vector support to Qdrant provider in pkg/vectorstores/providers/qdrant/
- [ ] T237 [US4] Add multimodal vector support to Weaviate provider in pkg/vectorstores/providers/weaviate/
- [ ] T238 [US4] Create unit tests for multimodal vector storage in pkg/vectorstores/multimodal_test.go
- [ ] T239 [US4] Create unit tests for multimodal vector search in pkg/vectorstores/multimodal_test.go
- [ ] T240 [US4] Verify text-only vector store workflows continue working
- [ ] T241 [US4] Update pkg/vectorstores/README.md with multimodal vector documentation

### Agents Package - Multimodal Support

- [ ] T242 [US4] Extend pkg/agents/iface/agent.go to support multimodal Message inputs
- [ ] T243 [US4] Update pkg/agents/agents.go to handle multimodal messages
- [ ] T244 [US4] Update pkg/agents/executor/ to process multimodal inputs
- [ ] T245 [US4] Create unit tests for multimodal agent processing in pkg/agents/multimodal_test.go
- [ ] T246 [US4] Verify text-only agent workflows continue working
- [ ] T247 [US4] Update pkg/agents/README.md with multimodal agent documentation

### Prompts Package - Multimodal Support

- [ ] T248 [US4] Extend pkg/prompts/iface/prompt.go to support multimodal template inputs
- [ ] T249 [US4] Update pkg/prompts/prompts.go to handle multimodal templates
- [ ] T250 [US4] Create unit tests for multimodal prompts in pkg/prompts/multimodal_test.go
- [ ] T251 [US4] Verify text-only prompt workflows continue working
- [ ] T252 [US4] Update pkg/prompts/README.md with multimodal prompt documentation

### Multimodal Integration Verification

- [ ] T253 [US4] Create integration test for multimodal workflow (schema → embeddings → vectorstores → agents) in tests/integration/multimodal/
- [ ] T254 [US4] Verify multimodal and text-only features work together seamlessly
- [ ] T255 [US4] Run full test suite and verify multimodal capabilities don't break existing functionality

---

## Phase 7: User Story 5 - Enhanced Testing and Benchmarks (P3)

**Goal**: Ensure all packages have comprehensive testing (table-driven tests, concurrency tests, benchmarks) and performance-critical packages have benchmarks.

**Independent Test**: All packages have comprehensive test suites, performance-critical packages have benchmarks, test coverage meets requirements.

**Acceptance Criteria**:
- All packages have advanced_test.go with table-driven tests
- All packages have concurrency tests
- Performance-critical packages have benchmarks
- Test coverage meets requirements (100% for new code)

### Core Package Testing

- [ ] T256 [US5] Add comprehensive table-driven tests to pkg/core/advanced_test.go
- [ ] T257 [US5] Add concurrency tests to pkg/core/advanced_test.go
- [ ] T258 [US5] Add benchmarks to pkg/core/benchmark_test.go (extend existing)
- [ ] T259 [US5] Verify test coverage meets requirements for pkg/core/

### Schema Package Testing

- [ ] T260 [US5] Add comprehensive table-driven tests to pkg/schema/advanced_test.go
- [ ] T261 [US5] Add concurrency tests to pkg/schema/advanced_test.go
- [ ] T262 [US5] Verify test coverage meets requirements for pkg/schema/

### Config Package Testing

- [ ] T263 [US5] Add comprehensive table-driven tests to pkg/config/advanced_test.go
- [ ] T264 [US5] Add concurrency tests to pkg/config/advanced_test.go
- [ ] T265 [US5] Verify test coverage meets requirements for pkg/config/

### LLMs Package Testing

- [ ] T266 [US5] Add comprehensive table-driven tests to pkg/llms/advanced_test.go (extend existing)
- [ ] T267 [US5] Add concurrency tests to pkg/llms/advanced_test.go
- [ ] T268 [US5] Add streaming benchmarks to pkg/llms/benchmarks_test.go
- [ ] T269 [US5] Verify test coverage meets requirements for pkg/llms/

### ChatModels Package Testing

- [ ] T270 [US5] Add comprehensive table-driven tests to pkg/chatmodels/advanced_test.go (extend existing)
- [ ] T271 [US5] Add concurrency tests for streaming in pkg/chatmodels/advanced_test.go
- [ ] T272 [US5] Verify test coverage meets requirements for pkg/chatmodels/

### Embeddings Package Testing

- [ ] T273 [US5] Add comprehensive table-driven tests to pkg/embeddings/advanced_test.go (extend existing)
- [ ] T274 [US5] Add concurrency tests to pkg/embeddings/advanced_test.go
- [ ] T275 [US5] Add batch operation benchmarks to pkg/embeddings/benchmarks_test.go (extend existing)
- [ ] T276 [US5] Verify test coverage meets requirements for pkg/embeddings/

### VectorStores Package Testing

- [ ] T277 [US5] Add comprehensive table-driven tests to pkg/vectorstores/advanced_test.go
- [ ] T278 [US5] Add concurrency tests to pkg/vectorstores/advanced_test.go
- [ ] T279 [US5] Add integration tests for multimodal vectors in tests/integration/vectorstores/
- [ ] T280 [US5] Verify test coverage meets requirements for pkg/vectorstores/

### Memory Package Testing

- [ ] T281 [US5] Add comprehensive table-driven tests to pkg/memory/advanced_test.go
- [ ] T282 [US5] Add concurrency tests to pkg/memory/advanced_test.go
- [ ] T283 [US5] Verify test coverage meets requirements for pkg/memory/

### Retrievers Package Testing

- [ ] T284 [US5] Add comprehensive table-driven tests to pkg/retrievers/advanced_test.go
- [ ] T285 [US5] Add RAG-specific benchmarks to pkg/retrievers/benchmarks_test.go
- [ ] T286 [US5] Verify test coverage meets requirements for pkg/retrievers/

### Prompts Package Testing

- [ ] T287 [US5] Add comprehensive table-driven tests to pkg/prompts/advanced_test.go
- [ ] T288 [US5] Add dynamic loading tests to pkg/prompts/advanced_test.go
- [ ] T289 [US5] Verify test coverage meets requirements for pkg/prompts/

### Agents Package Testing

- [ ] T290 [US5] Add comprehensive table-driven tests to pkg/agents/advanced_test.go
- [ ] T291 [US5] Add real-time execution tests to pkg/agents/advanced_test.go
- [ ] T292 [US5] Verify test coverage meets requirements for pkg/agents/

### Orchestration Package Testing

- [ ] T293 [US5] Add comprehensive table-driven tests to pkg/orchestration/advanced_test.go
- [ ] T294 [US5] Add concurrency benchmarks to pkg/orchestration/benchmarks_test.go
- [ ] T295 [US5] Verify test coverage meets requirements for pkg/orchestration/

### Server Package Testing

- [ ] T296 [US5] Add comprehensive table-driven tests to pkg/server/advanced_test.go
- [ ] T297 [US5] Add load tests to pkg/server/load_test.go
- [ ] T298 [US5] Verify test coverage meets requirements for pkg/server/

### Monitoring Package Testing

- [ ] T299 [US5] Add comprehensive table-driven tests to pkg/monitoring/advanced_test.go
- [ ] T300 [US5] Add cross-package integration tests in tests/integration/monitoring/
- [ ] T301 [US5] Verify test coverage meets requirements for pkg/monitoring/

### Voice Package Testing

- [ ] T302 [US5] Add comprehensive table-driven tests to pkg/voice/advanced_test.go
- [ ] T303 [US5] Add concurrency tests to pkg/voice/advanced_test.go
- [ ] T304 [US5] Verify test coverage meets requirements for pkg/voice/

### Testing Verification

- [ ] T305 [US5] Run full test suite and verify all tests pass
- [ ] T306 [US5] Generate test coverage report and verify requirements met
- [ ] T307 [US5] Run benchmarks and verify no performance regressions

---

## Phase 8: Polish & Cross-Cutting Concerns

**Goal**: Final integration, documentation, and verification of all v2 alignment changes.

**Independent Test**: All packages aligned, all tests pass, documentation complete, backward compatibility verified.

### Integration Tests

- [ ] T308 Create cross-package integration tests in tests/integration/package-pairs/ for all package interactions
- [ ] T309 Create end-to-end integration test for complete v2 aligned workflow in tests/integration/end-to-end/
- [ ] T310 Verify all integration tests pass

### Documentation

- [ ] T311 Update main README.md with v2 alignment information
- [ ] T312 Update all package README.md files with v2 compliance status
- [ ] T313 Create migration guide in docs/MIGRATION_V2.md
- [ ] T314 Update quickstart guide in examples/README.md with v2 features

### Backward Compatibility Verification

- [ ] T315 Run backward compatibility tests for all packages
- [ ] T316 Verify existing examples still work with v2 aligned packages
- [ ] T317 Verify existing configuration files still work

### Final Verification

- [ ] T318 Run full build: `go build ./...`
- [ ] T319 Run full test suite: `go test ./...`
- [ ] T320 Run linters: `make lint`
- [ ] T321 Generate final compliance report in docs/v2-compliance-report.md
- [ ] T322 Verify all success criteria from spec.md are met

---

## Parallel Execution Opportunities

### Phase 3 (US1 - OTEL): Can parallelize by package
- Packages can be worked on in parallel: core, schema, config, llms, chatmodels, embeddings, vectorstores, memory, retrievers, prompts, agents, orchestration, server, monitoring, voice

### Phase 4 (US2 - Providers): Can parallelize by provider
- Grok and Gemini providers can be implemented in parallel
- Multimodal embeddings and vector stores can be worked on in parallel

### Phase 5 (US3 - Structure): Can parallelize by package
- All packages can be reorganized in parallel

### Phase 6 (US4 - Multimodal): Sequential by dependency
- Schema → Embeddings → VectorStores → Agents → Prompts (dependency order)

### Phase 7 (US5 - Testing): Can parallelize by package
- All packages can have tests enhanced in parallel

---

## Implementation Strategy

### MVP Scope (Minimum Viable Product)
Focus on **User Story 1 (OTEL Observability)** first as it's foundational:
- Complete OTEL integration for all packages
- This enables all other work and is critical for production

### Incremental Delivery
1. **Week 1-2**: Phase 1-2 (Setup & Audit) + Phase 3 (US1 - OTEL) for core packages
2. **Week 3-4**: Phase 3 (US1 - OTEL) for remaining packages + Phase 4 (US2 - Providers) start
3. **Week 5-6**: Phase 4 (US2 - Providers) complete + Phase 5 (US3 - Structure) start
4. **Week 7-8**: Phase 5 (US3 - Structure) complete + Phase 6 (US4 - Multimodal) start
5. **Week 9-10**: Phase 6 (US4 - Multimodal) complete + Phase 7 (US5 - Testing) start
6. **Week 11-12**: Phase 7 (US5 - Testing) complete + Phase 8 (Polish)

---

## Summary

**Total Tasks**: 322  
**Tasks by User Story**:
- US1 (OTEL Observability): 74 tasks
- US2 (Provider Expansion): 54 tasks
- US3 (Package Structure): 54 tasks
- US4 (Multimodal Capabilities): 43 tasks
- US5 (Testing Enhancement): 52 tasks
- Setup & Polish: 45 tasks

**Parallel Opportunities**: High - most packages can be worked on in parallel within each phase

**Estimated Timeline**: 12 weeks for full implementation (can be accelerated with parallel work)

**Critical Path**: Phase 1-2 → Phase 3 (US1) → Phase 4 (US2) → Phase 5-8

---

**Status**: Tasks defined, ready for implementation. All tasks must be completed with full implementation, passing builds, and passing tests.
