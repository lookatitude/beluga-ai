# Implementation Plan: Multimodal Models Support

**Branch**: `009-multimodal` | **Date**: 2025-01-27 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/009-multimodal/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

This feature provides a unified interface for multimodal models (text+image/video/audio) that integrates with existing LLM and embedding providers. It enables agents to process and reason over non-text data, supports multimodal RAG workflows, real-time streaming, and extends agents with multimodal capabilities. The implementation follows framework design patterns with a global provider registry, comprehensive OTEL observability, and backward compatibility with text-only workflows.

## Technical Context

**Language/Version**: Go 1.21+ (matches existing framework)  
**Primary Dependencies**: 
- Existing Beluga AI packages (schema, embeddings, vectorstores, agents, orchestration, llms, voice, core, config, monitoring)
- OpenTelemetry (OTEL) for observability (already integrated)
- Provider SDKs: OpenAI SDK, Google AI SDK, Anthropic SDK, xAI SDK, open-source provider libraries
- Standard library: context, encoding/base64, encoding/json, net/http, sync, time

**Storage**: N/A (stateless package, integrates with existing vectorstores for multimodal RAG)  
**Testing**: 
- Standard Go testing (testing package)
- Framework test utilities (test_utils.go, advanced_test.go patterns)
- Integration tests for cross-package compatibility
- Benchmarks for performance-critical operations

**Target Platform**: Linux/macOS/Windows (Go cross-platform, matches existing framework)  
**Project Type**: Single Go package (pkg/multimodal/) following framework structure  
**Performance Goals**: 
- <500ms latency for voice processing (p95)
- <1s latency for video processing (p95)
- <200ms latency for image processing (p95)
- Support streaming with incremental results

**Constraints**: 
- Must maintain 100% backward compatibility with text-only workflows
- Must follow framework package design patterns (iface/, config.go, metrics.go, errors.go, test_utils.go, advanced_test.go)
- Must integrate with existing provider registries (llms, embeddings)
- Must support provider capability detection and graceful fallbacks
- Must handle large multimodal files (streaming, chunking)

**Scale/Scope**: 
- Support 10+ multimodal providers (OpenAI, Google, Anthropic, xAI, Alibaba, Mistral, Microsoft, DeepSeek, Google Gemma, others)
- Handle multimodal inputs up to 100MB per content block
- Support concurrent multimodal processing (100+ concurrent requests)
- Integrate with existing framework packages without breaking changes

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Framework Design Patterns Compliance

✅ **Package Structure**: Will follow standard v2 structure:
- `pkg/multimodal/iface/` - Interfaces
- `pkg/multimodal/internal/` - Private implementations
- `pkg/multimodal/providers/` - Provider implementations (openai, google, anthropic, etc.)
- `pkg/multimodal/config.go` - Configuration with struct tags
- `pkg/multimodal/metrics.go` - OTEL metrics
- `pkg/multimodal/errors.go` - Custom error types
- `pkg/multimodal/test_utils.go` - Test utilities
- `pkg/multimodal/advanced_test.go` - Advanced test scenarios

✅ **Interface Design**: Will follow ISP with focused interfaces:
- `MultimodalModel` interface for core operations
- `MultimodalProvider` interface for provider-specific implementations
- Small, focused interfaces following "er" suffix pattern where appropriate

✅ **Provider Registry**: Will use global registry pattern matching existing packages:
- `GetRegistry()` function for global registry access
- `Register()` method for provider registration
- `Create()` method for provider instantiation
- Provider registration in `providers/*/init.go` files

✅ **OTEL Observability**: Will include comprehensive observability:
- OTEL metrics in `metrics.go` (counters, histograms for latency, throughput)
- OTEL tracing for all public methods
- Structured logging with OTEL context (trace IDs, span IDs)

✅ **Error Handling**: Will use custom error types:
- `MultimodalError` with Op, Err, Code fields
- Error codes for common failures (invalid_format, provider_error, etc.)
- Context cancellation support

✅ **Configuration**: Will use struct tags and functional options:
- Config struct with mapstructure, yaml, env, validate tags
- Functional options for runtime configuration
- Validation at creation time

✅ **Testing**: Will include comprehensive tests:
- Table-driven tests in `advanced_test.go`
- Mocks in `internal/mock/` or `test_utils.go`
- Benchmarks for performance-critical operations
- Integration tests for cross-package compatibility

### Backward Compatibility

✅ **No Breaking Changes**: 
- Multimodal is opt-in (existing text-only workflows continue to work)
- New package doesn't modify existing packages
- Integration points use existing interfaces (MultimodalEmbedder, Message types)

### Complexity Justification

No violations - follows established framework patterns and adds new package without modifying existing ones.

## Project Structure

### Documentation (this feature)

```text
specs/009-multimodal/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
pkg/multimodal/
├── iface/
│   ├── model.go         # MultimodalModel interface
│   ├── provider.go      # MultimodalProvider interface
│   └── content.go       # ContentBlock interface
├── internal/
│   ├── model.go         # BaseMultimodalModel implementation
│   ├── router.go        # Content routing logic
│   ├── normalizer.go    # Format normalization
│   └── mock/
│       └── mock_model.go # Mock implementations for testing
├── providers/
│   ├── openai/
│   │   ├── init.go      # Provider registration
│   │   ├── provider.go  # OpenAI multimodal implementation
│   │   └── config.go    # OpenAI-specific config
│   ├── google/
│   │   ├── init.go
│   │   ├── provider.go
│   │   └── config.go
│   ├── anthropic/
│   │   ├── init.go
│   │   ├── provider.go
│   │   └── config.go
│   ├── xai/
│   │   ├── init.go
│   │   ├── provider.go
│   │   └── config.go
│   ├── qwen/
│   │   ├── init.go
│   │   ├── provider.go
│   │   └── config.go
│   └── [other providers]/
├── config.go            # Configuration structs and validation
├── metrics.go           # OTEL metrics definitions
├── errors.go            # Custom error types and codes
├── registry.go          # Global provider registry
├── factory.go           # Factory functions
├── multimodal.go        # Main package API
├── test_utils.go        # Test utilities and mocks
├── advanced_test.go     # Advanced test scenarios
├── multimodal_test.go   # Unit tests
└── README.md            # Package documentation

tests/integration/multimodal/
├── rag_test.go          # Multimodal RAG integration tests
├── agent_test.go        # Multimodal agent integration tests
└── streaming_test.go    # Streaming integration tests
```

**Structure Decision**: Single Go package following framework v2 standards. Providers are in `providers/` subdirectory with standard init.go registration pattern. Internal implementation details are in `internal/`. Integration tests are in `tests/integration/multimodal/` following framework testing patterns.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations - follows established patterns.
