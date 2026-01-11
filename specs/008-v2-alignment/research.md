# Research & Technical Decisions: V2 Framework Alignment

**Feature**: V2 Framework Alignment  
**Date**: 2025-01-27  
**Status**: Complete

## Overview

This document consolidates all technical research and decisions made for the v2 framework alignment feature. All decisions are based on existing framework patterns, package design guidelines, and backward compatibility requirements.

---

## Research Area 1: OTEL Integration Patterns

### Research Question
What patterns should be used for adding or completing OTEL observability across packages?

### Findings
- All compliant packages use standardized OTEL patterns from `pkg/monitoring`
- Standard pattern includes:
  - Metrics defined in `metrics.go` using OTEL API
  - Tracing in public methods with span creation
  - Structured logging with context and trace IDs
  - Consistent metric naming conventions

### Decision
Use standardized OTEL patterns from `pkg/monitoring` for all packages. For packages missing OTEL:
1. Add `metrics.go` with OTEL metric definitions
2. Add tracing to all public methods
3. Integrate structured logging with OTEL context

### Rationale
- Consistency with existing framework observability
- Leverages existing `pkg/monitoring` infrastructure
- No new observability infrastructure needed
- Follows mandatory framework patterns

### Alternatives Considered
- Custom metrics implementation: Rejected (violates framework standards)
- Third-party observability: Rejected (OTEL is mandatory)

---

## Research Area 2: Provider Expansion Priorities

### Research Question
Which providers should be prioritized for expansion in multi-provider packages?

### Findings
- High-demand providers identified:
  - **LLMs**: Grok (xAI), Gemini (Google) - both support multimodal
  - **Embeddings**: Multimodal embedding providers (OpenAI, Google, etc.)
  - **Vector Stores**: Additional enterprise vector stores (Qdrant, Weaviate, etc.)
- User feedback indicates strong demand for Grok and Gemini
- Multimodal capabilities are increasingly important

### Decision
**Phase 1 (P1)**: Add Grok and Gemini providers to `pkg/llms`
**Phase 2 (P2)**: Add multimodal embeddings to `pkg/embeddings`
**Phase 3 (P2)**: Add additional vector store providers to `pkg/vectorstores`

### Rationale
- Grok and Gemini are high-demand providers with strong user interest
- Multimodal capabilities align with v2 feature goals
- Incremental approach allows for proper testing and validation
- Follows existing provider integration patterns

### Alternatives Considered
- Adding all providers at once: Rejected (too large scope, risk of quality issues)
- Different provider priorities: Considered but Grok/Gemini have highest demand

---

## Research Area 3: Multimodal Schema Design

### Research Question
How should multimodal schemas (ImageMessage, VoiceDocument) be added to the schema package?

### Findings
- Existing schema package has `Message` type hierarchy
- Backward compatibility is critical (existing code uses current Message types)
- Multimodal data needs new types but should integrate with existing patterns

### Decision
Extend existing `Message` type hierarchy with new multimodal types:
- `ImageMessage`: Extends base Message, includes image data and metadata
- `VoiceDocument`: Extends base Document, includes audio data and metadata
- Maintain 100% backward compatibility (existing Message types unchanged)

### Rationale
- Preserves existing API contracts
- Allows incremental adoption of multimodal features
- Follows Go interface extension patterns
- Aligns with framework composition principles

### Alternatives Considered
- New schema package: Rejected (breaks compatibility, unnecessary complexity)
- Modify existing types: Rejected (breaks backward compatibility)
- Separate multimodal package: Rejected (fragments schema management)

---

## Research Area 4: Package Structure Standardization

### Research Question
How should non-compliant packages be reorganized to match v2 structure?

### Findings
- V2 structure is mandatory: `iface/`, `internal/`, `providers/` (if multi-provider), `config.go`, `metrics.go`, `errors.go`, `test_utils.go`, `advanced_test.go`
- Some packages have files in non-standard locations
- Some packages missing required files (test_utils.go, advanced_test.go)

### Decision
1. Audit all packages against v2 structure
2. Reorganize files to match standard layout (move to appropriate directories)
3. Add missing required files following framework templates
4. Maintain backward compatibility (public APIs unchanged, only internal structure changes)

### Rationale
- Consistency improves maintainability
- Standard structure makes packages easier to understand
- Required files ensure comprehensive testing and observability
- Internal reorganization doesn't break public APIs

### Alternatives Considered
- Grandfather existing structures: Rejected (violates v2 standards)
- Gradual migration: Considered but full alignment is required for v2

---

## Research Area 5: Testing Enhancement Patterns

### Research Question
What patterns should be used for adding comprehensive test suites and benchmarks?

### Findings
- Framework has standardized test patterns:
  - `test_utils.go`: AdvancedMock structs with options pattern
  - `advanced_test.go`: Table-driven tests, concurrency tests, benchmarks
  - Integration tests in `tests/integration/package_pairs/`

### Decision
Use existing framework test templates:
1. Add `test_utils.go` with AdvancedMock patterns (if missing)
2. Add `advanced_test.go` with table-driven tests, concurrency tests (if missing)
3. Add benchmarks for performance-critical packages
4. Add integration tests for cross-package compatibility

### Rationale
- Consistency with existing test patterns
- Leverages proven testing infrastructure
- Ensures comprehensive coverage
- Follows framework testing standards

### Alternatives Considered
- Custom test patterns: Rejected (violates framework standards)
- Minimal testing: Rejected (framework requires comprehensive testing)

---

## Research Area 6: Voice Package Standardization

### Research Question
How should the voice package be standardized to match v2 structure?

### Findings
- Voice package is branch-specific and may have partial compliance
- Other multi-provider packages (llms, embeddings, vectorstores) use `providers/` subdirectory
- Voice sub-packages (stt, tts, etc.) follow similar patterns but may need alignment

### Decision
Standardize voice package to match v2 structure:
1. Organize providers in `providers/` subdirectory (e.g., Deepgram, ElevenLabs)
2. Add comprehensive OTEL metrics (if missing)
3. Add global registry pattern (if missing)
4. Add `advanced_test.go` with comprehensive tests (if missing)
5. Follow same patterns as llms, embeddings, vectorstores

### Rationale
- Consistency with other multi-provider packages
- Enables provider discovery and configuration
- Ensures comprehensive observability
- Aligns with v2 standards

### Alternatives Considered
- Keep existing voice structure: Rejected (violates v2 standards)
- Different structure for voice: Rejected (consistency is key)

---

## Research Area 7: Multimodal Integration Strategy

### Research Question
How should multimodal capabilities be integrated across packages incrementally?

### Findings
- Multimodal support needed in: schema, embeddings, vectorstores, agents, prompts
- Integration should be incremental to manage complexity
- Backward compatibility is critical (text-only workflows must continue working)

### Decision
**Incremental Integration Order**:
1. **Schema package**: Add multimodal message types (ImageMessage, VoiceDocument)
2. **Embeddings package**: Add multimodal embedding support
3. **Vectorstores package**: Add multimodal vector storage and search
4. **Agents package**: Add multimodal input processing
5. **Prompts package**: Add multimodal prompt templates

Each phase maintains backward compatibility - text-only workflows continue working without changes.

### Rationale
- Incremental approach reduces risk
- Each phase builds on previous phases
- Backward compatibility ensures no breaking changes
- Allows users to adopt multimodal features gradually

### Alternatives Considered
- All-at-once integration: Rejected (too complex, high risk)
- Different order: Considered but schema-first makes most sense

---

## Research Area 8: Performance Impact Assessment

### Research Question
What is the expected performance impact of OTEL integration and structural changes?

### Findings
- OTEL has minimal overhead when properly implemented
- Structural changes (file moves) have no runtime impact
- Benchmarks should verify no regressions

### Decision
1. Add OTEL integration following framework patterns (minimal overhead)
2. Use benchmarks to verify no performance regressions
3. Add benchmarks for performance-critical packages (llms, embeddings, vectorstores, agents)
4. Monitor performance during alignment

### Rationale
- Framework patterns are optimized for performance
- Benchmarks provide objective measurement
- Performance is critical for production deployments
- Framework already uses OTEL successfully

### Alternatives Considered
- Skip performance verification: Rejected (performance is critical)
- Custom performance monitoring: Rejected (OTEL is sufficient)

---

## Summary of Decisions

| Research Area | Decision | Rationale |
|--------------|----------|-----------|
| OTEL Integration | Use standardized patterns from pkg/monitoring | Consistency, leverages existing infrastructure |
| Provider Expansion | Grok/Gemini first, then multimodal embeddings | High demand, incremental approach |
| Multimodal Schema | Extend existing Message types | Backward compatibility, incremental adoption |
| Package Structure | Reorganize to match v2 exactly | Consistency, maintainability |
| Testing Enhancement | Use framework test templates | Consistency, comprehensive coverage |
| Voice Standardization | Match other multi-provider packages | Consistency with llms, embeddings, vectorstores |
| Multimodal Integration | Incremental: schema → embeddings → vectorstores → agents → prompts | Risk reduction, backward compatibility |
| Performance Impact | Verify with benchmarks, expect minimal impact | Performance critical, framework patterns optimized |

---

## Open Questions Resolved

All technical questions have been resolved through research and framework pattern analysis. No outstanding clarifications remain.

---

## References

- `docs/package_design_patterns.md` - Framework package design patterns
- `pkg/monitoring/` - OTEL observability patterns
- Existing compliant packages (llms, embeddings, vectorstores) - Provider patterns
- Existing test patterns (test_utils.go, advanced_test.go) - Testing standards

---

**Status**: Research complete, all technical decisions made, ready for Phase 1 design.
